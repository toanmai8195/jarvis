package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[47][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// syncBuffer: bytes.Buffer an toàn khi httptest.Server ghi log từ goroutine khác.
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

// lines parse từng dòng log JSON.
func (b *syncBuffer) lines(t *testing.T) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(b.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("dòng log không phải JSON: %q (%v)", l, err)
		}
		out = append(out, m)
	}
	return out
}

// accessLines: các dòng access log (có key latency_ms).
func (b *syncBuffer) accessLines(t *testing.T) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, m := range b.lines(t) {
		if _, ok := m["latency_ms"]; ok {
			out = append(out, m)
		}
	}
	return out
}

// stack là router thật (NewRouter) cùng SDK OTel in-memory.
type stack struct {
	h      http.Handler
	logs   *syncBuffer
	log    *slog.Logger
	spans  *tracetest.SpanRecorder
	reader *sdkmetric.ManualReader
}

// testRoutes: route chỉ có trong test — route tham số, handler panic, 503.
func testRoutes(log *slog.Logger) func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/items/{id}", func(w http.ResponseWriter, r *http.Request) {
			log.InfoContext(r.Context(), "item handler")
			writeStatus(w, http.StatusOK, "ok")
		})
		r.Get("/panic", panicWith("boom"))
	}
}

func newStack(t *testing.T, db pinger, extra ...Option) *stack {
	t.Helper()
	logs := &syncBuffer{}
	lg := slog.New(NewLogHandler(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	reader := sdkmetric.NewManualReader()
	// SDK in-memory (SpanRecorder đồng bộ, ManualReader chỉ đọc khi Collect):
	// không có goroutine nền nên không cần tắt provider sau test.
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	if db == nil {
		db = fakeDB{}
	}
	opts := append([]Option{
		WithTracerProvider(tp),
		WithMeterProvider(mp),
		WithPropagator(propagation.TraceContext{}),
		withRoutes(testRoutes(lg)),
	}, extra...)
	return &stack{h: NewRouter(lg, db, opts...), logs: logs, log: lg, spans: sr, reader: reader}
}

func (s *stack) do(method, target string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	s.h.ServeHTTP(rec, req)
	return rec
}

func panicWith(v any) http.HandlerFunc {
	return func(http.ResponseWriter, *http.Request) {
		panic(v)
	}
}

// ---------- Request ID ----------

func TestValidRequestID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"", false},
		{"a", true},
		{"0192f0c4-7d1e-7c3a-9b2f-1a2b3c4d5e6f", true},
		{"abc-123_x.y:z", true},
		{strings.Repeat("a", 128), true},
		{strings.Repeat("a", 129), false},
		{"has space", false},
		{"<script>", false},
		{`a"b`, false},
		{"mã-việt", false},
		{"a/b", false},
		{"a\nb", false},
	}
	for _, tt := range tests {
		if got := validRequestID(tt.id); got != tt.want {
			t.Errorf("validRequestID(%q) = %v, muốn %v", tt.id, got, tt.want)
		}
	}
}

func TestRequestID(t *testing.T) {
	tests := []struct {
		name    string
		header  *string // nil = không gửi header
		wantKep bool    // true = giữ nguyên giá trị client
	}{
		{name: "thiếu header", header: nil},
		{name: "UUID hợp lệ", header: ptr("0192f0c4-7d1e-7c3a-9b2f-1a2b3c4d5e6f"), wantKep: true},
		{name: "ký tự -_.:", header: ptr("abc-123_x.y:z"), wantKep: true},
		{name: "đúng 128 ký tự", header: ptr(strings.Repeat("a", 128)), wantKep: true},
		{name: "129 ký tự", header: ptr(strings.Repeat("a", 129))},
		{name: "khoảng trắng", header: ptr("has space")},
		{name: "ký tự <", header: ptr("<script>")},
		{name: "dấu nháy kép", header: ptr(`a"b`)},
		{name: "non-ASCII", header: ptr("mã-việt")},
		{name: "rỗng", header: ptr("")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fromCtx string
			h := requestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fromCtx = RequestIDFrom(r.Context())
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != nil {
				req.Header.Set(HeaderRequestID, *tt.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			got := rec.Header().Get(HeaderRequestID)
			if got == "" {
				t.Fatal("response thiếu X-Request-ID")
			}
			if fromCtx != got {
				t.Errorf("RequestIDFrom(ctx) = %q, header = %q", fromCtx, got)
			}
			if tt.wantKep {
				if got != *tt.header {
					t.Errorf("X-Request-ID = %q, muốn giữ %q", got, *tt.header)
				}
				return
			}
			if !uuidRe.MatchString(got) {
				t.Errorf("ID tự sinh %q không phải UUID", got)
			}
			if tt.header != nil && got == *tt.header {
				t.Errorf("giá trị không hợp lệ %q bị giữ nguyên", got)
			}
		})
	}
}

// Mỗi request một ID mới; header gửi bằng tên chữ thường vẫn được nhận.
func TestRequestIDUniqueAndCaseInsensitive(t *testing.T) {
	s := newStack(t, nil)
	a := s.do(http.MethodGet, "/healthz", nil).Header().Get(HeaderRequestID)
	b := s.do(http.MethodGet, "/healthz", nil).Header().Get(HeaderRequestID)
	if a == b {
		t.Errorf("hai request cùng ID %q", a)
	}
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("x-request-id", "lower-case-name") // net/http chuẩn hoá tên header
	rec := httptest.NewRecorder()
	s.h.ServeHTTP(rec, req)
	if got := rec.Header().Get("x-request-id"); got != "lower-case-name" {
		t.Errorf("X-Request-ID = %q, muốn lower-case-name", got)
	}
}

func TestRequestIDFromEmptyContext(t *testing.T) {
	if got := RequestIDFrom(context.Background()); got != "" {
		t.Errorf("RequestIDFrom(Background) = %q, muốn rỗng", got)
	}
}

// Log *Context trong request có request_id; log không kèm ctx request thì không.
func TestRequestIDInContextLog(t *testing.T) {
	s := newStack(t, nil)
	s.log.Info("khởi động") // ngoài request
	rec := s.do(http.MethodGet, "/items/1", map[string]string{HeaderRequestID: "rid-log-1"})
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	var sawStart, sawHandler bool
	for _, m := range s.logs.lines(t) {
		switch m["msg"] {
		case "khởi động":
			sawStart = true
			if _, ok := m["request_id"]; ok {
				t.Errorf("log ngoài request có request_id: %v", m)
			}
		case "item handler":
			sawHandler = true
			if m["request_id"] != "rid-log-1" {
				t.Errorf("log trong handler request_id = %v, muốn rid-log-1", m["request_id"])
			}
		}
	}
	if !sawStart || !sawHandler {
		t.Fatalf("thiếu dòng log: %s", s.logs.String())
	}
}

// Bọc handler hai lần (main bọc, NewRouter bọc lại) không nhân đôi key; With()
// giữ contextHandler.
func TestRequestIDLogHandlerWrapOnce(t *testing.T) {
	var buf bytes.Buffer
	base := slog.NewJSONHandler(&buf, nil)
	lg := withContextLogging(slog.New(NewLogHandler(NewLogHandler(base)))).With("k", "v")
	lg.InfoContext(withRequestID(context.Background(), "rid-x"), "m")
	if n := strings.Count(buf.String(), `"request_id"`); n != 1 {
		t.Errorf("request_id xuất hiện %d lần: %s", n, buf.String())
	}
	if !strings.Contains(buf.String(), `"k":"v"`) {
		t.Errorf("mất attr của With: %s", buf.String())
	}
}

// ---------- Recover ----------

func TestRecoverPanic(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		inLogs string // chuỗi phải có trong key panic của log
	}{
		{name: "panic string", value: "boom", inLogs: "boom"},
		{name: "panic error", value: errors.New("lỗi nội bộ"), inLogs: "lỗi nội bộ"},
		{name: "panic chứa bí mật", value: fmt.Sprintf("db password=%s", "S3cr3tPANIC"), inLogs: "S3cr3tPANIC"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStack(t, nil, withRoutes(func(r chi.Router) { r.Get("/p", panicWith(tt.value)) }))
			rec := s.do(http.MethodGet, "/p", nil)

			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("code = %d, muốn 500", rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Errorf("Content-Type = %q", ct)
			}
			rid := rec.Header().Get(HeaderRequestID)
			if rid == "" {
				t.Error("response 500 thiếu X-Request-ID")
			}
			body := rec.Body.String()
			var eb errorBody
			if err := json.Unmarshal([]byte(body), &eb); err != nil {
				t.Fatalf("body không parse được: %q", body)
			}
			if eb.Error.Code != CodeInternal || eb.Error.Message == "" {
				t.Errorf("body = %+v, muốn code INTERNAL và message khác rỗng", eb)
			}
			for _, bad := range []string{"S3cr3tPANIC", "boom", "lỗi nội bộ", "goroutine", ".go:"} {
				if strings.Contains(body, bad) {
					t.Errorf("body lộ %q: %s", bad, body)
				}
			}

			var errLines []map[string]any
			for _, m := range s.logs.lines(t) {
				if m["level"] == "ERROR" {
					errLines = append(errLines, m)
				}
			}
			if len(errLines) != 1 {
				t.Fatalf("có %d dòng ERROR, muốn 1: %s", len(errLines), s.logs.String())
			}
			e := errLines[0]
			if e["request_id"] != rid {
				t.Errorf("log panic request_id = %v, header = %q", e["request_id"], rid)
			}
			if p, _ := e["panic"].(string); !strings.Contains(p, tt.inLogs) {
				t.Errorf("log panic = %v, muốn chứa %q", e["panic"], tt.inLogs)
			}
			st, _ := e["stack"].(string)
			if !strings.Contains(st, ".go:") || !strings.Contains(st, "panicWith") {
				t.Errorf("stack thiếu file .go hoặc tên handler: %.300s", st)
			}
		})
	}
}

// Sau request panic, server vẫn phục vụ request sau.
func TestRecoverServerSurvives(t *testing.T) {
	s := newStack(t, nil)
	srv := httptest.NewServer(s.h)
	defer srv.Close()

	for _, tc := range []struct {
		path string
		want int
	}{{"/panic", 500}, {"/healthz", 200}, {"/panic", 500}, {"/healthz", 200}} {
		resp, err := http.Get(srv.URL + tc.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tc.path, err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != tc.want {
			t.Errorf("GET %s = %d, muốn %d", tc.path, resp.StatusCode, tc.want)
		}
	}
}

// newCapturedServer: httptest.Server có ErrorLog ghi vào buffer để bắt log của net/http.
func newCapturedServer(t *testing.T, h http.Handler) (*httptest.Server, *syncBuffer) {
	t.Helper()
	errLog := &syncBuffer{}
	srv := httptest.NewUnstartedServer(h)
	srv.Config.ErrorLog = slog.NewLogLogger(slog.NewTextHandler(errLog, nil), slog.LevelError)
	srv.Start()
	t.Cleanup(srv.Close)
	return srv, errLog
}

func TestRecoverAbortHandler(t *testing.T) {
	s := newStack(t, nil, withRoutes(func(r chi.Router) { r.Get("/abort", panicWith(http.ErrAbortHandler)) }))
	srv, errLog := newCapturedServer(t, s.h)

	resp, err := http.Get(srv.URL + "/abort")
	if err == nil {
		_ = resp.Body.Close()
		t.Fatalf("muốn lỗi kết nối, nhận status %d", resp.StatusCode)
	}
	for _, m := range s.logs.lines(t) {
		if m["level"] == "ERROR" {
			t.Errorf("ErrAbortHandler không được log ERROR: %v", m)
		}
		if _, ok := m["stack"]; ok {
			t.Errorf("ErrAbortHandler không được log stack: %v", m)
		}
	}
	if strings.Contains(errLog.String(), "goroutine") {
		t.Errorf("net/http log stack: %s", errLog.String())
	}

	resp, err = http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("request sau abort lỗi: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("request sau abort = %d, muốn 200", resp.StatusCode)
	}
}

func TestRecoverAfterWrite(t *testing.T) {
	s := newStack(t, nil, withRoutes(func(r chi.Router) {
		r.Get("/half", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("partial"))
			panic("sau khi ghi")
		})
	}))
	srv, errLog := newCapturedServer(t, s.h)

	resp, err := http.Get(srv.URL + "/half")
	if err != nil {
		t.Fatalf("GET /half: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, muốn 200 (không bị ghi đè)", resp.StatusCode)
	}
	if !strings.HasPrefix(string(body), "partial") || strings.Contains(string(body), CodeInternal) {
		t.Errorf("body = %q, không được nối body lỗi", body)
	}
	if strings.Contains(errLog.String(), "superfluous") {
		t.Errorf("WriteHeader bị gọi lần hai: %s", errLog.String())
	}
	var nErr int
	for _, m := range s.logs.lines(t) {
		if m["level"] == "ERROR" && m["panic"] == "sau khi ghi" {
			nErr++
		}
	}
	if nErr != 1 {
		t.Errorf("có %d dòng ERROR panic, muốn 1", nErr)
	}
	acc := s.logs.accessLines(t)
	if len(acc) != 1 || acc[0]["status"] != float64(200) {
		t.Errorf("access log = %v, muốn 1 dòng status 200", acc)
	}
}

// ---------- Access log ----------

func TestAccessLog(t *testing.T) {
	secretHdr := map[string]string{
		"Authorization": "Bearer S3cr3tAUTH",
		"Cookie":        "sid=S3cr3tCOOKIE",
		"X-Api-Key":     "S3cr3tKEY",
		"User-Agent":    "test-agent",
	}
	tests := []struct {
		name      string
		db        pinger
		method    string
		target    string
		wantRoute string
		wantPath  string
		wantCode  int
		wantLevel string
	}{
		{name: "healthz 200", method: "GET", target: "/healthz", wantRoute: "/healthz", wantPath: "/healthz", wantCode: 200, wantLevel: "INFO"},
		{name: "route tham số", method: "GET", target: "/items/42", wantRoute: "/items/{id}", wantPath: "/items/42", wantCode: 200, wantLevel: "INFO"},
		{name: "query không vào log", method: "GET", target: "/healthz?token=S3cr3tQUERY", wantRoute: "/healthz", wantPath: "/healthz", wantCode: 200, wantLevel: "INFO"},
		{name: "404 không có route", method: "GET", target: "/khong-co-123?x=1", wantRoute: "", wantPath: "/khong-co-123", wantCode: 404, wantLevel: "INFO"},
		{name: "405", method: "POST", target: "/readyz", wantRoute: "", wantPath: "/readyz", wantCode: 405, wantLevel: "INFO"},
		{name: "503 readyz", db: fakeDB{err: errors.New("down")}, method: "GET", target: "/readyz", wantRoute: "/readyz", wantPath: "/readyz", wantCode: 503, wantLevel: "WARN"},
		{name: "500 panic", method: "GET", target: "/panic", wantRoute: "/panic", wantPath: "/panic", wantCode: 500, wantLevel: "WARN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStack(t, tt.db)
			rec := s.do(tt.method, tt.target, secretHdr)
			if rec.Code != tt.wantCode {
				t.Fatalf("code = %d, muốn %d", rec.Code, tt.wantCode)
			}
			acc := s.logs.accessLines(t)
			if len(acc) != 1 {
				t.Fatalf("có %d dòng access log, muốn 1: %s", len(acc), s.logs.String())
			}
			a := acc[0]
			if a["method"] != tt.method || a["path"] != tt.wantPath || a["status"] != float64(tt.wantCode) || a["level"] != tt.wantLevel {
				t.Errorf("access log = %v", a)
			}
			if r, _ := a["route"].(string); r != tt.wantRoute {
				t.Errorf("route = %q, muốn %q", r, tt.wantRoute)
			}
			if a["request_id"] != rec.Header().Get(HeaderRequestID) {
				t.Errorf("request_id = %v, header = %q", a["request_id"], rec.Header().Get(HeaderRequestID))
			}
			if l, ok := a["latency_ms"].(float64); !ok || l < 0 {
				t.Errorf("latency_ms = %v", a["latency_ms"])
			}
			if b, ok := a["bytes"].(float64); !ok || int(b) != rec.Body.Len() {
				t.Errorf("bytes = %v, body %d byte", a["bytes"], rec.Body.Len())
			}
			if strings.Contains(s.logs.String(), "S3cr3t") {
				t.Errorf("log lộ header/query: %s", s.logs.String())
			}
		})
	}
}

// ---------- OTel span ----------

func attrMap(kvs []attribute.KeyValue) map[attribute.Key]attribute.Value {
	m := make(map[attribute.Key]attribute.Value, len(kvs))
	for _, kv := range kvs {
		m[kv.Key] = kv.Value
	}
	return m
}

func TestOTelSpan(t *testing.T) {
	const tp = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	tests := []struct {
		name       string
		db         pinger
		target     string
		hdr        map[string]string
		wantName   string
		wantRoute  string // "" = không có http.route
		wantCode   int
		wantStatus codes.Code
		wantTrace  string // "" = không kiểm
		wantParent string
	}{
		{name: "healthz", target: "/healthz", wantName: "GET /healthz", wantRoute: "/healthz", wantCode: 200, wantStatus: codes.Unset},
		{name: "route tham số", target: "/items/42", wantName: "GET /items/{id}", wantRoute: "/items/{id}", wantCode: 200, wantStatus: codes.Unset},
		{name: "route lạ 404", target: "/x/abc", wantName: "GET", wantCode: 404, wantStatus: codes.Unset},
		{name: "503 là Error", db: fakeDB{err: errors.New("down")}, target: "/readyz", wantName: "GET /readyz", wantRoute: "/readyz", wantCode: 503, wantStatus: codes.Error},
		{name: "traceparent hợp lệ", target: "/healthz", hdr: map[string]string{"traceparent": tp},
			wantName: "GET /healthz", wantRoute: "/healthz", wantCode: 200, wantStatus: codes.Unset,
			wantTrace: "4bf92f3577b34da6a3ce929d0e0e4736", wantParent: "00f067aa0ba902b7"},
		{name: "traceparent sai", target: "/healthz", hdr: map[string]string{"traceparent": "garbage"},
			wantName: "GET /healthz", wantRoute: "/healthz", wantCode: 200, wantStatus: codes.Unset},
		{name: "Authorization không vào span", target: "/healthz", hdr: map[string]string{"Authorization": "Bearer S3cr3tAUTH"},
			wantName: "GET /healthz", wantRoute: "/healthz", wantCode: 200, wantStatus: codes.Unset},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStack(t, tt.db)
			rec := s.do(http.MethodGet, tt.target, tt.hdr)
			if rec.Code != tt.wantCode {
				t.Fatalf("code = %d, muốn %d", rec.Code, tt.wantCode)
			}
			spans := s.spans.Ended()
			if len(spans) != 1 {
				t.Fatalf("có %d span, muốn 1", len(spans))
			}
			sp := spans[0]
			if sp.Name() != tt.wantName {
				t.Errorf("tên span = %q, muốn %q", sp.Name(), tt.wantName)
			}
			if strings.Contains(sp.Name(), "42") || strings.Contains(sp.Name(), "abc") {
				t.Errorf("tên span chứa path thô: %q", sp.Name())
			}
			if sp.SpanKind() != trace.SpanKindServer {
				t.Errorf("kind = %v, muốn Server", sp.SpanKind())
			}
			am := attrMap(sp.Attributes())
			if am["http.request.method"].AsString() != "GET" {
				t.Errorf("http.request.method = %v", am["http.request.method"])
			}
			if am["http.response.status_code"].AsInt64() != int64(tt.wantCode) {
				t.Errorf("http.response.status_code = %v", am["http.response.status_code"])
			}
			route, hasRoute := am["http.route"]
			if tt.wantRoute == "" && hasRoute {
				t.Errorf("không muốn http.route, có %q", route.AsString())
			}
			if tt.wantRoute != "" && route.AsString() != tt.wantRoute {
				t.Errorf("http.route = %q, muốn %q", route.AsString(), tt.wantRoute)
			}
			if sp.Status().Code != tt.wantStatus {
				t.Errorf("status = %v, muốn %v", sp.Status().Code, tt.wantStatus)
			}
			for _, kv := range sp.Attributes() {
				if strings.Contains(kv.Value.String(), "S3cr3t") {
					t.Errorf("thuộc tính span lộ header: %v", kv)
				}
			}
			sc := sp.SpanContext()
			if !sc.IsValid() {
				t.Fatal("span context không hợp lệ")
			}
			if tt.wantTrace != "" {
				if sc.TraceID().String() != tt.wantTrace {
					t.Errorf("TraceID = %s, muốn %s", sc.TraceID(), tt.wantTrace)
				}
				if sp.Parent().SpanID().String() != tt.wantParent {
					t.Errorf("Parent.SpanID = %s, muốn %s", sp.Parent().SpanID(), tt.wantParent)
				}
			} else if sp.Parent().IsValid() {
				t.Errorf("không có traceparent hợp lệ nhưng span có parent %v", sp.Parent())
			}
			// Access log mang trace_id của span.
			acc := s.logs.accessLines(t)
			if len(acc) != 1 || acc[0]["trace_id"] != sc.TraceID().String() {
				t.Errorf("access log trace_id = %v, muốn %s", acc, sc.TraceID())
			}
		})
	}
}

// Log *Context trong handler có trace_id/span_id của span server.
func TestTraceIDInHandlerLog(t *testing.T) {
	s := newStack(t, nil)
	s.do(http.MethodGet, "/items/7", nil)
	sp := s.spans.Ended()[0]
	var found bool
	for _, m := range s.logs.lines(t) {
		if m["msg"] == "item handler" {
			found = true
			if m["trace_id"] != sp.SpanContext().TraceID().String() || m["span_id"] != sp.SpanContext().SpanID().String() {
				t.Errorf("log handler trace_id/span_id = %v/%v, span %s/%s",
					m["trace_id"], m["span_id"], sp.SpanContext().TraceID(), sp.SpanContext().SpanID())
			}
		}
	}
	if !found {
		t.Fatal("thiếu log của handler")
	}
}

// ---------- Metric ----------

func collectDuration(t *testing.T, r *sdkmetric.ManualReader) (metricdata.Metrics, metricdata.Histogram[float64]) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := r.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "http.server.request.duration" {
				h, ok := m.Data.(metricdata.Histogram[float64])
				if !ok {
					t.Fatalf("kiểu dữ liệu = %T, muốn Histogram[float64]", m.Data)
				}
				return m, h
			}
		}
	}
	t.Fatal("không có metric http.server.request.duration")
	return metricdata.Metrics{}, metricdata.Histogram[float64]{}
}

var forbiddenMetricAttrs = []attribute.Key{"url.path", "url.full", "url.query", "http.target", "user_agent.original", "client.address"}

func TestMetricDuration(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		wantCode  int64
		wantRoute string
	}{
		{name: "healthz", target: "/healthz", wantCode: 200, wantRoute: "/healthz"},
		{name: "route tham số", target: "/items/9", wantCode: 200, wantRoute: "/items/{id}"},
		{name: "404", target: "/khong-co", wantCode: 404},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStack(t, nil)
			s.do(http.MethodGet, tt.target, map[string]string{"User-Agent": "ua-test"})
			m, h := collectDuration(t, s.reader)
			if m.Unit != "s" {
				t.Errorf("unit = %q, muốn s", m.Unit)
			}
			if len(h.DataPoints) != 1 || h.DataPoints[0].Count != 1 {
				t.Fatalf("data points = %+v, muốn 1 điểm Count 1", h.DataPoints)
			}
			attrs := h.DataPoints[0].Attributes
			if v, _ := attrs.Value("http.request.method"); v.AsString() != "GET" {
				t.Errorf("http.request.method = %v", v)
			}
			if v, _ := attrs.Value("http.response.status_code"); v.AsInt64() != tt.wantCode {
				t.Errorf("http.response.status_code = %v, muốn %d", v, tt.wantCode)
			}
			v, ok := attrs.Value("http.route")
			if tt.wantRoute == "" && ok {
				t.Errorf("không muốn http.route, có %v", v)
			}
			if tt.wantRoute != "" && v.AsString() != tt.wantRoute {
				t.Errorf("http.route = %v, muốn %s", v, tt.wantRoute)
			}
			for _, k := range forbiddenMetricAttrs {
				if attrs.HasValue(k) {
					t.Errorf("metric có thuộc tính cấm %s", k)
				}
			}
		})
	}
}

// 50 path /items/<i> và 50 path lạ khác nhau chỉ sinh 2 tập thuộc tính.
func TestMetricCardinality(t *testing.T) {
	s := newStack(t, nil)
	for i := 0; i < 50; i++ {
		s.do(http.MethodGet, fmt.Sprintf("/items/%d", i), nil)
		s.do(http.MethodGet, fmt.Sprintf("/rand-%d/x", i), nil)
	}
	_, h := collectDuration(t, s.reader)
	if len(h.DataPoints) > 2 {
		t.Errorf("có %d tập thuộc tính, muốn ≤ 2", len(h.DataPoints))
	}
	var total uint64
	seen := map[string]bool{}
	for _, dp := range h.DataPoints {
		total += dp.Count
		route, _ := dp.Attributes.Value("http.route")
		code, _ := dp.Attributes.Value("http.response.status_code")
		seen[fmt.Sprintf("%s|%d", route.AsString(), code.AsInt64())] = true
	}
	if total != 100 {
		t.Errorf("tổng Count = %d, muốn 100", total)
	}
	if !seen["/items/{id}|200"] || !seen["|404"] {
		t.Errorf("tập thuộc tính = %v", seen)
	}
}

// Method lạ không tạo chuỗi thời gian mới.
func TestMetricUnknownMethod(t *testing.T) {
	s := newStack(t, nil)
	s.do("FOOBAR", "/healthz", nil)
	_, h := collectDuration(t, s.reader)
	if v, _ := h.DataPoints[0].Attributes.Value("http.request.method"); v.AsString() != "_OTHER" {
		t.Errorf("http.request.method = %v, muốn _OTHER", v)
	}
}

// ---------- Thứ tự middleware ----------

// Dùng đúng NewRouter mà main dùng (không lắp lại chuỗi trong test) với handler
// panic: mọi tầng phải thấy 500 và cùng request_id/trace_id.
func TestMiddlewareChainOrder(t *testing.T) {
	s := newStack(t, nil)
	rec := s.do(http.MethodGet, "/panic", nil)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d, muốn 500", rec.Code)
	}
	rid := rec.Header().Get(HeaderRequestID)
	if rid == "" {
		t.Fatal("response 500 thiếu X-Request-ID")
	}

	var panicLine, accLine map[string]any
	for _, m := range s.logs.lines(t) {
		if m["level"] == "ERROR" {
			panicLine = m
		}
		if _, ok := m["latency_ms"]; ok {
			accLine = m
		}
	}
	if panicLine == nil || accLine == nil {
		t.Fatalf("thiếu dòng panic hoặc access log: %s", s.logs.String())
	}
	if accLine["status"] != float64(500) {
		t.Errorf("access log status = %v, muốn 500", accLine["status"])
	}
	if accLine["request_id"] != rid || panicLine["request_id"] != rid {
		t.Errorf("request_id access=%v panic=%v header=%s", accLine["request_id"], panicLine["request_id"], rid)
	}

	spans := s.spans.Ended()
	if len(spans) != 1 {
		t.Fatalf("có %d span, muốn 1", len(spans))
	}
	sp := spans[0]
	if sp.Status().Code != codes.Error {
		t.Errorf("span status = %v, muốn Error", sp.Status().Code)
	}
	if v := attrMap(sp.Attributes())["http.response.status_code"]; v.AsInt64() != 500 {
		t.Errorf("span http.response.status_code = %v", v)
	}
	if accLine["trace_id"] != sp.SpanContext().TraceID().String() {
		t.Errorf("access log trace_id = %v, span %s", accLine["trace_id"], sp.SpanContext().TraceID())
	}

	_, h := collectDuration(t, s.reader)
	if len(h.DataPoints) != 1 {
		t.Fatalf("metric có %d điểm, muốn 1", len(h.DataPoints))
	}
	if v, _ := h.DataPoints[0].Attributes.Value("http.response.status_code"); v.AsInt64() != 500 {
		t.Errorf("metric status = %v, muốn 500", v)
	}
}

func ptr(s string) *string { return &s }
