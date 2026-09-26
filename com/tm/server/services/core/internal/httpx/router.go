// Package httpx dựng router HTTP của core: middleware (request ID, OTel,
// access log, recover), health check, readiness, metrics runtime.
// Route nghiệp vụ được gắn vào đây ở các task sau.
package httpx

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// ReadyTimeout là thời gian tối đa /readyz chờ ping DB. PG treo mạng thì
// /readyz vẫn trả 503 nhanh thay vì treo theo client.
const ReadyTimeout = 2 * time.Second

// pinger là thứ /readyz cần từ DB — khai báo ở phía dùng; *pgxpool.Pool thoả
// interface này, unit test dùng bản giả.
type pinger interface {
	Ping(ctx context.Context) error
}

// Option tuỳ chỉnh NewRouter.
type Option func(*options)

type options struct {
	tp           trace.TracerProvider
	mp           metric.MeterProvider
	prop         propagation.TextMapPropagator
	readyTimeout time.Duration
	// extraRoutes chỉ dùng trong test: gắn route tham số / handler panic vào
	// cùng router (cùng chuỗi middleware) với code chạy thật.
	extraRoutes func(chi.Router)
}

// WithTracerProvider đặt TracerProvider cho span HTTP server.
// Mặc định otel.GetTracerProvider().
func WithTracerProvider(tp trace.TracerProvider) Option {
	return func(o *options) { o.tp = tp }
}

// WithMeterProvider đặt MeterProvider cho metric http.server.request.duration.
// Mặc định otel.GetMeterProvider().
func WithMeterProvider(mp metric.MeterProvider) Option {
	return func(o *options) { o.mp = mp }
}

// WithPropagator đặt propagator đọc traceparent từ header request.
// Mặc định otel.GetTextMapPropagator().
func WithPropagator(p propagation.TextMapPropagator) Option {
	return func(o *options) { o.prop = p }
}

// withReadyTimeout đổi timeout của /readyz (test).
func withReadyTimeout(d time.Duration) Option {
	return func(o *options) { o.readyTimeout = d }
}

// withRoutes gắn thêm route (test) — code chạy thật không có route debug.
func withRoutes(f func(chi.Router)) Option {
	return func(o *options) { o.extraRoutes = f }
}

// NewRouter trả handler của core: chuỗi middleware + GET /healthz, GET /readyz,
// GET /metrics. Route lạ → 404, method khác GET trên route có sẵn → 405
// (mặc định của chi).
//
// Thứ tự middleware (ngoài → trong):
//
//	requestID → telemetry (span, metric) → accessLog → recoverer → route
//
// recoverer nằm TRONG accessLog/telemetry để panic được ghi thành 500 ở access
// log, span và metric; requestID ngoài cùng để mọi response (kể cả 404/405/500)
// có X-Request-ID. Middleware gắn bằng r.Use nên ctx đã có chi.RouteContext và
// đọc được route pattern sau khi handler chạy.
func NewRouter(log *slog.Logger, db pinger, opts ...Option) http.Handler {
	o := options{
		tp:           otel.GetTracerProvider(),
		mp:           otel.GetMeterProvider(),
		prop:         otel.GetTextMapPropagator(),
		readyTimeout: ReadyTimeout,
	}
	for _, opt := range opts {
		opt(&o)
	}
	log = withContextLogging(log)

	r := chi.NewRouter()
	r.Use(
		requestID,
		telemetry(log, o.tp, o.mp, o.prop),
		accessLog(log),
		recoverer(log),
	)
	r.Get("/healthz", healthz)
	r.Get("/readyz", readyz(log, db, o.readyTimeout))
	// r.Method thay vì r.Handle: r.Handle nhận mọi method, POST sẽ không ra 405.
	r.Method(http.MethodGet, "/metrics", promhttp.Handler())
	if o.extraRoutes != nil {
		o.extraRoutes(r)
	}
	return r
}

// newRouter giữ chữ ký cũ cho test P0-T07.
func newRouter(log *slog.Logger, db pinger, readyTimeout time.Duration) http.Handler {
	return NewRouter(log, db, withReadyTimeout(readyTimeout))
}

// healthz: process còn sống. Không chạm DB.
func healthz(w http.ResponseWriter, _ *http.Request) {
	writeStatus(w, http.StatusOK, "ok")
}

// readyz: sẵn sàng nhận request khi ping được DB trong readyTimeout.
func readyz(log *slog.Logger, db pinger, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			// Lỗi chỉ vào log (pgx không đưa mật khẩu vào error), body giữ chung chung.
			log.WarnContext(r.Context(), "readyz: ping database thất bại", "err", err)
			writeStatus(w, http.StatusServiceUnavailable, "unavailable")
			return
		}
		writeStatus(w, http.StatusOK, "ok")
	}
}

func writeStatus(w http.ResponseWriter, code int, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": status})
}
