package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// syncBuffer: bytes.Buffer an toàn khi run ghi còn test đọc (-race).
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

// logLines tách log JSON thành map; dòng không phải JSON object → test fail.
func logLines(t *testing.T, s string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(s), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("dòng log không phải JSON object: %q: %v", l, err)
		}
		for _, k := range []string{"time", "level", "msg"} {
			if _, ok := m[k]; !ok {
				t.Fatalf("dòng log thiếu key %q: %q", k, l)
			}
		}
		out = append(out, m)
	}
	return out
}

func steps(lines []map[string]any) []string {
	var s []string
	for _, m := range lines {
		if v, ok := m["shutdown_step"].(string); ok {
			s = append(s, v)
		}
	}
	return s
}

func TestRun(t *testing.T) {
	errStop := errors.New("dừng theo yêu cầu")
	tests := []struct {
		name       string
		signal     os.Signal // gửi qua sigs; nil thì huỷ ctx
		wantReason string
	}{
		{name: "SIGTERM", signal: syscall.SIGTERM, wantReason: "terminated"},
		{name: "SIGINT", signal: os.Interrupt, wantReason: "interrupt"},
		{name: "ctx huỷ", wantReason: errStop.Error()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf syncBuffer
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			sigs := make(chan os.Signal, 1)
			done := make(chan int, 1)
			go func() { done <- run(ctx, newLogger(&buf), sigs) }()

			// Chưa có tín hiệu → run phải đang chờ, đã log khởi động.
			waitFor(t, func() bool { return strings.Contains(buf.String(), "khởi động") })
			select {
			case code := <-done:
				t.Fatalf("run trả về %d trước khi có tín hiệu", code)
			case <-time.After(50 * time.Millisecond):
			}

			if tc.signal != nil {
				sigs <- tc.signal
			} else {
				cancel(errStop)
			}
			var code int
			select {
			case code = <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("run không trả về sau tín hiệu")
			}
			if code != 0 {
				t.Errorf("exit code = %d, muốn 0", code)
			}

			lines := logLines(t, buf.String())
			if len(lines) != 3 {
				t.Fatalf("số dòng log = %d, muốn 3: %s", len(lines), buf.String())
			}
			if _, ok := lines[0]["shutdown_step"]; ok {
				t.Errorf("dòng đầu phải là khởi động, không có shutdown_step: %v", lines[0])
			}
			if got := strings.Join(steps(lines), " "); got != "signal done" {
				t.Errorf("shutdown_step = %q, muốn %q", got, "signal done")
			}
			if got := lines[1]["signal"]; got != tc.wantReason {
				t.Errorf("signal = %v, muốn %q", got, tc.wantReason)
			}
			if got := lines[2]["exit_code"]; got != float64(0) {
				t.Errorf("exit_code = %v, muốn 0", got)
			}
			for _, m := range lines {
				if m["level"] != "INFO" {
					t.Errorf("level = %v, muốn INFO: %v", m["level"], m)
				}
			}
		})
	}
}

// TestRunRealSignal: SIGTERM thật gửi tới chính process qua signal.Notify như
// main → run dừng sạch (không bị giết).
func TestRunRealSignal(t *testing.T) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)
	defer signal.Stop(sigs)

	var buf syncBuffer
	done := make(chan int, 1)
	go func() { done <- run(context.Background(), newLogger(&buf), sigs) }()
	waitFor(t, func() bool { return strings.Contains(buf.String(), "khởi động") })

	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("kill: %v", err)
	}
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("exit code = %d, muốn 0", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run không trả về sau SIGTERM")
	}
	if got := strings.Join(steps(logLines(t, buf.String())), " "); got != "signal done" {
		t.Errorf("shutdown_step = %q", got)
	}
}

func TestNewLoggerLevel(t *testing.T) {
	tests := []struct {
		level slog.Level
		want  bool
	}{
		{level: slog.LevelDebug, want: false},
		{level: slog.LevelInfo, want: true},
		{level: slog.LevelWarn, want: true},
		{level: slog.LevelError, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.level.String(), func(t *testing.T) {
			var buf syncBuffer
			newLogger(&buf).Log(context.Background(), tc.level, "xin-chao")
			if got := strings.Contains(buf.String(), "xin-chao"); got != tc.want {
				t.Errorf("có dòng log = %v, muốn %v", got, tc.want)
			}
			if tc.want {
				lines := logLines(t, buf.String())
				if lines[0]["level"] != tc.level.String() {
					t.Errorf("level = %v, muốn %s", lines[0]["level"], tc.level)
				}
			}
		})
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("hết giờ chờ điều kiện")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
