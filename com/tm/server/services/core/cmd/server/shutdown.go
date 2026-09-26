package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// httpServer là phần của *http.Server mà run cần. Interface đặt ở phía dùng
// để unit test bọc server thật, ghi lại thứ tự Shutdown/Close.
type httpServer interface {
	Serve(net.Listener) error
	Shutdown(context.Context) error
	Close() error
}

// poolCloser là phần của *pgxpool.Pool mà run cần: chỉ đóng.
type poolCloser interface {
	Close()
}

// minPoolCloseWait là thời gian tối thiểu chờ pool đóng, kể cả khi HTTP đã
// dùng hết hạn shutdown.
const minPoolCloseWait = 1 * time.Second

// Giá trị của key shutdown_step trong log, theo thứ tự xảy ra.
const (
	stepSignal           = "signal"
	stepHTTPStopped      = "http_stopped"
	stepTimeout          = "timeout"
	stepPoolClosed       = "pool_closed"
	stepPoolCloseTimeout = "pool_close_timeout"
	stepDone             = "done"
)

var (
	// errPoolCloseTimeout: pool.Close() chưa xong trong hạn (pgxpool có thể
	// treo tới ~15s khi đang huỷ một kết nối dở).
	errPoolCloseTimeout = errors.New("đóng pool PG quá hạn")
	// errSecondSignal: người vận hành gửi tín hiệu lần hai, thoát ngay.
	errSecondSignal = errors.New("nhận tín hiệu thứ hai trong lúc dừng")
)

// shutdownSignals là các tín hiệu kích hoạt graceful shutdown.
var shutdownSignals = []os.Signal{syscall.SIGTERM, os.Interrupt}

// notifyShutdown trả channel nhận SIGTERM/SIGINT. Buffer 2 để không lỡ tín
// hiệu thứ hai khi run đang bận.
func notifyShutdown() chan os.Signal {
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, shutdownSignals...)
	return sigs
}

// run mở cổng addr, phục vụ srv tới khi nhận tín hiệu từ sigs (hoặc ctx bị
// huỷ), rồi dừng theo thứ tự:
//
//  1. log mốc signal;
//  2. srv.Shutdown với hạn timeout: đóng listener (kết nối mới bị từ chối),
//     đóng kết nối rảnh, chờ request đang chạy;
//  3. hết hạn → srv.Close() cắt các kết nối còn lại, không chờ handler;
//  4. đóng pool (luôn sau bước 2/3), chờ tối đa phần hạn còn lại, ít nhất
//     minPoolCloseWait;
//  5. log mốc done kèm exit_code.
//
// Lỗi listen hoặc Serve lỗi trước khi có tín hiệu → vẫn đóng pool rồi trả
// lỗi. Tín hiệu thứ hai trong bước 2–4 → cắt server và trả errSecondSignal
// ngay. Trả nil khi dừng sạch; exitCode ánh xạ kết quả ra exit code.
func run(ctx context.Context, log *slog.Logger, srv httpServer, pool poolCloser,
	addr string, timeout time.Duration, sigs <-chan os.Signal,
) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("mở cổng HTTP thất bại", "addr", addr, "err", err)
		return finish(log, fmt.Errorf("listen: %w", err), closePool(log, pool, minPoolCloseWait, nil))
	}
	log.Info("core đang lắng nghe", "addr", ln.Addr().String())

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	var reason string
	select {
	case err := <-serveErr:
		// Serve chỉ trả về trước khi có tín hiệu khi lỗi (ví dụ Accept).
		log.Error("http server dừng bất thường", "err", err)
		return finish(log, fmt.Errorf("serve: %w", err), closePool(log, pool, minPoolCloseWait, nil))
	case s := <-sigs:
		reason = s.String()
	case <-ctx.Done():
		reason = context.Cause(ctx).Error()
	}
	start := time.Now()
	log.Info("bắt đầu graceful shutdown", "shutdown_step", stepSignal,
		"signal", reason, "timeout", timeout.String())

	httpErr := stopHTTP(log, srv, timeout, sigs)
	if errors.Is(httpErr, errSecondSignal) {
		return finish(log, httpErr)
	}
	wait := max(timeout-time.Since(start), minPoolCloseWait)
	return finish(log, httpErr, closePool(log, pool, wait, sigs))
}

// stopHTTP chạy bước 2–3: Shutdown có hạn, hết hạn thì Close.
func stopHTTP(log *slog.Logger, srv httpServer, timeout time.Duration, sigs <-chan os.Signal) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- srv.Shutdown(ctx) }()

	select {
	case err := <-done:
		if err == nil {
			log.Info("http server đã dừng", "shutdown_step", stepHTTPStopped)
			return nil
		}
		log.Warn("hết hạn chờ request đang chạy, đóng cưỡng bức",
			"shutdown_step", stepTimeout, "timeout", timeout.String(), "err", err)
		if cerr := srv.Close(); cerr != nil {
			log.Error("đóng http server lỗi", "err", cerr)
		}
		return fmt.Errorf("http shutdown: %w", err)
	case s := <-sigs:
		// Không chờ Shutdown trả về: goroutine của nó tự kết thúc sau Close.
		log.Warn("nhận tín hiệu thứ hai, thoát ngay", "signal", s.String())
		if cerr := srv.Close(); cerr != nil {
			log.Error("đóng http server lỗi", "err", cerr)
		}
		return errSecondSignal
	}
}

// closePool gọi pool.Close() trong goroutine và chờ tối đa wait. Quá hạn thì
// bỏ goroutine lại (chết theo process) và trả errPoolCloseTimeout.
func closePool(log *slog.Logger, pool poolCloser, wait time.Duration, sigs <-chan os.Signal) error {
	done := make(chan struct{})
	go func() {
		pool.Close()
		close(done)
	}()
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-done:
		log.Info("pool PG đã đóng", "shutdown_step", stepPoolClosed)
		return nil
	case <-timer.C:
		log.Warn("đóng pool PG quá hạn, bỏ qua", "shutdown_step", stepPoolCloseTimeout, "wait", wait.String())
		return errPoolCloseTimeout
	case s := <-sigs:
		log.Warn("nhận tín hiệu thứ hai, thoát ngay", "signal", s.String())
		return errSecondSignal
	}
}

// finish gom lỗi, log mốc done và trả lỗi gộp.
func finish(log *slog.Logger, errs ...error) error {
	err := errors.Join(errs...)
	log.Info("core đã dừng", "shutdown_step", stepDone, "exit_code", exitCode(err))
	return err
}

// exitCode: 0 khi dừng sạch, 1 cho mọi lỗi (timeout, pool quá hạn, listen,
// tín hiệu thứ hai) để orchestrator phân biệt lần dừng không sạch.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	return 1
}
