// Command server là HTTP server của core. main chỉ wiring thủ công:
// config → logger → pool PG → router → http.Server → run (graceful shutdown).
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"

	"github.com/toanmai8195/snaptix/com/tm/server/services/core/internal/config"
	"github.com/toanmai8195/snaptix/com/tm/server/services/core/internal/httpx"
)

func main() {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		// Chưa biết mức log → dùng mặc định; lỗi cấu hình vẫn là log JSON.
		newLogger(config.DefaultLogLevel).Error("cấu hình không hợp lệ", "err", err)
		os.Exit(1)
	}
	log := newLogger(cfg.LogLevel)

	// Pool lười: NewWithConfig không mở kết nối (MinConns = 0), nên core vẫn
	// khởi động khi PG chưa lên; /readyz báo 503 tới khi ping được.
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg.DB)
	if err != nil {
		log.Error("tạo pool PG thất bại", "err", err)
		os.Exit(1)
	}
	// Không đóng pool bằng defer: os.Exit bỏ qua defer. run đóng pool tường
	// minh, sau khi HTTP server đã dừng.
	log.Debug("pool PG đã tạo",
		"db_host", cfg.DB.ConnConfig.Host,
		"db_port", cfg.DB.ConnConfig.Port,
		"db_name", cfg.DB.ConnConfig.Database,
		"max_conns", cfg.DB.MaxConns)

	// Provider/propagator global hiện là no-op (delegate); P0-T10 cấu hình bản
	// thật trong pkg/otelx và delegate tự chuyển sang bản đó.
	handler := httpx.NewRouter(log, pool,
		httpx.WithTracerProvider(otel.GetTracerProvider()),
		httpx.WithMeterProvider(otel.GetMeterProvider()),
		httpx.WithPropagator(otel.GetTextMapPropagator()),
	)
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	// Đăng ký tín hiệu trước khi mở cổng để SIGTERM đến sớm vẫn được xử lý
	// graceful thay vì giết process.
	sigs := notifyShutdown()
	err = run(context.Background(), log, srv, pool, cfg.HTTPAddr, cfg.ShutdownTimeout, sigs)
	os.Exit(exitCode(err))
}

// newLogger trả logger slog JSON ghi ra stdout với mức cho trước. Handler được
// bọc để log *Context trong request tự có request_id, trace_id.
func newLogger(level slog.Level) *slog.Logger {
	return slog.New(httpx.NewLogHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
}
