// Package config đọc và kiểm tra cấu hình của core từ biến môi trường.
//
// Mọi lỗi cấu hình được phát hiện ở đây, trước khi mở cổng HTTP hay tạo pool,
// và thông báo lỗi luôn nêu tên biến. Thông báo không bao giờ chứa mật khẩu
// của DSN.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Tên biến môi trường mà core đọc.
const (
	EnvDatabaseURL = "CORE_DATABASE_URL"
	EnvHTTPAddr    = "CORE_HTTP_ADDR"
	EnvLogLevel    = "CORE_LOG_LEVEL"
	// EnvShutdownTimeout là thời gian tối đa chờ request đang chạy khi dừng
	// (định dạng time.ParseDuration, ví dụ 10s, 1500ms).
	EnvShutdownTimeout = "CORE_SHUTDOWN_TIMEOUT"
)

// Giá trị mặc định.
const (
	DefaultHTTPAddr        = ":8080"
	DefaultLogLevel        = slog.LevelInfo
	DefaultShutdownTimeout = 10 * time.Second

	// defaultConnectTimeout áp cho mỗi lần mở kết nối PG khi DSN không có
	// connect_timeout: kết nối mở dở (PG treo mạng) không bị giữ mãi.
	defaultConnectTimeout = 5 * time.Second
)

// Config là cấu hình đã được kiểm tra của core.
type Config struct {
	// DB là cấu hình pool đã parse từ CORE_DATABASE_URL. Giữ bản đã parse
	// thay vì chuỗi DSN để không phải parse lại và không mang mật khẩu đi
	// dưới dạng chuỗi dễ bị log nhầm.
	DB       *pgxpool.Config
	HTTPAddr string
	LogLevel slog.Level
	// ShutdownTimeout luôn > 0: hạn chờ request đang chạy khi nhận SIGTERM.
	ShutdownTimeout time.Duration
}

// Load đọc cấu hình qua lookup (thường là os.LookupEnv). Mọi biến sai được
// gom lại trong một error, mỗi lỗi bắt đầu bằng tên biến.
func Load(lookup func(string) (string, bool)) (Config, error) {
	cfg := Config{HTTPAddr: DefaultHTTPAddr, LogLevel: DefaultLogLevel, ShutdownTimeout: DefaultShutdownTimeout}
	var errs []error

	if dsn, _ := lookup(EnvDatabaseURL); strings.TrimSpace(dsn) == "" {
		errs = append(errs, fmt.Errorf("%s: bắt buộc, chưa đặt hoặc rỗng", EnvDatabaseURL))
	} else if db, err := pgxpool.ParseConfig(dsn); err != nil {
		// Lỗi của pgx đã che mật khẩu (redact) — unit test kiểm điều này.
		errs = append(errs, fmt.Errorf("%s: DSN không hợp lệ: %w", EnvDatabaseURL, err))
	} else {
		if db.ConnConfig.ConnectTimeout == 0 {
			db.ConnConfig.ConnectTimeout = defaultConnectTimeout
		}
		cfg.DB = db
	}

	if v, ok := lookup(EnvHTTPAddr); ok && strings.TrimSpace(v) != "" {
		if err := validateAddr(v); err != nil {
			errs = append(errs, fmt.Errorf("%s: %q không phải host:port hợp lệ: %w", EnvHTTPAddr, v, err))
		} else {
			cfg.HTTPAddr = v
		}
	}

	if v, ok := lookup(EnvLogLevel); ok && strings.TrimSpace(v) != "" {
		lvl, err := parseLevel(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", EnvLogLevel, err))
		} else {
			cfg.LogLevel = lvl
		}
	}

	if v, ok := lookup(EnvShutdownTimeout); ok && strings.TrimSpace(v) != "" {
		d, err := parsePositiveDuration(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", EnvShutdownTimeout, err))
		} else {
			cfg.ShutdownTimeout = d
		}
	}

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}
	return cfg, nil
}

// parsePositiveDuration nhận định dạng time.ParseDuration và bắt buộc > 0.
// "10" (thiếu đơn vị) lỗi do ParseDuration; "0" parse được nên phải chặn riêng.
func parsePositiveDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("%q không phải duration hợp lệ (ví dụ 10s, 1500ms): %w", s, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%q phải lớn hơn 0", s)
	}
	return d, nil
}

// validateAddr chấp nhận "host:port" hoặc ":port", port là số 0..65535.
func validateAddr(addr string) error {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 0 || n > 65535 {
		return fmt.Errorf("port %q phải là số 0..65535", port)
	}
	return nil
}

// parseLevel chỉ nhận 4 mức debug|info|warn|error (không phân biệt hoa thường).
// Không dùng slog.Level.UnmarshalText vì nó nhận cả dạng "info+2".
func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	}
	return 0, fmt.Errorf("%q không hợp lệ, chỉ nhận debug|info|warn|error", s)
}
