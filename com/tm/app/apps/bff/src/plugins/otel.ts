/**
 * Dựng OpenTelemetry SDK của bff: trace + metric export OTLP/HTTP (protobuf)
 * tới otel-collector, propagator W3C TraceContext + Baggage (mặc định của
 * NodeSDK), resource mặc định `service.name=bff`, `service.namespace=snaptix`.
 *
 * Chỉ `src/instrumentation.ts` (nạp bằng `node --import`) gọi module này, TRƯỚC
 * khi fastify/node:http được nạp. `server.ts`/`app.ts` không import nó.
 *
 * Cấu hình bằng env chuẩn `OTEL_*` mà SDK tự đọc (OTEL_SERVICE_NAME,
 * OTEL_RESOURCE_ATTRIBUTES, OTEL_TRACES_SAMPLER[_ARG], OTEL_BSP_SCHEDULE_DELAY,
 * OTEL_EXPORTER_OTLP_TIMEOUT...). Module này chỉ tự xử lý: kiểm định dạng
 * endpoint, OTEL_SDK_DISABLED, OTEL_METRIC_EXPORT_INTERVAL, và đưa log nội bộ
 * của OTel (diag) về pino JSON.
 */
import { FastifyOtelInstrumentation } from '@fastify/otel';
import { diag, DiagLogLevel, type DiagLogger } from '@opentelemetry/api';
import { OTLPMetricExporter } from '@opentelemetry/exporter-metrics-otlp-proto';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-proto';
import { HttpInstrumentation } from '@opentelemetry/instrumentation-http';
import { UndiciInstrumentation } from '@opentelemetry/instrumentation-undici';
import { defaultResource, resourceFromAttributes, type Resource } from '@opentelemetry/resources';
import { PeriodicExportingMetricReader, type IMetricReader } from '@opentelemetry/sdk-metrics';
import { NodeSDK, type NodeSDKConfiguration } from '@opentelemetry/sdk-node';
import type { SpanProcessor } from '@opentelemetry/sdk-trace-base';
import type { Logger } from 'pino';

import { httpUrlProblem, type ConfigProblem } from './config.js';

export const DEFAULT_SERVICE_NAME = 'bff';
export const DEFAULT_SERVICE_NAMESPACE = 'snaptix';

/** Biến endpoint OTLP được kiểm định dạng (nếu đặt). */
export const OTLP_ENDPOINT_VARS = [
  'OTEL_EXPORTER_OTLP_ENDPOINT',
  'OTEL_EXPORTER_OTLP_TRACES_ENDPOINT',
  'OTEL_EXPORTER_OTLP_METRICS_ENDPOINT',
] as const;

/** Mặc định của spec OTel cho chu kỳ / hạn export metric (ms). */
export const DEFAULT_METRIC_EXPORT_INTERVAL_MS = 60_000;
export const DEFAULT_METRIC_EXPORT_TIMEOUT_MS = 30_000;

type Env = Readonly<Record<string, string | undefined>>;

/**
 * Lỗi định dạng của các biến endpoint OTLP. SDK gặp giá trị sai chỉ log rồi
 * dùng endpoint khác, che mất lỗi cấu hình — nên bff từ chối khởi động (giống
 * `pkg/otelx` của core). Chỉ nêu tên biến, không nêu giá trị.
 */
export function otelEnvProblems(env: Env): ConfigProblem[] {
  const problems: ConfigProblem[] = [];
  for (const name of OTLP_ENDPOINT_VARS) {
    const v = (env[name] ?? '').trim();
    if (v === '') continue;
    const reason = httpUrlProblem(v);
    if (reason) problems.push({ name, reason });
  }
  return problems;
}

/** `OTEL_SDK_DISABLED=true` (không phân biệt hoa thường, như spec). */
export function otelDisabled(env: Env): boolean {
  return (env.OTEL_SDK_DISABLED ?? '').trim().toLowerCase() === 'true';
}

/** OTEL_METRIC_EXPORT_INTERVAL (ms, số nguyên dương), sai/thiếu → mặc định 60 s. */
export function metricExportIntervalMs(env: Env): number {
  const raw = (env.OTEL_METRIC_EXPORT_INTERVAL ?? '').trim();
  const n = /^\d+$/.test(raw) ? Number(raw) : NaN;
  return Number.isSafeInteger(n) && n > 0 ? n : DEFAULT_METRIC_EXPORT_INTERVAL_MS;
}

/**
 * Resource mặc định do code đặt. NodeSDK merge thêm env detector SAU nên
 * OTEL_RESOURCE_ATTRIBUTES / OTEL_SERVICE_NAME đè được từng khoá.
 */
export function defaultBffResource(): Resource {
  return defaultResource().merge(
    resourceFromAttributes({
      'service.name': DEFAULT_SERVICE_NAME,
      'service.namespace': DEFAULT_SERVICE_NAMESPACE,
    }),
  );
}

/** diag của OTel (lỗi export, cảnh báo nội bộ) → pino JSON, không in text ra stderr. */
export function pinoDiagLogger(log: Logger): DiagLogger {
  const at =
    (level: 'error' | 'warn' | 'info' | 'debug' | 'trace') =>
    (message: string, ...args: unknown[]) => {
      const fields = args.length > 0 ? { otel_args: args.map((a) => (a instanceof Error ? `${a.name}: ${a.message}` : String(a))) } : {};
      log[level]({ component: 'otel', ...fields }, `otel: ${message}`);
    };
  return { error: at('error'), warn: at('warn'), info: at('info'), debug: at('debug'), verbose: at('trace') };
}

/** Instrumentation bff dùng — chỉ những gì cần, không auto-instrumentations. */
export function bffInstrumentations(): NonNullable<NodeSDKConfiguration['instrumentations']> {
  return [
    // Span SERVER cho mọi request vào (đọc `traceparent`), metric
    // http.server.request.duration (giây). Outbound của bff đi bằng fetch
    // (undici) nên tắt phần client của node:http.
    new HttpInstrumentation({ disableOutgoingRequestInstrumentation: true }),
    // Span CLIENT + inject `traceparent` cho fetch global / undici.
    new UndiciInstrumentation(),
    // Span INTERNAL theo route/hook của Fastify; cấp `http.route` cho span
    // SERVER và metric của instrumentation-http (qua RPC metadata). Tự gắn
    // vào mọi instance Fastify qua diagnostics_channel `fastify.initialization`.
    new FastifyOtelInstrumentation({ registerOnInitialization: true }),
  ];
}

export interface OtelSdkOptions {
  env: Env;
  logger: Logger;
  /** Thay span processor (test: SimpleSpanProcessor + InMemorySpanExporter). Mặc định batch → OTLP. */
  spanProcessors?: SpanProcessor[];
  /** Thay metric reader (test). Mặc định periodic → OTLP. */
  metricReaders?: IMetricReader[];
  /** Thay danh sách instrumentation (test). Mặc định bffInstrumentations(). */
  instrumentations?: NodeSDKConfiguration['instrumentations'];
}

/**
 * Tạo NodeSDK (chưa start). `OTEL_SDK_DISABLED=true` → `undefined`: không
 * export, không instrumentation, bff vẫn chạy bình thường.
 */
export function createOtelSdk(opts: OtelSdkOptions): NodeSDK | undefined {
  diag.setLogger(pinoDiagLogger(opts.logger), { logLevel: DiagLogLevel.WARN, suppressOverrideMessage: true });

  if (otelDisabled(opts.env)) {
    opts.logger.info('otel: OTEL_SDK_DISABLED=true, không export telemetry');
    return undefined;
  }

  const interval = metricExportIntervalMs(opts.env);
  const metricReaders = opts.metricReaders ?? [
    new PeriodicExportingMetricReader({
      exporter: new OTLPMetricExporter(),
      exportIntervalMillis: interval,
      exportTimeoutMillis: Math.min(interval, DEFAULT_METRIC_EXPORT_TIMEOUT_MS),
    }),
  ];

  return new NodeSDK({
    resource: defaultBffResource(),
    // Có spanProcessors thì NodeSDK bỏ qua traceExporter; không có thì bọc
    // exporter bằng BatchSpanProcessor đọc OTEL_BSP_* từ env.
    ...(opts.spanProcessors ? { spanProcessors: opts.spanProcessors } : { traceExporter: new OTLPTraceExporter() }),
    metricReaders,
    // Log đi stdout qua pino, không gửi OTLP logs.
    logRecordProcessors: [],
    instrumentations: opts.instrumentations ?? bffInstrumentations(),
  });
}
