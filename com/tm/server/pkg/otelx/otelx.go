// Package otelx dựng OpenTelemetry SDK dùng chung cho các service Go của
// snaptix: TracerProvider + MeterProvider export OTLP/HTTP (cổng 4318) tới
// otel-collector, propagator W3C TraceContext + Baggage, và đưa lỗi nội bộ
// của OTel về logger slog của service.
//
// Cấu hình chỉ qua biến môi trường chuẩn OTEL_* mà SDK tự đọc
// (OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_EXPORTER_OTLP_TIMEOUT, OTEL_SERVICE_NAME,
// OTEL_RESOURCE_ATTRIBUTES, OTEL_TRACES_SAMPLER[_ARG], OTEL_BSP_SCHEDULE_DELAY,
// OTEL_METRIC_EXPORT_INTERVAL...). otelx không truyền option nào đè lên các
// biến đó; chỉ tự xử lý OTEL_SDK_DISABLED (SDK Go bỏ qua biến này) và kiểm
// định dạng endpoint trước khi khởi tạo.
package otelx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// Biến môi trường otelx tự đọc (các biến OTEL_* khác do SDK đọc).
const (
	EnvEndpoint        = "OTEL_EXPORTER_OTLP_ENDPOINT"
	EnvTracesEndpoint  = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
	EnvMetricsEndpoint = "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"
	EnvSDKDisabled     = "OTEL_SDK_DISABLED"
)

// Config là giá trị MẶC ĐỊNH của resource do service truyền vào. Biến môi
// trường đè theo thứ tự: OTEL_SERVICE_NAME > service.name trong
// OTEL_RESOURCE_ATTRIBUTES > Config.ServiceName; mọi key khác trong
// OTEL_RESOURCE_ATTRIBUTES đè giá trị cùng key ở đây.
type Config struct {
	ServiceName      string // service.name, vd "core"
	ServiceNamespace string // service.namespace, vd "snaptix" → Prometheus job "snaptix/core"
	ServiceVersion   string // service.version
}

// ShutdownFunc flush dữ liệu còn trong bộ đệm rồi dừng provider. Tôn trọng
// deadline của ctx; gọi nhiều lần an toàn (lần sau trả kết quả lần đầu).
type ShutdownFunc func(context.Context) error

// Setup khởi tạo SDK và đặt provider, propagator, error handler, logger làm
// global của OTel. Trả ShutdownFunc để service gọi khi dừng.
//
//   - Endpoint sai định dạng → trả lỗi nêu tên biến, không đổi global nào.
//   - Collector không tới được KHÔNG phải lỗi: export chạy nền, lỗi đi qua
//     error handler thành log WARN, collector sống lại thì export tiếp.
//   - OTEL_SDK_DISABLED=true → không tạo SDK (tracer/meter global giữ no-op),
//     vẫn đặt propagator để traceparent vào vẫn có trace_id trong log;
//     ShutdownFunc là no-op.
func Setup(ctx context.Context, log *slog.Logger, cfg Config) (ShutdownFunc, error) {
	if err := validateEndpoints(os.LookupEnv); err != nil {
		return nil, err
	}
	res, err := newResource(ctx, cfg)
	if err != nil {
		if !errors.Is(err, resource.ErrPartialResource) {
			return nil, fmt.Errorf("otelx: tạo resource: %w", err)
		}
		// Ví dụ OTEL_RESOURCE_ATTRIBUTES có cặp sai: vẫn dùng phần hợp lệ.
		log.Warn("otelx: resource thiếu một phần", "err", err)
	}

	// Lỗi export/nội bộ của OTel → slog (JSON), không in text ra stderr qua
	// logger mặc định (stdr) của OTel.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Warn("otel: lỗi telemetry", "err", err)
	}))
	otel.SetLogger(logr.FromSlogHandler(log.Handler()))
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{}))

	if sdkDisabled(os.LookupEnv) {
		log.Info("otelx: OTEL_SDK_DISABLED=true, không export telemetry")
		return func(context.Context) error { return nil }, nil
	}

	// Không đặt endpoint → SDK mặc định https://localhost:4318, nhưng
	// collector local chỉ nghe http. Chỉ khi đó otelx thêm WithInsecure để
	// mặc định là http://localhost:4318; có env endpoint thì scheme của env
	// quyết định, otelx không đè.
	var (
		topts []otlptracehttp.Option
		mopts []otlpmetrichttp.Option
	)
	if !endpointSet(os.LookupEnv, EnvTracesEndpoint) {
		topts = append(topts, otlptracehttp.WithInsecure())
	}
	if !endpointSet(os.LookupEnv, EnvMetricsEndpoint) {
		mopts = append(mopts, otlpmetrichttp.WithInsecure())
	}
	// New không mở kết nối: collector chết lúc khởi động không chặn service.
	texp, err := otlptracehttp.New(ctx, topts...)
	if err != nil {
		return nil, fmt.Errorf("otelx: tạo trace exporter: %w", err)
	}
	mexp, err := otlpmetrichttp.New(ctx, mopts...)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("otelx: tạo metric exporter: %w", err), texp.Shutdown(ctx))
	}
	// Sampler, BSP delay, metric interval: SDK đọc từ env; temporality mặc
	// định cumulative (Prometheus OTLP receiver cần cumulative).
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(texp), sdktrace.WithResource(res))
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(mexp)),
		sdkmetric.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	return shutdownOnce(tp, mp), nil
}

// provider là phần của TracerProvider/MeterProvider SDK mà shutdown cần.
type provider interface {
	Shutdown(context.Context) error
}

// shutdownOnce dừng các provider SONG SONG trong cùng ctx: provider trace
// không ăn hết hạn của provider metric. Chỉ chạy một lần.
func shutdownOnce(ps ...provider) ShutdownFunc {
	var (
		once sync.Once
		err  error
	)
	return func(ctx context.Context) error {
		once.Do(func() {
			errs := make([]error, len(ps))
			var wg sync.WaitGroup
			for i, p := range ps {
				wg.Go(func() { errs[i] = p.Shutdown(ctx) })
			}
			wg.Wait()
			err = errors.Join(errs...)
		})
		return err
	}
}

// newResource: giá trị Config trước, rồi thuộc tính SDK, rồi env — detector
// sau đè detector trước, nên env thắng giá trị mặc định của code.
func newResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	var attrs []attribute.KeyValue
	if cfg.ServiceName != "" {
		attrs = append(attrs, semconv.ServiceName(cfg.ServiceName))
	}
	if cfg.ServiceNamespace != "" {
		attrs = append(attrs, semconv.ServiceNamespace(cfg.ServiceNamespace))
	}
	if cfg.ServiceVersion != "" {
		attrs = append(attrs, semconv.ServiceVersion(cfg.ServiceVersion))
	}
	return resource.New(ctx,
		resource.WithAttributes(attrs...),
		resource.WithTelemetrySDK(),
		resource.WithFromEnv(),
	)
}

// validateEndpoints kiểm các biến endpoint OTLP (nếu đặt) là URL http/https
// có host. SDK gặp giá trị sai sẽ chỉ log rồi dùng endpoint mặc định — sai
// cấu hình bị che đi, nên otelx báo lỗi khởi động thay vì để vậy.
func validateEndpoints(lookup func(string) (string, bool)) error {
	var errs []error
	for _, key := range []string{EnvEndpoint, EnvTracesEndpoint, EnvMetricsEndpoint} {
		v, ok := lookup(key)
		if !ok || strings.TrimSpace(v) == "" {
			continue
		}
		u, err := url.Parse(strings.TrimSpace(v))
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("%s: %q không phải URL hợp lệ: %w", key, v, err))
		case u.Scheme != "http" && u.Scheme != "https":
			errs = append(errs, fmt.Errorf("%s: %q phải là URL http:// hoặc https://", key, v))
		case u.Host == "":
			errs = append(errs, fmt.Errorf("%s: %q thiếu host", key, v))
		}
	}
	return errors.Join(errs...)
}

// endpointSet: có endpoint từ env cho signal (biến chung hoặc biến riêng
// signalKey) hay không.
func endpointSet(lookup func(string) (string, bool), signalKey string) bool {
	for _, key := range []string{EnvEndpoint, signalKey} {
		if v, ok := lookup(key); ok && strings.TrimSpace(v) != "" {
			return true
		}
	}
	return false
}

// sdkDisabled: OTEL_SDK_DISABLED=true (không phân biệt hoa thường) theo spec
// cấu hình chung của OTel; giá trị khác hoặc không đặt → bật.
func sdkDisabled(lookup func(string) (string, bool)) bool {
	v, _ := lookup(EnvSDKDisabled)
	return strings.EqualFold(strings.TrimSpace(v), "true")
}
