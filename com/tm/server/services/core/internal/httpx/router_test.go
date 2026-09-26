package httpx

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeDB giả lập pool: trả err, hoặc (block=true) chờ tới khi ctx hết hạn.
type fakeDB struct {
	err   error
	block bool
}

func (f fakeDB) Ping(ctx context.Context) error {
	if f.block {
		<-ctx.Done()
		return ctx.Err()
	}
	return f.err
}

const secret = "S3cr3tXYZ"

func TestRouter(t *testing.T) {
	// Lỗi ping giống pgx khi sai mật khẩu, cố tình nhét chuỗi bí mật để kiểm body không lộ.
	pingErr := errors.New("failed to connect: password authentication failed " + secret)

	tests := []struct {
		name       string
		db         fakeDB
		method     string
		path       string
		wantCode   int
		wantBody   string // chuỗi con phải có trong body; rỗng = bỏ qua
		wantCType  string // tiền tố Content-Type; rỗng = bỏ qua
		maxLatency time.Duration
	}{
		{name: "healthz 200", method: http.MethodGet, path: "/healthz", wantCode: 200, wantBody: `"ok"`, wantCType: "application/json"},
		{name: "healthz 200 kể cả khi DB lỗi", db: fakeDB{err: pingErr}, method: http.MethodGet, path: "/healthz", wantCode: 200},
		{name: "healthz 200 kể cả khi DB treo", db: fakeDB{block: true}, method: http.MethodGet, path: "/healthz", wantCode: 200, maxLatency: 100 * time.Millisecond},
		{name: "readyz 200 khi ping OK", method: http.MethodGet, path: "/readyz", wantCode: 200, wantBody: `"ok"`},
		{name: "readyz 503 khi ping lỗi", db: fakeDB{err: pingErr}, method: http.MethodGet, path: "/readyz", wantCode: 503, wantBody: `"unavailable"`},
		{name: "readyz 503 khi ping quá timeout", db: fakeDB{block: true}, method: http.MethodGet, path: "/readyz", wantCode: 503, maxLatency: time.Second},
		{name: "metrics 200 định dạng Prometheus", method: http.MethodGet, path: "/metrics", wantCode: 200, wantBody: "go_goroutines", wantCType: "text/plain"},
		{name: "route lạ 404", method: http.MethodGet, path: "/khong-co", wantCode: 404},
		{name: "route nghiệp vụ chưa có 404", method: http.MethodGet, path: "/internal/v1/trips", wantCode: 404},
		{name: "POST healthz 405", method: http.MethodPost, path: "/healthz", wantCode: 405},
		{name: "DELETE healthz 405", method: http.MethodDelete, path: "/healthz", wantCode: 405},
		{name: "POST readyz 405", method: http.MethodPost, path: "/readyz", wantCode: 405},
		{name: "POST metrics 405", method: http.MethodPost, path: "/metrics", wantCode: 405},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			log := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
			h := newRouter(log, tt.db, 50*time.Millisecond)

			start := time.Now()
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			elapsed := time.Since(start)

			body, _ := io.ReadAll(rec.Body)
			if rec.Code != tt.wantCode {
				t.Fatalf("code = %d, muốn %d (body %q)", rec.Code, tt.wantCode, body)
			}
			if tt.wantBody != "" && !strings.Contains(string(body), tt.wantBody) {
				t.Errorf("body %q thiếu %q", body, tt.wantBody)
			}
			if ct := rec.Header().Get("Content-Type"); tt.wantCType != "" && !strings.HasPrefix(ct, tt.wantCType) {
				t.Errorf("Content-Type = %q, muốn tiền tố %q", ct, tt.wantCType)
			}
			if tt.maxLatency > 0 && elapsed > tt.maxLatency {
				t.Errorf("mất %v, muốn ≤ %v", elapsed, tt.maxLatency)
			}
			if strings.Contains(string(body), secret) {
				t.Errorf("body lộ bí mật: %q", body)
			}
		})
	}
}

// Lỗi ping được log mức WARN dạng JSON; body 503 không mang chi tiết lỗi.
func TestReadyzLogsPingError(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))
	h := NewRouter(log, fakeDB{err: errors.New("connection refused")})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code = %d, muốn 503", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "connection refused") {
		t.Errorf("body lộ lỗi nội bộ: %q", rec.Body.String())
	}
	out := logs.String()
	for _, s := range []string{`"level":"WARN"`, "connection refused"} {
		if !strings.Contains(out, s) {
			t.Errorf("log %q thiếu %q", out, s)
		}
	}
}

// ReadyTimeout phải ≤ 2 s để /readyz trả trong ≤ 3 s khi PG treo (A4).
func TestReadyTimeout(t *testing.T) {
	if ReadyTimeout <= 0 || ReadyTimeout > 2*time.Second {
		t.Fatalf("ReadyTimeout = %v, muốn trong (0, 2s]", ReadyTimeout)
	}
}
