import { context, diag, metrics, propagation, trace } from '@opentelemetry/api';
import { InMemorySpanExporter, SimpleSpanProcessor } from '@opentelemetry/sdk-trace-base';
import type { NodeSDK } from '@opentelemetry/sdk-node';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { createLogger } from '../logger.js';
import {
  bffInstrumentations,
  createOtelSdk,
  metricExportIntervalMs,
  otelDisabled,
  otelEnvProblems,
  pinoDiagLogger,
} from './otel.js';

type LogLine = Record<string, unknown>;

function capture() {
  const lines: LogLine[] = [];
  const logger = createLogger('info', {
    write: (s: string) => {
      lines.push(JSON.parse(s) as LogLine);
    },
  });
  return { logger, lines };
}

let running: NodeSDK | undefined;

afterEach(async () => {
  await running?.shutdown();
  running = undefined;
  // Gỡ provider/propagator/context manager global để test sau đăng ký lại được.
  trace.disable();
  metrics.disable();
  propagation.disable();
  context.disable();
  diag.disable();
  vi.unstubAllEnvs();
});

/** Start SDK (không instrumentation) với exporter in-memory, tạo 1 span, trả resource của span. */
async function resourceOfOneSpan(): Promise<Record<string, unknown>> {
  const exporter = new InMemorySpanExporter();
  const processor = new SimpleSpanProcessor(exporter);
  const { logger } = capture();
  running = createOtelSdk({
    env: process.env,
    logger,
    spanProcessors: [processor],
    metricReaders: [],
    instrumentations: [],
  });
  expect(running).toBeDefined();
  running?.start();
  trace.getTracer('test').startSpan('probe').end();
  // host detector có thuộc tính async → SimpleSpanProcessor export muộn; chờ xong.
  await processor.forceFlush();
  const [span] = exporter.getFinishedSpans();
  expect(span).toBeDefined();
  await span?.resource.waitForAsyncAttributes?.();
  return { ...span?.resource.attributes };
}

describe('otelEnvProblems (OTEL_EXPORTER_OTLP_ENDPOINT)', () => {
  it('không đặt hoặc rỗng → không lỗi (dùng mặc định http://localhost:4318)', () => {
    expect(otelEnvProblems({})).toEqual([]);
    expect(otelEnvProblems({ OTEL_EXPORTER_OTLP_ENDPOINT: '' })).toEqual([]);
    expect(otelEnvProblems({ OTEL_EXPORTER_OTLP_ENDPOINT: '  ' })).toEqual([]);
  });

  it.each(['http://localhost:4318', 'https://collector.example:4318/', 'http://otel-collector:4318'])(
    'OTEL_EXPORTER_OTLP_ENDPOINT=%s hợp lệ',
    (v) => {
      expect(otelEnvProblems({ OTEL_EXPORTER_OTLP_ENDPOINT: v })).toEqual([]);
    },
  );

  it.each(['localhost:4318', 'ftp://x:1', 'http://', 'khong-phai-url'])(
    'OTEL_EXPORTER_OTLP_ENDPOINT=%j sai → lỗi nêu tên biến, không chứa giá trị',
    (v) => {
      const p = otelEnvProblems({ OTEL_EXPORTER_OTLP_ENDPOINT: v });
      expect(p.map((x) => x.name)).toEqual(['OTEL_EXPORTER_OTLP_ENDPOINT']);
      // Chỉ có tên + lý do cố định, không có trường nào mang giá trị.
      expect(Object.keys(p[0] ?? {}).sort()).toEqual(['name', 'reason']);
      expect(p[0]?.reason).toMatch(/^must (be an http:\/\/ or https:\/\/ URL|have a host|not contain credentials)$/);
    },
  );

  it('kiểm cả OTEL_EXPORTER_OTLP_TRACES_ENDPOINT và OTEL_EXPORTER_OTLP_METRICS_ENDPOINT', () => {
    const p = otelEnvProblems({
      OTEL_EXPORTER_OTLP_TRACES_ENDPOINT: 'grpc://x:4317',
      OTEL_EXPORTER_OTLP_METRICS_ENDPOINT: 'localhost:4318',
    });
    expect(p.map((x) => x.name)).toEqual(['OTEL_EXPORTER_OTLP_TRACES_ENDPOINT', 'OTEL_EXPORTER_OTLP_METRICS_ENDPOINT']);
  });
});

describe('otelDisabled, metricExportIntervalMs', () => {
  it.each([
    ['true', true],
    ['TRUE', true],
    [' true ', true],
    ['false', false],
    ['1', false],
    ['', false],
  ])('OTEL_SDK_DISABLED=%j → %s', (v, want) => {
    expect(otelDisabled({ OTEL_SDK_DISABLED: v })).toBe(want);
  });

  it('OTEL_METRIC_EXPORT_INTERVAL: số nguyên dương được dùng, thiếu/sai → 60000', () => {
    expect(metricExportIntervalMs({ OTEL_METRIC_EXPORT_INTERVAL: '2000' })).toBe(2000);
    for (const v of [undefined, '', 'abc', '0', '-5', '1.5']) {
      expect(metricExportIntervalMs({ OTEL_METRIC_EXPORT_INTERVAL: v })).toBe(60000);
    }
  });
});

describe('resource', () => {
  it('resource mặc định service.name=bff, service.namespace=snaptix, telemetry.sdk.language=nodejs', async () => {
    vi.stubEnv('OTEL_SERVICE_NAME', '');
    vi.stubEnv('OTEL_RESOURCE_ATTRIBUTES', '');
    const attrs = await resourceOfOneSpan();
    expect(attrs['service.name']).toBe('bff');
    expect(attrs['service.namespace']).toBe('snaptix');
    expect(attrs['telemetry.sdk.language']).toBe('nodejs');
  });

  it('OTEL_SERVICE_NAME đè service.name, service.namespace mặc định vẫn giữ; OTEL_RESOURCE_ATTRIBUTES thêm khoá', async () => {
    vi.stubEnv('OTEL_SERVICE_NAME', 'bff-tc');
    vi.stubEnv('OTEL_RESOURCE_ATTRIBUTES', 'deployment.environment.name=tc');
    const attrs = await resourceOfOneSpan();
    expect(attrs['service.name']).toBe('bff-tc');
    expect(attrs['service.namespace']).toBe('snaptix');
    expect(attrs['deployment.environment.name']).toBe('tc');
  });

  it('OTEL_RESOURCE_ATTRIBUTES đè được service.namespace', async () => {
    vi.stubEnv('OTEL_SERVICE_NAME', '');
    vi.stubEnv('OTEL_RESOURCE_ATTRIBUTES', 'service.namespace=khac');
    const attrs = await resourceOfOneSpan();
    expect(attrs['service.name']).toBe('bff');
    expect(attrs['service.namespace']).toBe('khac');
  });
});

describe('OTEL_SDK_DISABLED', () => {
  it('OTEL_SDK_DISABLED=true → không tạo SDK, log info JSON, tracer global là no-op', () => {
    const { logger, lines } = capture();
    const sdk = createOtelSdk({ env: { OTEL_SDK_DISABLED: 'true' }, logger });
    expect(sdk).toBeUndefined();
    expect(lines).toHaveLength(1);
    expect(lines[0]).toMatchObject({ level: 'info', msg: expect.stringContaining('OTEL_SDK_DISABLED') });
    const span = trace.getTracer('test').startSpan('x');
    expect(span.isRecording()).toBe(false);
    span.end();
  });
});

describe('diag → pino', () => {
  it('lỗi nội bộ OTel (diag) thành dòng log JSON component=otel, không có stack thô', () => {
    const { logger, lines } = capture();
    const d = pinoDiagLogger(logger);
    const e = new Error('connect ECONNREFUSED 127.0.0.1:4318');
    d.error('export failed', e);
    d.warn('something');
    expect(lines).toHaveLength(2);
    expect(lines[0]).toMatchObject({
      level: 'error',
      component: 'otel',
      msg: 'otel: export failed',
      otel_args: ['Error: connect ECONNREFUSED 127.0.0.1:4318'],
    });
    expect(lines[1]).toMatchObject({ level: 'warn', msg: 'otel: something' });
    expect(JSON.stringify(lines)).not.toContain('    at ');
  });

  it('createOtelSdk gắn diag của OTel vào pino ở mức warn (debug/info bị bỏ)', () => {
    const { logger, lines } = capture();
    running = createOtelSdk({ env: {}, logger, spanProcessors: [], metricReaders: [], instrumentations: [] });
    diag.debug('bo qua');
    diag.info('bo qua');
    diag.warn('canh bao');
    expect(lines.map((l) => l.msg)).toEqual(['otel: canh bao']);
  });
});

describe('bffInstrumentations', () => {
  it('chỉ bật http, undici, @fastify/otel (không instrumentation-fastify deprecated, không auto-instrumentations)', () => {
    const names = bffInstrumentations()
      .flat()
      .map((i) => (i as { instrumentationName: string }).instrumentationName)
      .sort();
    expect(names).toEqual(['@fastify/otel', '@opentelemetry/instrumentation-http', '@opentelemetry/instrumentation-undici']);
    for (const i of bffInstrumentations().flat()) (i as { disable(): void }).disable();
  });
});
