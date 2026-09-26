package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// noFlush là telemetry shutdown no-op cho các test P0-T09 (chỉ đổi call site).
func noFlush(context.Context) error { return nil }

// fakeFlush thay otelx.ShutdownFunc: ghi số lần gọi, ctx có deadline không,
// tuỳ chọn ngủ (sleep) hoặc treo bỏ qua ctx tới khi release đóng.
type fakeFlush struct {
	ev          *events
	calls       atomic.Int32
	hadDeadline atomic.Bool
	sleep       time.Duration
	release     chan struct{} // khác nil = treo tới khi đóng, bỏ qua ctx
	err         error
}

func (f *fakeFlush) flush(ctx context.Context) error {
	f.calls.Add(1)
	_, ok := ctx.Deadline()
	f.hadDeadline.Store(ok)
	f.ev.add("telemetry_flush")
	if f.release != nil {
		<-f.release
	}
	time.Sleep(f.sleep)
	return f.err
}

// sleepPool: Close mất đúng d (0 = trả ngay); block khác nil = treo tới khi đóng.
type sleepPool struct {
	ev    *events
	calls atomic.Int32
	d     time.Duration
	block chan struct{}
}

func (p *sleepPool) Close() {
	p.calls.Add(1)
	p.ev.add("pool_close")
	if p.block != nil {
		<-p.block
	}
	time.Sleep(p.d)
}

// startRunFlush giống startRun nhưng truyền telemetry shutdown và pool tuỳ ý.
func startRunFlush(t *testing.T, handler http.Handler, timeout time.Duration, ev *events,
	pool poolCloser, flush func(context.Context) error,
) *harness {
	t.Helper()
	h := &harness{
		ev: ev, logs: &syncBuffer{}, sigs: make(chan os.Signal, 2), errc: make(chan error, 1),
		client: &http.Client{Transport: &http.Transport{DisableKeepAlives: true}},
	}
	t.Cleanup(h.client.CloseIdleConnections)
	srv := &recServer{
		Server: &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second},
		ev:     ev, addr: make(chan string, 1),
	}
	log := slog.New(slog.NewJSONHandler(h.logs, nil))
	go func() { h.errc <- run(context.Background(), log, srv, pool, flush, "127.0.0.1:0", timeout, h.sigs) }()
	select {
	case h.addr = <-srv.addr:
	case err := <-h.errc:
		t.Fatalf("run trả về trước khi phục vụ: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("run không mở cổng trong 3s")
	}
	return h
}

// flushRecords trả các bản ghi log có key telemetry_flush và vị trí của
// chúng cùng vị trí dòng done trong log.
func flushRecords(t *testing.T, b *syncBuffer) (recs []map[string]any, flushAt []int, doneAt int) {
	t.Helper()
	doneAt = -1
	for i, r := range logRecords(t, b) {
		if _, ok := r["telemetry_flush"]; ok {
			recs = append(recs, r)
			flushAt = append(flushAt, i)
		}
		if r["shutdown_step"] == stepDone {
			doneAt = i
		}
	}
	return recs, flushAt, doneAt
}

// checkFlushLine: đúng 1 dòng telemetry_flush = want, đúng mức, không có
// shutdown_step/latency_ms, đứng trước dòng done.
func checkFlushLine(t *testing.T, b *syncBuffer, want string) {
	t.Helper()
	recs, at, doneAt := flushRecords(t, b)
	if len(recs) != 1 {
		t.Fatalf("có %d dòng telemetry_flush, muốn 1: %v", len(recs), recs)
	}
	r := recs[0]
	wantLevel := "INFO"
	if want == flushFailed {
		wantLevel = "WARN"
		if e, _ := r["err"].(string); e == "" {
			t.Errorf("dòng telemetry_flush=failed thiếu err: %v", r)
		}
	}
	if r["telemetry_flush"] != want || r["level"] != wantLevel {
		t.Errorf("dòng flush = %v, muốn telemetry_flush=%s level=%s", r, want, wantLevel)
	}
	for _, k := range []string{"shutdown_step", "latency_ms"} {
		if _, ok := r[k]; ok {
			t.Errorf("dòng flush có key %s: %v", k, r)
		}
	}
	if doneAt < 0 || at[0] > doneAt {
		t.Errorf("dòng flush (vị trí %d) phải trước dòng done (vị trí %d)", at[0], doneAt)
	}
}

// ---- U9, U10, U11: flush telemetry song song với đóng pool ----

func TestRunTelemetryFlush(t *testing.T) {
	errExport := errors.New("export: connection refused")
	tests := []struct {
		name         string
		timeout      time.Duration
		blockHandler bool // request treo → HTTP quá hạn
		flushSleep   time.Duration
		flushHang    bool // treo, bỏ qua ctx
		flushErr     error
		poolSleep    time.Duration
		poolHang     bool
		wantSteps    []string
		wantFlush    string
		wantErrIs    []error
		minDur       time.Duration
		maxDur       time.Duration // tính từ tín hiệu
	}{
		{
			name:      "U9 dừng sạch → flush ok 1 lần, chuỗi mốc như P0-T09",
			timeout:   2 * time.Second,
			wantSteps: []string{stepSignal, stepHTTPStopped, stepPoolClosed, stepDone},
			wantFlush: flushOK,
			maxDur:    time.Second,
		},
		{
			name:      "U10 flush trả lỗi → failed WARN, exit vẫn 0",
			timeout:   2 * time.Second,
			flushErr:  errExport,
			wantSteps: []string{stepSignal, stepHTTPStopped, stepPoolClosed, stepDone},
			wantFlush: flushFailed,
			maxDur:    time.Second,
		},
		{
			name:         "U10 HTTP quá hạn + flush lỗi → chuỗi timeout, exit 1 như P0-T09",
			timeout:      300 * time.Millisecond,
			blockHandler: true,
			flushErr:     errExport,
			wantSteps:    []string{stepSignal, stepTimeout, stepPoolClosed, stepDone},
			wantFlush:    flushFailed,
			wantErrIs:    []error{context.DeadlineExceeded},
			minDur:       250 * time.Millisecond,
			maxDur:       300*time.Millisecond + 500*time.Millisecond,
		},
		{
			name:      "U11(i) flush treo, pool đóng ngay → hết hạn 1s rồi failed, exit 0",
			timeout:   300 * time.Millisecond,
			flushHang: true,
			wantSteps: []string{stepSignal, stepHTTPStopped, stepPoolClosed, stepDone},
			wantFlush: flushFailed,
			minDur:    minPoolCloseWait - 100*time.Millisecond,
			maxDur:    minPoolCloseWait + 500*time.Millisecond, // max(300ms − đã trôi, 1s) + 0.5s
		},
		{
			name:       "U11(ii) flush và pool cùng 500ms → song song, ~0.5s",
			timeout:    2 * time.Second,
			flushSleep: 500 * time.Millisecond,
			poolSleep:  500 * time.Millisecond,
			wantSteps:  []string{stepSignal, stepHTTPStopped, stepPoolClosed, stepDone},
			wantFlush:  flushOK,
			minDur:     450 * time.Millisecond,
			maxDur:     900 * time.Millisecond,
		},
		{
			name:      "U11(iii) cả hai treo → pool_close_timeout, flush failed, exit 1",
			timeout:   300 * time.Millisecond,
			flushHang: true,
			poolHang:  true,
			wantSteps: []string{stepSignal, stepHTTPStopped, stepPoolCloseTimeout, stepDone},
			wantFlush: flushFailed,
			wantErrIs: []error{errPoolCloseTimeout},
			minDur:    minPoolCloseWait - 100*time.Millisecond,
			maxDur:    minPoolCloseWait + 500*time.Millisecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkNoLeak(t)
			ev := &events{}
			pool := &sleepPool{ev: ev, d: tt.poolSleep}
			if tt.poolHang {
				pool.block = make(chan struct{})
				t.Cleanup(func() { close(pool.block) })
			}
			ff := &fakeFlush{ev: ev, sleep: tt.flushSleep, err: tt.flushErr}
			if tt.flushHang {
				ff.release = make(chan struct{})
				t.Cleanup(func() { close(ff.release) })
			}
			release := make(chan struct{})
			t.Cleanup(func() { close(release) })
			started := make(chan struct{}, 1)
			mux := http.NewServeMux()
			mux.HandleFunc("/hold", func(w http.ResponseWriter, r *http.Request) {
				started <- struct{}{}
				<-release
			})
			h := startRunFlush(t, mux, tt.timeout, ev, pool, ff.flush)
			if tt.blockHandler {
				result := h.get("/hold")
				<-started
				t.Cleanup(func() { <-result })
			}

			trigger := time.Now()
			h.sigs <- syscall.SIGTERM
			dur, err := h.wait(t, trigger, tt.maxDur+2*time.Second)

			if tt.wantErrIs == nil && err != nil {
				t.Fatalf("run() = %v, muốn nil (exit 0)", err)
			}
			for _, target := range tt.wantErrIs {
				if !errors.Is(err, target) {
					t.Errorf("run() = %v, muốn errors.Is %v", err, target)
				}
			}
			if errors.Is(err, errExport) {
				t.Errorf("run() = %v: lỗi flush không được gộp vào lỗi trả về", err)
			}
			if dur < tt.minDur || dur > tt.maxDur {
				t.Errorf("run trả về sau %v, muốn trong [%v, %v]", dur, tt.minDur, tt.maxDur)
			}

			got, byStep := steps(t, h.logs)
			if !slices.Equal(got, tt.wantSteps) {
				t.Errorf("shutdown_step = %v, muốn %v", got, tt.wantSteps)
			}
			wantCode := 0
			if tt.wantErrIs != nil {
				wantCode = 1
			}
			if done := byStep[stepDone]; done == nil || done["exit_code"] != float64(wantCode) || done["level"] != "INFO" {
				t.Errorf("mốc done = %v, muốn INFO exit_code=%d", done, wantCode)
			}
			checkFlushLine(t, h.logs, tt.wantFlush)

			// Gọi đúng 1 lần, ctx có deadline, sau khi srv.Shutdown/Close trả về.
			if n := ff.calls.Load(); n != 1 {
				t.Errorf("telemetry shutdown gọi %d lần, muốn 1", n)
			}
			if !ff.hadDeadline.Load() {
				t.Error("ctx truyền vào telemetry shutdown không có deadline")
			}
			order := ev.snapshot()
			flushAt := slices.Index(order, "telemetry_flush")
			stopAt := max(slices.Index(order, "shutdown_returned"), slices.Index(order, "close_returned"))
			if stopAt < 0 || flushAt < stopAt {
				t.Errorf("thứ tự sự kiện %v: telemetry_flush phải sau Shutdown/Close", order)
			}
			if n := pool.calls.Load(); n != 1 {
				t.Errorf("pool.Close gọi %d lần, muốn 1", n)
			}
		})
	}
}

// ---- U12: tín hiệu thứ hai → thoát ngay, không chờ telemetry ----

func TestRunTelemetrySecondSignal(t *testing.T) {
	tests := []struct {
		name      string
		inHTTP    bool // tín hiệu 2 khi request còn chạy (trong stopHTTP)
		poolHang  bool
		wantCalls int32
	}{
		{name: "trong stopHTTP → không gọi flush", inHTTP: true, wantCalls: 0},
		{name: "đang chờ pool và flush", poolHang: true, wantCalls: 1},
		{name: "pool đã đóng, đang chờ flush", wantCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkNoLeak(t)
			ev := &events{}
			pool := &sleepPool{ev: ev}
			if tt.poolHang {
				pool.block = make(chan struct{})
				t.Cleanup(func() { close(pool.block) })
			}
			ff := &fakeFlush{ev: ev, release: make(chan struct{})}
			t.Cleanup(func() { close(ff.release) })
			release := make(chan struct{})
			t.Cleanup(func() { close(release) })
			started := make(chan struct{}, 1)
			mux := http.NewServeMux()
			mux.HandleFunc("/hold", func(w http.ResponseWriter, r *http.Request) {
				started <- struct{}{}
				<-release
			})
			h := startRunFlush(t, mux, 10*time.Second, ev, pool, ff.flush)
			if tt.inHTTP {
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
			if n := ff.calls.Load(); n != tt.wantCalls {
				t.Errorf("telemetry shutdown gọi %d lần, muốn %d", n, tt.wantCalls)
			}
			if _, byStep := steps(t, h.logs); byStep[stepDone]["exit_code"] != float64(1) {
				t.Errorf("mốc done = %v, muốn exit_code=1", byStep[stepDone])
			}
		})
	}
}

// ---- U13: lỗi listen → vẫn flush telemetry, chuỗi mốc như P0-T09 ----

func TestRunTelemetryListenError(t *testing.T) {
	checkNoLeak(t)
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = busy.Close() })

	ev := &events{}
	pool := &sleepPool{ev: ev}
	ff := &fakeFlush{ev: ev}
	logs := &syncBuffer{}
	srv := &http.Server{Handler: http.NewServeMux(), ReadHeaderTimeout: time.Second}
	errc := make(chan error, 1)
	go func() {
		errc <- run(context.Background(), slog.New(slog.NewJSONHandler(logs, nil)), srv, pool, ff.flush,
			busy.Addr().String(), 10*time.Second, make(chan os.Signal))
	}()
	select {
	case err = <-errc:
	case <-time.After(2 * time.Second):
		t.Fatal("run không trả lỗi trong 2s khi cổng bận")
	}
	if err == nil || !strings.Contains(err.Error(), "address already in use") || exitCode(err) != 1 {
		t.Fatalf("run() = %v, muốn lỗi address already in use, exit 1", err)
	}
	if n := ff.calls.Load(); n != 1 || !ff.hadDeadline.Load() {
		t.Errorf("telemetry shutdown gọi %d lần (deadline=%v), muốn 1 lần có deadline", n, ff.hadDeadline.Load())
	}
	got, _ := steps(t, logs)
	if want := []string{stepPoolClosed, stepDone}; !slices.Equal(got, want) {
		t.Errorf("shutdown_step = %v, muốn %v", got, want)
	}
	checkFlushLine(t, logs, flushOK)
}
