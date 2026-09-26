package otelx

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// otelEnv là mọi biến OTEL_* có thể ảnh hưởng test; bị xoá trước mỗi case để
// môi trường của máy chạy test không lọt vào.
var otelEnv = []string{
	EnvEndpoint, EnvTracesEndpoint, EnvMetricsEndpoint, EnvSDKDisabled,
	"OTEL_SERVICE_NAME", "OTEL_RESOURCE_ATTRIBUTES", "OTEL_TRACES_SAMPLER", "OTEL_TRACES_SAMPLER_ARG",
	"OTEL_BSP_SCHEDULE_DELAY", "OTEL_METRIC_EXPORT_INTERVAL", "OTEL_EXPORTER_OTLP_TIMEOUT",
	"OTEL_EXPORTER_OTLP_HEADERS", "OTEL_EXPORTER_OTLP_PROTOCOL", "OTEL_EXPORTER_OTLP_COMPRESSION",
}

// isolate xoá env OTEL_*, đặt env của case, và đưa global OTel về trạng thái
// "chưa cấu hình" (no-op, propagator chỉ TraceContext làm dấu) trước và sau
// case.
func isolate(t *testing.T, env map[string]string) {
	t.Helper()
	for _, k := range otelEnv {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
	// Chỉ Set khi khác giá trị hiện tại: Set lại đúng giá trị đang có khiến
	// OTel log lỗi "infinite loop".
	reset := func() {
		if _, ok := otel.GetTracerProvider().(tracenoop.TracerProvider); !ok {
			otel.SetTracerProvider(tracenoop.NewTracerProvider())
		}
		if _, ok := otel.GetMeterProvider().(metricnoop.MeterProvider); !ok {
			otel.SetMeterProvider(metricnoop.NewMeterProvider())
		}
		if _, ok := otel.GetTextMapPropagator().(propagation.TraceContext); !ok {
			otel.SetTextMapPropagator(propagation.TraceContext{})
		}
		otel.SetErrorHandler(otel.ErrorHandlerFunc(func(error) {}))
	}
	reset()
	t.Cleanup(reset)
}

// otlpRequest là một request mà fake collector nhận.
type otlpRequest struct {
	path, contentType string
	body              []byte
}

// fakeCollector là OTLP/HTTP giả: ghi mọi request, trả status (mặc định 200),
// hoặc treo tới cleanup khi hang.
type fakeCollector struct {
	*httptest.Server
	mu     sync.Mutex
	reqs   []otlpRequest
	status int
	hang   chan struct{}
}

func newFakeCollector(t *testing.T, status int, hang bool) *fakeCollector {
	t.Helper()
	f := &fakeCollector{status: status}
	if hang {
		f.hang = make(chan struct{})
	}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.reqs = append(f.reqs, otlpRequest{r.URL.Path, r.Header.Get("Content-Type"), body})
		f.mu.Unlock()
		if f.hang != nil {
			select {
			case <-f.hang:
			case <-r.Context().Done():
			}
			return
		}
		w.WriteHeader(f.status)
	}))
	t.Cleanup(f.Close)
	if hang {
		t.Cleanup(func() { close(f.hang) }) // chạy trước f.Close (LIFO)
	}
	return f
}

func (f *fakeCollector) requests() []otlpRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.reqs)
}

// has: fake đã nhận POST path có Content-Type protobuf và body chứa mọi chuỗi.
func (f *fakeCollector) has(path string, subs ...string) bool {
	for _, r := range f.requests() {
		if r.path != path || r.contentType != "application/x-protobuf" {
			continue
		}
		ok := true
		for _, s := range subs {
			ok = ok && bytes.Contains(r.body, []byte(s))
		}
		if ok {
			return true
		}
	}
	return false
}

// logBuffer bắt log slog JSON từ nhiều goroutine.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *logBuffer) records(t *testing.T) []map[string]any {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(b.buf.String()), "\n") {
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

func newTestLogger() (*slog.Logger, *logBuffer) {
	b := &logBuffer{}
	return slog.New(slog.NewJSONHandler(b, &slog.HandlerOptions{Level: slog.LevelDebug})), b
}

var testCfg = Config{ServiceName: "svc-test", ServiceNamespace: "ns-test", ServiceVersion: "v-test"}

// emit tạo một span và một điểm metric qua global provider.
func emit(t *testing.T, span, counter string) {
	t.Helper()
	_, s := otel.Tracer("otelx-test").Start(context.Background(), span)
	s.End()
	c, err := otel.Meter("otelx-test").Int64Counter(counter)
	if err != nil {
		t.Fatal(err)
	}
	c.Add(context.Background(), 1)
}

func eventually(t *testing.T, limit time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cond()
}

func mustSetup(t *testing.T, log *slog.Logger) ShutdownFunc {
	t.Helper()
	shutdown, err := Setup(context.Background(), log, testCfg)
	if err != nil {
		t.Fatalf("Setup() lỗi: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = shutdown(ctx)
	})
	return shutdown
}

// ---- U1: khởi tạo đặt provider + propagator global ----

func TestSetupGlobals(t *testing.T) {
	fake := newFakeCollector(t, http.StatusOK, false)
	isolate(t, map[string]string{EnvEndpoint: fake.URL})
	log, _ := newTestLogger()
	mustSetup(t, log)

	_, span := otel.Tracer("u1").Start(context.Background(), "u1")
	defer span.End()
	if !span.IsRecording() || !span.SpanContext().IsValid() {
		t.Errorf("span recording=%v valid=%v, muốn cả hai true", span.IsRecording(), span.SpanContext().IsValid())
	}
	if _, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); !ok {
		t.Errorf("TracerProvider global = %T, muốn *sdktrace.TracerProvider", otel.GetTracerProvider())
	}
	if _, ok := otel.GetMeterProvider().(*sdkmetric.MeterProvider); !ok {
		t.Errorf("MeterProvider global = %T, muốn *sdkmetric.MeterProvider", otel.GetMeterProvider())
	}
	fields := otel.GetTextMapPropagator().Fields()
	for _, want := range []string{"traceparent", "tracestate", "baggage"} {
		if !slices.Contains(fields, want) {
			t.Errorf("propagator Fields() = %v, thiếu %s", fields, want)
		}
	}
}

// ---- U2: resource — giá trị mặc định và thứ tự ưu tiên env ----

func TestResource(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want map[string]string
	}{
		{
			name: "mặc định theo Config + telemetry.sdk",
			want: map[string]string{
				"service.name": "svc-test", "service.namespace": "ns-test", "service.version": "v-test",
				"telemetry.sdk.language": "go", "telemetry.sdk.name": "opentelemetry",
			},
		},
		{
			name: "OTEL_SERVICE_NAME đè service.name",
			env:  map[string]string{"OTEL_SERVICE_NAME": "svc-env"},
			want: map[string]string{"service.name": "svc-env", "service.namespace": "ns-test"},
		},
		{
			name: "OTEL_RESOURCE_ATTRIBUTES thêm key mới và đè service.namespace",
			env:  map[string]string{"OTEL_RESOURCE_ATTRIBUTES": "deployment.environment.name=r1,service.namespace=ns-env"},
			want: map[string]string{
				"service.name": "svc-test", "service.namespace": "ns-env",
				"deployment.environment.name": "r1", "service.version": "v-test",
			},
		},
		{
			name: "OTEL_SERVICE_NAME thắng service.name trong OTEL_RESOURCE_ATTRIBUTES",
			env: map[string]string{
				"OTEL_SERVICE_NAME":        "svc-env",
				"OTEL_RESOURCE_ATTRIBUTES": "service.name=ignored",
			},
			want: map[string]string{"service.name": "svc-env", "service.namespace": "ns-test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolate(t, tt.env)
			res, err := newResource(context.Background(), testCfg)
			if err != nil {
				t.Fatalf("newResource() lỗi: %v", err)
			}
			for k, want := range tt.want {
				v, ok := res.Set().Value(attribute.Key(k))
				if !ok || v.String() != want {
					t.Errorf("%s = %q (có=%v), muốn %q", k, v.String(), ok, want)
				}
			}
		})
	}
}

// ---- U3: export OTLP/HTTP protobuf tới endpoint từ env ----

func TestExportOTLPHTTP(t *testing.T) {
	fake := newFakeCollector(t, http.StatusOK, false)
	isolate(t, map[string]string{
		EnvEndpoint:                   fake.URL,
		"OTEL_BSP_SCHEDULE_DELAY":     "100",
		"OTEL_METRIC_EXPORT_INTERVAL": "200",
	})
	log, _ := newTestLogger()
	mustSetup(t, log)
	emit(t, "u3-span", "u3.counter")

	if !eventually(t, 5*time.Second, func() bool { return fake.has("/v1/traces", "u3-span", "svc-test") }) {
		t.Errorf("fake không nhận POST /v1/traces protobuf chứa span u3-span + service.name; nhận %d request", len(fake.requests()))
	}
	if !eventually(t, 5*time.Second, func() bool { return fake.has("/v1/metrics", "u3.counter", "svc-test") }) {
		t.Errorf("fake không nhận POST /v1/metrics protobuf chứa u3.counter + service.name")
	}
}

// ---- U4: shutdown flush dữ liệu còn trong bộ đệm ----

func TestShutdownFlushes(t *testing.T) {
	fake := newFakeCollector(t, http.StatusOK, false)
	isolate(t, map[string]string{
		EnvEndpoint:                   fake.URL,
		"OTEL_BSP_SCHEDULE_DELAY":     "60000",
		"OTEL_METRIC_EXPORT_INTERVAL": "600000",
	})
	log, _ := newTestLogger()
	shutdown := mustSetup(t, log)
	emit(t, "u4-span", "u4.counter")
	time.Sleep(200 * time.Millisecond)
	if n := len(fake.requests()); n != 0 {
		t.Fatalf("fake đã nhận %d request trước shutdown, muốn 0 (batch/interval dài)", n)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown() = %v, muốn nil", err)
	}
	// Kiểm NGAY sau khi shutdown trả về: dữ liệu phải tới trước đó.
	if !fake.has("/v1/traces", "u4-span") || !fake.has("/v1/metrics", "u4.counter") {
		t.Errorf("sau shutdown fake chưa có span/metric; nhận %d request", len(fake.requests()))
	}
	if err := shutdown(ctx); err != nil {
		t.Errorf("shutdown lần hai = %v, muốn nil (kết quả lần đầu)", err)
	}
}

// ---- U5: shutdown tôn trọng hạn ctx khi collector treo / cổng đóng ----

func TestShutdownBounded(t *testing.T) {
	tests := []struct {
		name     string
		endpoint func(t *testing.T) string
	}{
		{"collector treo", func(t *testing.T) string { return newFakeCollector(t, 0, true).URL }},
		{"cổng đóng", func(t *testing.T) string { return closedPortURL(t) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolate(t, map[string]string{
				EnvEndpoint:                   tt.endpoint(t),
				"OTEL_BSP_SCHEDULE_DELAY":     "60000",
				"OTEL_METRIC_EXPORT_INTERVAL": "600000",
			})
			log, _ := newTestLogger()
			shutdown := mustSetup(t, log)
			emit(t, "u5-span", "u5.counter")

			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			start := time.Now()
			err := shutdown(ctx)
			if d := time.Since(start); d >= 2*time.Second {
				t.Errorf("shutdown mất %v, muốn < 2s", d)
			}
			t.Logf("shutdown() = %v sau %v", err, time.Since(start))
		})
	}
}

// closedPortURL trả URL tới cổng vừa đóng (kết nối bị từ chối).
func closedPortURL(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return "http://" + addr
}

// ---- U6: OTEL_SDK_DISABLED ----

func TestSDKDisabled(t *testing.T) {
	tests := []struct {
		value    string
		disabled bool
	}{
		{"true", true},
		{"TRUE", true},
		{" True ", true},
		{"false", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run("OTEL_SDK_DISABLED="+tt.value, func(t *testing.T) {
			fake := newFakeCollector(t, http.StatusOK, false)
			isolate(t, map[string]string{EnvEndpoint: fake.URL, EnvSDKDisabled: tt.value})
			log, _ := newTestLogger()
			shutdown := mustSetup(t, log)

			_, span := otel.Tracer("u6").Start(context.Background(), "u6-span")
			recording := span.IsRecording()
			span.End()
			emit(t, "u6-span-2", "u6.counter")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := shutdown(ctx); err != nil && tt.disabled {
				t.Errorf("shutdown() = %v, muốn nil khi tắt", err)
			}
			if !slices.Contains(otel.GetTextMapPropagator().Fields(), "baggage") {
				t.Errorf("propagator chưa được đặt: Fields() = %v", otel.GetTextMapPropagator().Fields())
			}
			n := len(fake.requests())
			if tt.disabled {
				if recording || n != 0 {
					t.Errorf("tắt: span recording=%v, fake nhận %d request; muốn false, 0", recording, n)
				}
				if _, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); ok {
					t.Error("tắt nhưng TracerProvider global là SDK")
				}
			} else if !recording || n == 0 {
				t.Errorf("bật: span recording=%v, fake nhận %d request; muốn true, > 0", recording, n)
			}
		})
	}
}

// ---- U7: kiểm định dạng endpoint ----

func TestEndpointValidation(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string // rỗng = không lỗi
	}{
		{"thiếu scheme", map[string]string{EnvEndpoint: "localhost:4318"}, EnvEndpoint},
		{"không parse được", map[string]string{EnvEndpoint: "://bad"}, EnvEndpoint},
		{"scheme ftp", map[string]string{EnvEndpoint: "ftp://127.0.0.1:4318"}, EnvEndpoint},
		{"endpoint riêng cho trace sai", map[string]string{EnvTracesEndpoint: "localhost:4318"}, EnvTracesEndpoint},
		{"http không tới được vẫn hợp lệ", map[string]string{EnvEndpoint: "http://127.0.0.1:1"}, ""},
		{"https", map[string]string{EnvEndpoint: "https://x:4318"}, ""},
		{"không đặt", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolate(t, tt.env)
			log, _ := newTestLogger()
			shutdown, err := Setup(context.Background(), log, testCfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Setup() = %v, muốn nil", err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				defer cancel()
				_ = shutdown(ctx) // không có dữ liệu; lỗi kết nối không quan trọng ở đây
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Setup() = %v, muốn lỗi nêu %s", err, tt.wantErr)
			}
			if shutdown != nil {
				t.Error("lỗi nhưng vẫn trả shutdown khác nil")
			}
			// Global không bị thay: vẫn no-op, propagator vẫn là dấu của isolate.
			if _, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); ok {
				t.Error("TracerProvider global bị thay dù Setup lỗi")
			}
			if _, ok := otel.GetMeterProvider().(*sdkmetric.MeterProvider); ok {
				t.Error("MeterProvider global bị thay dù Setup lỗi")
			}
			if f := otel.GetTextMapPropagator().Fields(); slices.Contains(f, "baggage") {
				t.Errorf("propagator bị thay dù Setup lỗi: %v", f)
			}
		})
	}
}

// endpointSet quyết định có thêm WithInsecure (mặc định http://localhost:4318
// thay cho https của SDK) hay không: chỉ khi không có env endpoint nào.
func TestEndpointSet(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"không đặt → mặc định http://localhost:4318", nil, false},
		{"rỗng coi như không đặt", map[string]string{EnvEndpoint: "  "}, false},
		{"biến chung", map[string]string{EnvEndpoint: "https://x:4318"}, true},
		{"biến riêng của signal", map[string]string{EnvTracesEndpoint: "http://x:4318/v1/traces"}, true},
		{"biến riêng của signal khác không tính", map[string]string{EnvMetricsEndpoint: "http://x:4318/v1/metrics"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := func(k string) (string, bool) { v, ok := tt.env[k]; return v, ok }
			if got := endpointSet(lookup, EnvTracesEndpoint); got != tt.want {
				t.Errorf("endpointSet() = %v, muốn %v", got, tt.want)
			}
		})
	}
}

// ---- U8: lỗi export → log slog WARN, không in ra stderr ----

func TestExportErrorLogged(t *testing.T) {
	tests := []struct {
		name     string
		endpoint func(t *testing.T) string
	}{
		{"collector trả 500", func(t *testing.T) string { return newFakeCollector(t, http.StatusInternalServerError, false).URL }},
		{"cổng đóng", func(t *testing.T) string { return closedPortURL(t) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolate(t, map[string]string{
				EnvEndpoint:                   tt.endpoint(t),
				"OTEL_BSP_SCHEDULE_DELAY":     "50",
				"OTEL_METRIC_EXPORT_INTERVAL": "100",
				"OTEL_EXPORTER_OTLP_TIMEOUT":  "1000",
			})
			log, logs := newTestLogger()
			mustSetup(t, log)
			emit(t, "u8-span", "u8.counter")
			ok := eventually(t, 5*time.Second, func() bool {
				for _, r := range logs.records(t) {
					if r["level"] == "WARN" && r["err"] != nil {
						return true
					}
				}
				return false
			})
			if !ok {
				t.Errorf("không có log WARN có err sau lỗi export; log: %v", logs.records(t))
			}
		})
	}
}

// TestNoStderrOnExportError chạy lại test binary làm tiến trình con (collector
// trả 500) và kiểm: lỗi export chỉ ra stdout dạng JSON WARN, stderr rỗng —
// logger mặc định của OTel (stdr → stderr) không được dùng.
func TestNoStderrOnExportError(t *testing.T) {
	if os.Getenv("OTELX_HELPER_ENDPOINT") != "" {
		return // đang là tiến trình con: TestHelperExport làm việc
	}
	fake := newFakeCollector(t, http.StatusInternalServerError, false)
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperExport$", "-test.count=1")
	cmd.Env = append(os.Environ(), "OTELX_HELPER_ENDPOINT="+fake.URL)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("tiến trình con lỗi: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr không rỗng:\n%s", stderr.String())
	}
	var warn bool
	for _, l := range strings.Split(stdout.String(), "\n") {
		var m map[string]any
		if json.Unmarshal([]byte(l), &m) == nil && m["level"] == "WARN" && m["err"] != nil {
			warn = true
		}
	}
	if !warn {
		t.Errorf("stdout không có dòng JSON WARN lỗi export:\n%s", stdout.String())
	}
}

// TestHelperExport chỉ chạy trong tiến trình con của TestNoStderrOnExportError.
func TestHelperExport(t *testing.T) {
	ep := os.Getenv("OTELX_HELPER_ENDPOINT")
	if ep == "" {
		return
	}
	isolate(t, map[string]string{
		EnvEndpoint:                   ep,
		"OTEL_BSP_SCHEDULE_DELAY":     "50",
		"OTEL_METRIC_EXPORT_INTERVAL": "100",
	})
	logs := &logBuffer{}
	log := slog.New(slog.NewJSONHandler(io.MultiWriter(os.Stdout, logs), nil))
	shutdown := mustSetup(t, log)
	emit(t, "helper-span", "helper.counter")
	eventually(t, 5*time.Second, func() bool {
		for _, r := range logs.records(t) {
			if r["level"] == "WARN" {
				return true
			}
		}
		return false
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = shutdown(ctx)
}
