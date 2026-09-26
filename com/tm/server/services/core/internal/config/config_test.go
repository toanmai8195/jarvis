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

func TestLoadShutdownTimeout(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    time.Duration
		wantErr []string // nil = không lỗi
	}{
		{name: "không đặt → mặc định 10s", env: map[string]string{}, want: 10 * time.Second},
		{name: "rỗng → mặc định", env: map[string]string{EnvShutdownTimeout: ""}, want: DefaultShutdownTimeout},
		{name: "chỉ khoảng trắng → mặc định", env: map[string]string{EnvShutdownTimeout: "  "}, want: DefaultShutdownTimeout},
		{name: "30s", env: map[string]string{EnvShutdownTimeout: "30s"}, want: 30 * time.Second},
		{name: "1500ms", env: map[string]string{EnvShutdownTimeout: "1500ms"}, want: 1500 * time.Millisecond},
		{name: "khoảng trắng hai đầu được bỏ", env: map[string]string{EnvShutdownTimeout: " 2s "}, want: 2 * time.Second},
		{name: "abc không parse được", env: map[string]string{EnvShutdownTimeout: "abc"}, wantErr: []string{"abc"}},
		{name: "10 thiếu đơn vị", env: map[string]string{EnvShutdownTimeout: "10"}, wantErr: []string{`"10"`}},
		{name: "0 bị từ chối", env: map[string]string{EnvShutdownTimeout: "0"}, wantErr: []string{`"0"`}},
		{name: "0s bị từ chối", env: map[string]string{EnvShutdownTimeout: "0s"}, wantErr: []string{`"0s"`}},
		{name: "âm bị từ chối", env: map[string]string{EnvShutdownTimeout: "-1s"}, wantErr: []string{`"-1s"`}},
		{
			name:    "gom chung với lỗi biến khác",
			env:     map[string]string{EnvShutdownTimeout: "abc", EnvLogLevel: "verbose"},
			wantErr: []string{EnvLogLevel},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := map[string]string{EnvDatabaseURL: goodDSN}
			for k, v := range tt.env {
				m[k] = v
			}
			cfg, err := Load(env(m))
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Load() không lỗi, muốn lỗi %s", EnvShutdownTimeout)
				}
				// Mỗi lỗi con (errors.Join nối bằng \n) có một dòng bắt đầu bằng tên biến.
				var found bool
				for _, line := range strings.Split(err.Error(), "\n") {
					if strings.HasPrefix(line, EnvShutdownTimeout+":") {
						found = true
					}
				}
				if !found {
					t.Errorf("lỗi %q không có dòng bắt đầu bằng %s", err, EnvShutdownTimeout)
				}
				for _, s := range tt.wantErr {
					if !strings.Contains(err.Error(), s) {
						t.Errorf("lỗi %q thiếu %q", err, s)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() lỗi: %v", err)
			}
			if cfg.ShutdownTimeout != tt.want {
				t.Errorf("ShutdownTimeout = %v, muốn %v", cfg.ShutdownTimeout, tt.want)
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
