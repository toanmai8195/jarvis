// Package httpx dựng router HTTP của core: health check, readiness, metrics.
// Route nghiệp vụ và middleware được gắn vào đây ở các task sau.
package httpx

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ReadyTimeout là thời gian tối đa /readyz chờ ping DB. PG treo mạng thì
// /readyz vẫn trả 503 nhanh thay vì treo theo client.
const ReadyTimeout = 2 * time.Second

// pinger là thứ /readyz cần từ DB — khai báo ở phía dùng; *pgxpool.Pool thoả
// interface này, unit test dùng bản giả.
type pinger interface {
	Ping(ctx context.Context) error
}

// NewRouter trả handler với GET /healthz, GET /readyz, GET /metrics.
// Route lạ → 404, method khác GET trên route có sẵn → 405 (mặc định của chi).
func NewRouter(log *slog.Logger, db pinger) http.Handler {
	return newRouter(log, db, ReadyTimeout)
}

func newRouter(log *slog.Logger, db pinger, readyTimeout time.Duration) http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", healthz)
	r.Get("/readyz", readyz(log, db, readyTimeout))
	// r.Method thay vì r.Handle: r.Handle nhận mọi method, POST sẽ không ra 405.
	r.Method(http.MethodGet, "/metrics", promhttp.Handler())
	return r
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
