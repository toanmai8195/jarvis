package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/trace"
)

// contextHandler bọc một slog.Handler: mỗi bản ghi được thêm request_id (và
// trace_id, span_id khi ctx có span hợp lệ) lấy từ ctx. Nhờ đó mọi lời gọi
// log.*Context(r.Context(), ...) trong request đều truy vết được, không cần
// truyền ID bằng tay.
type contextHandler struct {
	slog.Handler
}

// NewLogHandler bọc h để log theo context (xem contextHandler). Bọc nhiều lần
// vẫn chỉ thêm mỗi key một lần.
func NewLogHandler(h slog.Handler) slog.Handler {
	if _, ok := h.(contextHandler); ok {
		return h
	}
	return contextHandler{h}
}

// withContextLogging trả logger mà handler đã được bọc bởi contextHandler.
func withContextLogging(log *slog.Logger) *slog.Logger {
	if _, ok := log.Handler().(contextHandler); ok {
		return log
	}
	return slog.New(NewLogHandler(log.Handler()))
}

func (h contextHandler) Handle(ctx context.Context, rec slog.Record) error {
	if id := RequestIDFrom(ctx); id != "" {
		rec.AddAttrs(slog.String("request_id", id))
	}
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		rec.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, rec)
}

// WithAttrs/WithGroup phải bọc lại, nếu không logger.With(...) sẽ mất
// contextHandler.
func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}

// accessLog ghi đúng một dòng log JSON cho mỗi request sau khi handler xong.
// request_id/trace_id do contextHandler thêm từ ctx. Không log header nào và
// không log query string (có thể chứa token).
func accessLog(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			status := statusOf(ww)
			level := slog.LevelInfo
			if status >= http.StatusInternalServerError {
				level = slog.LevelWarn
			}
			log.LogAttrs(r.Context(), level, "http request",
				slog.String("method", r.Method),
				slog.String("route", routePattern(r)),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Float64("latency_ms", float64(time.Since(start).Microseconds())/1000),
				slog.Int("bytes", ww.BytesWritten()),
			)
		})
	}
}

// statusOf trả status đã gửi; handler không gọi WriteHeader/Write thì
// net/http trả 200.
func statusOf(ww middleware.WrapResponseWriter) int {
	if s := ww.Status(); s != 0 {
		return s
	}
	return http.StatusOK
}

// routePattern trả route pattern chi đã khớp (ví dụ "/items/{id}"), hoặc ""
// khi không khớp route (404/405). Chỉ đúng khi gọi SAU next.ServeHTTP và
// middleware được gắn bằng r.Use của chi (ctx có RouteContext).
func routePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return ""
	}
	return rctx.RoutePattern()
}
