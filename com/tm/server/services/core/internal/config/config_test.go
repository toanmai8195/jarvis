package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

// env giả cho Load: map → lookup.
func env(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}

const (
	goodDSN = "postgres://snaptix:snaptix@localhost:5432/core?sslmode=disable"
	secret  = "S3cr3tXYZ"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		wantErr   []string // chuỗi phải có trong lỗi; nil = không lỗi
		wantAddr  string
		wantLevel slog.Level
	}{
		{
			name:      "mặc định khi chỉ đặt CORE_DATABASE_URL",
			env:       map[string]string{EnvDatabaseURL: goodDSN},
			wantAddr:  ":8080",
			wantLevel: slog.LevelInfo,
		},
		{
			name:      "biến tuỳ chọn rỗng dùng mặc định",
			env:       map[string]string{EnvDatabaseURL: goodDSN, EnvHTTPAddr: "", EnvLogLevel: " "},
			wantAddr:  ":8080",
			wantLevel: slog.LevelInfo,
		},
		{
			name:      "ghi đè addr và level",
			env:       map[string]string{EnvDatabaseURL: goodDSN, EnvHTTPAddr: "127.0.0.1:18080", EnvLogLevel: "warn"},
			wantAddr:  "127.0.0.1:18080",
			wantLevel: slog.LevelWarn,
		},
		{
			name:      "level không phân biệt hoa thường",
			env:       map[string]string{EnvDatabaseURL: goodDSN, EnvLogLevel: "DeBuG"},
			wantAddr:  ":8080",
			wantLevel: slog.LevelDebug,
		},
		{
			name:      "level ERROR viết hoa",
			env:       map[string]string{EnvDatabaseURL: goodDSN, EnvLogLevel: "ERROR"},
			wantAddr:  ":8080",
			wantLevel: slog.LevelError,
		},
		{
			name:    "thiếu CORE_DATABASE_URL",
			env:     map[string]string{},
			wantErr: []string{EnvDatabaseURL},
		},
		{
			name:    "CORE_DATABASE_URL rỗng",
			env:     map[string]string{EnvDatabaseURL: ""},
			wantErr: []string{EnvDatabaseURL},
		},
		{
			name:    "CORE_DATABASE_URL chỉ khoảng trắng",
			env:     map[string]string{EnvDatabaseURL: "   "},
			wantErr: []string{EnvDatabaseURL},
		},
		{
			name:    "DSN sai port",
			env:     map[string]string{EnvDatabaseURL: "postgres://snaptix:" + secret + "@localhost:notaport/core"},
			wantErr: []string{EnvDatabaseURL},
		},
		{
			name:    "DSN sai sslmode",
			env:     map[string]string{EnvDatabaseURL: "postgres://snaptix:" + secret + "@localhost:5432/core?sslmode=bogus"},
			wantErr: []string{EnvDatabaseURL},
		},
		{
			name:    "CORE_HTTP_ADDR thiếu port (abc)",
			env:     map[string]string{EnvDatabaseURL: goodDSN, EnvHTTPAddr: "abc"},
			wantErr: []string{EnvHTTPAddr, "abc"},
		},
		{
			name:    "CORE_HTTP_ADDR port không phải số",
			env:     map[string]string{EnvDatabaseURL: goodDSN, EnvHTTPAddr: ":http"},
			wantErr: []string{EnvHTTPAddr},
		},
		{
			name:    "CORE_HTTP_ADDR port vượt 65535",
			env:     map[string]string{EnvDatabaseURL: goodDSN, EnvHTTPAddr: ":70000"},
			wantErr: []string{EnvHTTPAddr},
		},
		{
			name:    "CORE_LOG_LEVEL sai",
			env:     map[string]string{EnvDatabaseURL: goodDSN, EnvLogLevel: "verbose"},
			wantErr: []string{EnvLogLevel, "verbose"},
		},
		{
			name:    "CORE_LOG_LEVEL dạng offset của slog bị từ chối",
			env:     map[string]string{EnvDatabaseURL: goodDSN, EnvLogLevel: "info+2"},
			wantErr: []string{EnvLogLevel},
		},
		{
			name:    "nhiều biến sai cùng lúc được gom",
			env:     map[string]string{EnvHTTPAddr: "abc", EnvLogLevel: "verbose"},
			wantErr: []string{EnvDatabaseURL, EnvHTTPAddr, EnvLogLevel},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(env(tt.env))
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Load() không lỗi, muốn lỗi chứa %v", tt.wantErr)
				}
				for _, s := range tt.wantErr {
					if !strings.Contains(err.Error(), s) {
						t.Errorf("lỗi %q thiếu %q", err, s)
					}
				}
				if strings.Contains(err.Error(), secret) {
					t.Errorf("lỗi lộ mật khẩu DSN: %q", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() lỗi: %v", err)
			}
			if cfg.HTTPAddr != tt.wantAddr {
				t.Errorf("HTTPAddr = %q, muốn %q", cfg.HTTPAddr, tt.wantAddr)
			}
			if cfg.LogLevel != tt.wantLevel {
				t.Errorf("LogLevel = %v, muốn %v", cfg.LogLevel, tt.wantLevel)
			}
			if cfg.DB == nil {
				t.Fatal("DB = nil")
			}
			if got := cfg.DB.ConnConfig.Database; got != "core" {
				t.Errorf("Database = %q, muốn core", got)
			}
		})
	}
}

func TestLoadConnectTimeout(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want time.Duration
	}{
		{"mặc định khi DSN không có connect_timeout", goodDSN, defaultConnectTimeout},
		{"giữ connect_timeout của DSN", goodDSN + "&connect_timeout=9", 9 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(env(map[string]string{EnvDatabaseURL: tt.dsn}))
			if err != nil {
				t.Fatalf("Load() lỗi: %v", err)
			}
			if got := cfg.DB.ConnConfig.ConnectTimeout; got != tt.want {
				t.Errorf("ConnectTimeout = %v, muốn %v", got, tt.want)
			}
		})
	}
}
