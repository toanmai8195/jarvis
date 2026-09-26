package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/semconv/v1.40.0/httpconv"
	"go.opentelemetry.io/otel/trace"
)

// instrumentationName là tên scope của tracer/meter HTTP server của core.
const instrumentationName = "github.com/toanmai8195/snaptix/com/tm/server/services/core/internal/httpx"

// durationBuckets là biên histogram (giây) semconv khuyến nghị cho
// http.server.request.duration.
var durationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10}

// knownMethods: method ngoài danh sách ghi là "_OTHER" (semconv) để client
// không tạo được chuỗi thời gian tuỳ ý bằng method lạ.
var knownMethods = map[string]bool{
	http.MethodGet: true, http.MethodHead: true, http.MethodPost: true, http.MethodPut: true,
	http.MethodPatch: true, http.MethodDelete: true, http.MethodConnect: true,
	http.MethodOptions: true, http.MethodTrace: true,
}

// telemetry tạo span server và ghi metric http.server.request.duration cho
// mỗi request. Chỉ dùng OTel API: provider/propagator được truyền vào; SDK và
// exporter thật do P0-T10 cấu hình.
//
// Tên span và thuộc tính http.route chỉ biết SAU khi chi định tuyến, nên
// span được đặt tên "<METHOD>" lúc bắt đầu rồi SetName khi handler xong.
// Không ghi path thô, query hay header vào span/metric: tránh lộ dữ liệu và
// giữ số chuỗi thời gian không tăng theo path.
func telemetry(log *slog.Logger, tp trace.TracerProvider, mp metric.MeterProvider, prop propagation.TextMapPropagator) func(http.Handler) http.Handler {
	tracer := tp.Tracer(instrumentationName, trace.WithSchemaURL(semconv.SchemaURL))
	meter := mp.Meter(instrumentationName, metric.WithSchemaURL(semconv.SchemaURL))
	duration, err := httpconv.NewServerRequestDuration(meter, metric.WithExplicitBucketBoundaries(durationBuckets...))
	if err != nil {
		// Chỉ xảy ra khi tên/đơn vị instrument sai; httpconv trả histogram
		// no-op nên request vẫn chạy, chỉ mất metric.
		log.Error("tạo histogram http.server.request.duration thất bại", "err", err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			method := r.Method
			if !knownMethods[method] {
				method = "_OTHER"
			}

			ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
			ctx, span := tracer.Start(ctx, method,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(semconv.HTTPRequestMethodKey.String(method)),
			)
			defer span.End()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			r = r.WithContext(ctx)
			next.ServeHTTP(ww, r)

			status := statusOf(ww)
			attrs := []attribute.KeyValue{semconv.HTTPResponseStatusCode(status)}
			if route := routePattern(r); route != "" {
				span.SetName(method + " " + route)
				attrs = append(attrs, semconv.HTTPRoute(route))
			}
			span.SetAttributes(attrs...)
			// Server span: chỉ 5xx là Error; 4xx là lỗi của client, để Unset.
			if status >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, http.StatusText(status))
			}

			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			duration.Record(ctx, time.Since(start).Seconds(),
				httpconv.RequestMethodAttr(method), scheme, attrs...)
		})
	}
}
