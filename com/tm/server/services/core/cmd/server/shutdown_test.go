package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// ---- fake / bọc dùng chung ----

// events ghi thứ tự sự kiện từ nhiều goroutine.
type events struct {
	mu   sync.Mutex
	list []string
}

func (e *events) add(s string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.list = append(e.list, s)
}

func (e *events) snapshot() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.list)
}

// recServer bọc *http.Server thật: báo địa chỉ listener, ghi lúc
// Shutdown/Close trả về.
type recServer struct {
	*http.Server
	ev   *events
	addr chan string
}

func (s *recServer) Serve(ln net.Listener) error {
	s.addr <- ln.Addr().String()
	return s.Server.Serve(ln)
}

func (s *recServer) Shutdown(ctx context.Context) error {
	err := s.Server.Shutdown(ctx)
	s.ev.add("shutdown_returned")
	return err
}

func (s *recServer) Close() error {
	err := s.Server.Close()
	s.ev.add("close_returned")
	return err
}

// fakePool thay pgxpool: đếm số lần Close, tuỳ chọn treo tới khi block đóng.
type fakePool struct {
	ev    *events
	calls atomic.Int32
	block chan struct{} // nil = trả ngay
}

func (p *fakePool) Close() {
	p.calls.Add(1)
	p.ev.add("pool_close")
	if p.block != nil {
		<-p.block
	}
}

// syncBuffer: log được ghi từ nhiều goroutine, đọc từ test.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// logRecords parse log JSON; mọi dòng phải là JSON.
func logRecords(t *testing.T, b *syncBuffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(b.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("dòng log không phải JSON: %q", l)
		}
		out = append(out, m)
	}
	return out
}

// steps trả chuỗi shutdown_step và bản ghi theo từng mốc.
func steps(t *testing.T, b *syncBuffer) ([]string, map[string]map[string]any) {
	t.Helper()
	var list []string
	byStep := map[string]map[string]any{}
	for _, r := range logRecords(t, b) {
		if s, ok := r["shutdown_step"].(string); ok {
			list = append(list, s)
			byStep[s] = r
		}
	}
	return list, byStep
}

// checkNoLeak so số goroutine khi test kết thúc với lúc bắt đầu, có chờ ổn
// định. Đăng ký đầu tiên để t.Cleanup chạy nó sau cùng (LIFO), sau khi các
// cleanup khác đã mở khoá handler/pool treo.
func checkNoLeak(t *testing.T) {
	t.Helper()
	before := runtime.NumGoroutine()
	t.Cleanup(func() {
		deadline := time.Now().Add(5 * time.Second)
		for {
			now := runtime.NumGoroutine()
			if now <= before {
				return
			}
			if time.Now().After(deadline) {
				buf := make([]byte, 1<<16)
				n := runtime.Stack(buf, true)
				t.Errorf("rò goroutine: trước %d, sau %d\n%s", before, now, buf[:n])
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	})
}

// harness chạy run trong goroutine với server thật bọc recServer.
type harness struct {
	ev     *events
	pool   *fakePool
	logs   *syncBuffer
	sigs   chan os.Signal
	cancel context.CancelFunc
	errc   chan error
	addr   string
	client *http.Client
}

func startRun(t *testing.T, handler http.Handler, timeout time.Duration, pool *fakePool, sigs chan os.Signal) *harness {
	t.Helper()
	ev := pool.ev
	if sigs == nil {
		sigs = make(chan os.Signal, 2)
	}
	h := &harness{
		ev: ev, pool: pool, logs: &syncBuffer{}, sigs: sigs, errc: make(chan error, 1),
		client: &http.Client{Transport: &http.Transport{DisableKeepAlives: true}},
	}
	t.Cleanup(h.client.CloseIdleConnections)
	srv := &recServer{
		Server: &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second},
		ev:     ev, addr: make(chan string, 1),
	}
	log := slog.New(slog.NewJSONHandler(h.logs, nil))
	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	t.Cleanup(cancel)
	go func() { h.errc <- run(ctx, log, srv, pool, "127.0.0.1:0", timeout, sigs) }()
	select {
	case h.addr = <-srv.addr:
	case err := <-h.errc:
		t.Fatalf("run trả về trước khi phục vụ: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("run không mở cổng trong 3s")
	}
	return h
}

// wait chờ run trả về, trả thời gian từ from và lỗi của run.
func (h *harness) wait(t *testing.T, from time.Time, limit time.Duration) (time.Duration, error) {
	t.Helper()
	select {
	case err := <-h.errc:
		return time.Since(from), err
	case <-time.After(limit):
		t.Fatalf("run chưa trả về sau %v", limit)
		return 0, nil
	}
}

// get gửi GET trong goroutine, kết quả (status hoặc -1 khi lỗi) về channel.
func (h *harness) get(path string) <-chan int {
	out := make(chan int, 1)
	go func() {
		resp, err := h.client.Get("http://" + h.addr + path)
		if err != nil {
			out <- -1
			return
		}
		_ = resp.Body.Close()
		out <- resp.StatusCode
	}()
	return out
}

// ---- U1, U2, U4, U6, U7, U10: bảng kịch bản dừng ----

func TestRunShutdown(t *testing.T) {
	type handlerKind int
	const (
		noRequest    handlerKind = iota
		sleepHandler             // ngủ sleep rồi trả 200
		blockHandler             // treo tới cleanup (chậm hơn mọi timeout)
	)
	tests := []struct {
		name         string
		timeout      time.Duration
		handler      handlerKind
		sleep        time.Duration
		triggerAfter time.Duration // tính từ lúc handler bắt đầu
		byCtx        bool          // trigger bằng huỷ ctx thay vì tín hiệu
		poolBlocks   bool
		wantSteps    []string
		wantErrIs    []error // nil = trả nil
		wantStatus   int     // 0 = không có request; -1 = client lỗi (không 200)
		minDur       time.Duration
		maxDur       time.Duration // tính từ trigger
	}{
		{
			name:      "U1 rảnh: SIGTERM → dừng sạch ngay",
			timeout:   2 * time.Second,
			wantSteps: []string{stepSignal, stepHTTPStopped, stepPoolClosed, stepDone},
			maxDur:    time.Second,
		},
		{
			name:      "U1 rảnh: huỷ ctx cũng dừng",
			timeout:   2 * time.Second,
			byCtx:     true,
			wantSteps: []string{stepSignal, stepHTTPStopped, stepPoolClosed, stepDone},
			maxDur:    time.Second,
		},
		{
			name:         "U2 request đang chạy được trả 200 trước khi run trả về",
			timeout:      5 * time.Second,
			handler:      sleepHandler,
			sleep:        800 * time.Millisecond,
			triggerAfter: 200 * time.Millisecond,
			wantSteps:    []string{stepSignal, stepHTTPStopped, stepPoolClosed, stepDone},
			wantStatus:   http.StatusOK,
			minDur:       400 * time.Millisecond,
			maxDur:       2 * time.Second,
		},
		{
			name:         "U4 handler chậm hơn timeout → lỗi DeadlineExceeded, không chờ handler",
			timeout:      500 * time.Millisecond,
			handler:      blockHandler,
			triggerAfter: 50 * time.Millisecond,
			wantSteps:    []string{stepSignal, stepTimeout, stepPoolClosed, stepDone},
			wantErrIs:    []error{context.DeadlineExceeded},
			wantStatus:   -1,
			minDur:       450 * time.Millisecond,
			maxDur:       500*time.Millisecond + time.Second,
		},
		{
			name:       "U10 pool treo, HTTP xong trong hạn → chờ phần hạn còn lại",
			timeout:    time.Second,
			poolBlocks: true,
			wantSteps:  []string{stepSignal, stepHTTPStopped, stepPoolCloseTimeout, stepDone},
			wantErrIs:  []error{errPoolCloseTimeout},
			minDur:     900 * time.Millisecond,
			maxDur:     time.Second + 500*time.Millisecond,
		},
		{
			name:         "U10 HTTP hết hạn rồi pool treo → vẫn chờ tối thiểu 1s",
			timeout:      300 * time.Millisecond,
			handler:      blockHandler,
			triggerAfter: 50 * time.Millisecond,
			poolBlocks:   true,
			wantSteps:    []string{stepSignal, stepTimeout, stepPoolCloseTimeout, stepDone},
			wantErrIs:    []error{context.DeadlineExceeded, errPoolCloseTimeout},
			wantStatus:   -1,
			minDur:       300*time.Millisecond + minPoolCloseWait - 100*time.Millisecond,
			maxDur:       300*time.Millisecond + minPoolCloseWait + 500*time.Millisecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkNoLeak(t)
			ev := &events{}
			pool := &fakePool{ev: ev}
			if tt.poolBlocks {
				pool.block = make(chan struct{})
				t.Cleanup(func() { close(pool.block) })
			}
			release := make(chan struct{})
			t.Cleanup(func() { close(release) })
			started := make(chan struct{}, 1)
			mux := http.NewServeMux()
			mux.HandleFunc("/work", func(w http.ResponseWriter, r *http.Request) {
				started <- struct{}{}
				switch tt.handler {
				case sleepHandler:
					time.Sleep(tt.sleep)
				case blockHandler:
					<-release
				}
				ev.add("handler_done")
				w.WriteHeader(http.StatusOK)
			})
			h := startRun(t, mux, tt.timeout, pool, nil)

			var result <-chan int
			if tt.handler != noRequest {
				result = h.get("/work")
				select {
				case <-started:
				case <-time.After(2 * time.Second):
					t.Fatal("handler không bắt đầu")
				}
				time.Sleep(tt.triggerAfter)
			}
			trigger := time.Now()
			if tt.byCtx {
				h.cancel()
			} else {
				h.sigs <- syscall.SIGTERM
			}
			dur, err := h.wait(t, trigger, tt.maxDur+2*time.Second)

			// Lỗi trả về và exit code.
			if tt.wantErrIs == nil {
				if err != nil {
					t.Fatalf("run() = %v, muốn nil", err)
				}
			}
			for _, target := range tt.wantErrIs {
				if !errors.Is(err, target) {
					t.Errorf("run() = %v, muốn errors.Is %v", err, target)
				}
			}
			wantCode := 0
			if tt.wantErrIs != nil {
				wantCode = 1
			}
			if got := exitCode(err); got != wantCode {
				t.Errorf("exitCode = %d, muốn %d", got, wantCode)
			}

			// Thời gian: không chờ quá hạn, không trả về quá sớm.
			if dur < tt.minDur || dur > tt.maxDur {
				t.Errorf("run trả về sau %v, muốn trong [%v, %v]", dur, tt.minDur, tt.maxDur)
			}

			// Request đang chạy.
			if tt.handler != noRequest {
				select {
				case got := <-result:
					if tt.wantStatus == http.StatusOK && got != http.StatusOK {
						t.Errorf("status = %d, muốn 200", got)
					}
					if tt.wantStatus == -1 && got == http.StatusOK {
						t.Error("request bị cắt vẫn nhận 200")
					}
				case <-time.After(2 * time.Second):
					t.Error("client chưa nhận kết quả")
				}
			}

			// Log mốc shutdown.
			got, byStep := steps(t, h.logs)
			if !slices.Equal(got, tt.wantSteps) {
				t.Errorf("shutdown_step = %v, muốn %v", got, tt.wantSteps)
			}
			for step, rec := range byStep {
				wantLevel := "INFO"
				if step == stepTimeout || step == stepPoolCloseTimeout {
					wantLevel = "WARN"
				}
				if rec["level"] != wantLevel {
					t.Errorf("mốc %s level = %v, muốn %s", step, rec["level"], wantLevel)
				}
			}
			if sig := byStep[stepSignal]; sig != nil {
				wantSig := "terminated"
				if tt.byCtx {
					wantSig = context.Canceled.Error()
				}
				if sig["signal"] != wantSig || sig["timeout"] != tt.timeout.String() {
					t.Errorf("mốc signal = %v, muốn signal=%s timeout=%s", sig, wantSig, tt.timeout)
				}
			}
			if done := byStep[stepDone]; done == nil || done["exit_code"] != float64(wantCode) {
				t.Errorf("mốc done = %v, muốn exit_code=%d", done, wantCode)
			}

			// U6: pool đóng đúng 1 lần, sau khi server dừng và handler xong.
			if n := pool.calls.Load(); n != 1 {
				t.Errorf("pool.Close gọi %d lần, muốn 1", n)
			}
			order := ev.snapshot()
			poolAt := slices.Index(order, "pool_close")
			stopAt := slices.Index(order, "shutdown_returned")
			if closeAt := slices.Index(order, "close_returned"); closeAt >= 0 {
				stopAt = max(stopAt, closeAt)
			}
			if stopAt < 0 || poolAt < stopAt {
				t.Errorf("thứ tự sự kiện %v: pool_close phải sau Shutdown/Close", order)
			}
			if tt.handler == sleepHandler {
				if doneAt := slices.Index(order, "handler_done"); doneAt < 0 || doneAt > poolAt {
					t.Errorf("thứ tự sự kiện %v: handler_done phải trước pool_close", order)
				}
			}
		})
	}
}

// P0-AT08 nguyên văn: handler ngủ 3s, SIGTERM thật gửi tới chính process sau
// 1s, qua cùng notifyShutdown + run mà main dùng → 200, trả nil (exit 0).
func TestRunAT08InFlightSleep3sRealSIGTERM(t *testing.T) {
	// signal.Notify lần đầu khởi động goroutine nhận tín hiệu của runtime,
	// sống suốt process: đo goroutine sau khi đã đăng ký.
	sigs := notifyShutdown()
	t.Cleanup(func() { signal.Stop(sigs) })
	checkNoLeak(t)

	ev := &events{}
	pool := &fakePool{ev: ev}
	started := make(chan struct{}, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/sleep", func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		time.Sleep(3 * time.Second)
		ev.add("handler_done")
		_, _ = fmt.Fprint(w, "ok")
	})
	h := startRun(t, mux, 10*time.Second, pool, sigs)
	result := h.get("/sleep")
	<-started
	time.Sleep(1 * time.Second)

	trigger := time.Now()
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("gửi SIGTERM: %v", err)
	}
	dur, err := h.wait(t, trigger, 6*time.Second)
	if err != nil {
		t.Fatalf("run() = %v, muốn nil (exit 0)", err)
	}
	if got := <-result; got != http.StatusOK {
		t.Errorf("status = %d, muốn 200", got)
	}
	t.Logf("SIGTERM → run trả về: %v", dur)
	if dur < 1500*time.Millisecond || dur >= 3*time.Second {
		t.Errorf("run trả về sau %v tính từ SIGTERM, muốn ≈2s ([1.5s, 3s))", dur)
	}
	got, byStep := steps(t, h.logs)
	if want := []string{stepSignal, stepHTTPStopped, stepPoolClosed, stepDone}; !slices.Equal(got, want) {
		t.Errorf("shutdown_step = %v, muốn %v", got, want)
	}
	if byStep[stepSignal]["signal"] != "terminated" {
		t.Errorf("mốc signal = %v, muốn signal=terminated", byStep[stepSignal])
	}
	if order := ev.snapshot(); slices.Index(order, "handler_done") > slices.Index(order, "pool_close") {
		t.Errorf("thứ tự %v: handler phải xong trước khi đóng pool", order)
	}
}

// U3: sau khi bắt đầu shutdown, kết nối mới bị từ chối trong lúc request cũ
// vẫn đang chạy.
func TestRunRejectsNewConnections(t *testing.T) {
	checkNoLeak(t)
	ev := &events{}
	pool := &fakePool{ev: ev}
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/hold", func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-release
	})
	h := startRun(t, mux, 5*time.Second, pool, nil)
	result := h.get("/hold")
	<-started
	h.sigs <- syscall.SIGTERM

	// Listener đóng ngay khi Shutdown bắt đầu; thử dial tới khi bị từ chối.
	// Các lần dial trước đó có thể còn lọt vào listener đang đóng dở.
	var dialErr error
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", h.addr, 200*time.Millisecond)
		if err != nil {
			dialErr = err
			if errors.Is(err, syscall.ECONNREFUSED) {
				break
			}
			continue
		}
		_ = c.Close()
		time.Sleep(10 * time.Millisecond)
	}
	if !errors.Is(dialErr, syscall.ECONNREFUSED) {
		t.Errorf("dial sau shutdown = %v, muốn connection refused", dialErr)
	}
	select {
	case err := <-h.errc:
		t.Fatalf("run trả về (%v) khi request cũ chưa xong", err)
	default:
	}
	close(release)
	if _, err := h.wait(t, time.Now(), 3*time.Second); err != nil {
		t.Errorf("run() = %v, muốn nil", err)
	}
	if got := <-result; got != http.StatusOK {
		t.Errorf("request cũ status = %d, muốn 200", got)
	}
}

// U5: cổng bận → trả lỗi ngay, không chờ tín hiệu, pool vẫn được đóng.
func TestRunListenError(t *testing.T) {
	checkNoLeak(t)
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = busy.Close() })

	ev := &events{}
	pool := &fakePool{ev: ev}
	logs := &syncBuffer{}
	srv := &http.Server{Handler: http.NewServeMux(), ReadHeaderTimeout: time.Second}
	start := time.Now()
	errc := make(chan error, 1)
	go func() {
		errc <- run(context.Background(), slog.New(slog.NewJSONHandler(logs, nil)), srv, pool,
			busy.Addr().String(), 10*time.Second, make(chan os.Signal))
	}()
	select {
	case err = <-errc:
	case <-time.After(time.Second):
		t.Fatal("run không trả lỗi trong 1s khi cổng bận")
	}
	if err == nil || !strings.Contains(err.Error(), "address already in use") {
		t.Fatalf("run() = %v, muốn lỗi address already in use", err)
	}
	if exitCode(err) != 1 {
		t.Errorf("exitCode = %d, muốn 1", exitCode(err))
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("mất %v, muốn < 1s", d)
	}
	if n := pool.calls.Load(); n != 1 {
		t.Errorf("pool.Close gọi %d lần, muốn 1", n)
	}
	got, byStep := steps(t, logs)
	if want := []string{stepPoolClosed, stepDone}; !slices.Equal(got, want) {
		t.Errorf("shutdown_step = %v, muốn %v", got, want)
	}
	if byStep[stepDone]["exit_code"] != float64(1) {
		t.Errorf("mốc done = %v, muốn exit_code=1", byStep[stepDone])
	}
	var errLine bool
	for _, r := range logRecords(t, logs) {
		if r["level"] == "ERROR" && strings.Contains(fmt.Sprint(r["err"]), "address already in use") {
			errLine = true
		}
	}
	if !errLine {
		t.Error("không có dòng ERROR chứa address already in use")
	}
}

// A8: tín hiệu thứ hai khi đang chờ request → thoát ngay, lỗi ↔ exit 1.
func TestRunSecondSignal(t *testing.T) {
	tests := []struct {
		name       string
		poolBlocks bool // true: tín hiệu 2 đến lúc đang chờ pool
	}{
		{name: "đang chờ request"},
		{name: "đang chờ pool đóng", poolBlocks: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkNoLeak(t)
			ev := &events{}
			pool := &fakePool{ev: ev}
			if tt.poolBlocks {
				pool.block = make(chan struct{})
				t.Cleanup(func() { close(pool.block) })
			}
			release := make(chan struct{})
			t.Cleanup(func() { close(release) })
			started := make(chan struct{}, 1)
			mux := http.NewServeMux()
			mux.HandleFunc("/hold", func(w http.ResponseWriter, r *http.Request) {
				started <- struct{}{}
				<-release
			})
			h := startRun(t, mux, 10*time.Second, pool, nil)
			if !tt.poolBlocks {
				result := h.get("/hold")
				<-started
				t.Cleanup(func() { <-result })
			}
			h.sigs <- syscall.SIGTERM
			time.Sleep(300 * time.Millisecond)
			select {
			case err := <-h.errc:
				t.Fatalf("run trả về (%v) trước tín hiệu thứ hai", err)
			default:
			}
			second := time.Now()
			h.sigs <- syscall.SIGINT
			dur, err := h.wait(t, second, 2*time.Second)
			if !errors.Is(err, errSecondSignal) || exitCode(err) != 1 {
				t.Errorf("run() = %v (exit %d), muốn errSecondSignal, exit 1", err, exitCode(err))
			}
			if dur > 500*time.Millisecond {
				t.Errorf("thoát sau %v từ tín hiệu 2, muốn gần như ngay", dur)
			}
			if _, byStep := steps(t, h.logs); byStep[stepDone]["exit_code"] != float64(1) {
				t.Errorf("mốc done = %v, muốn exit_code=1", byStep[stepDone])
			}
		})
	}
}

// U9: ánh xạ lỗi → exit code.
func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"dừng sạch", nil, 0},
		{"hết timeout HTTP", fmt.Errorf("http shutdown: %w", context.DeadlineExceeded), 1},
		{"đóng pool quá hạn", errPoolCloseTimeout, 1},
		{"timeout + pool quá hạn", errors.Join(context.DeadlineExceeded, errPoolCloseTimeout), 1},
		{"lỗi listen", fmt.Errorf("listen: %w", syscall.EADDRINUSE), 1},
		{"tín hiệu thứ hai", errSecondSignal, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exitCode(tt.err); got != tt.want {
				t.Errorf("exitCode(%v) = %d, muốn %d", tt.err, got, tt.want)
			}
		})
	}
}
