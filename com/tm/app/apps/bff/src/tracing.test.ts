/**
 * Trace/log/metric của bff với SDK thật + exporter in-memory, trong một
 * process: không cần Docker, collector hay core thật. Core giả là
 * `http.createServer` cổng 0. Fastify được import ĐỘNG sau `sdk.start()`
 * (giống `node --import ./dist/instrumentation.js dist/server.js`).
 */
import { randomBytes } from 'node:crypto';
import { createServer, request, type IncomingHttpHeaders, type Server } from 'node:http';
import type { AddressInfo } from 'node:net';

import { SpanKind, SpanStatusCode } from '@opentelemetry/api';
import {
  AggregationTemporality,
  InMemoryMetricExporter,
  PeriodicExportingMetricReader,
  type HistogramMetricData,
} from '@opentelemetry/sdk-metrics';
import type { NodeSDK } from '@opentelemetry/sdk-node';
import { InMemorySpanExporter, SimpleSpanProcessor, type ReadableSpan } from '@opentelemetry/sdk-trace-base';
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

import { createLogger } from './logger.js';
import { loadConfig } from './plugins/config.js';
import { createOtelSdk } from './plugins/otel.js';

type LogLine = Record<string, unknown>;
type App = Awaited<ReturnType<(typeof import('./app.js'))['buildApp']>>;

const spans = new InMemorySpanExporter();
const metricExporter = new InMemoryMetricExporter(AggregationTemporality.CUMULATIVE);
const reader = new PeriodicExportingMetricReader({ exporter: metricExporter, exportIntervalMillis: 3_600_000 });
let sdk: NodeSDK | undefined;

/** Hành vi core giả của từng test. */
let coreMode: 'ok' | '503' | 'hang' = 'ok';
const coreHits: IncomingHttpHeaders[] = [];
let coreServer: Server;
let coreBase = '';

let app: App;
let bffBase = '';
const lines: LogLine[] = [];

const hex = (bytes: number) => randomBytes(bytes).toString('hex');

/** GET qua node:http (phía client không bị instrument) để tự đặt header traceparent. */
function get(path: string, headers: Record<string, string> = {}): Promise<{ status: number; body: string }> {
  return new Promise((resolve, reject) => {
    const req = request(`${bffBase}${path}`, { headers }, (res) => {
      let body = '';
      res.setEncoding('utf8');
      res.on('data', (c: string) => (body += c));
      res.on('end', () => resolve({ status: res.statusCode ?? 0, body }));
    });
    req.on('error', reject);
    req.end();
  });
}

const kindOf = (s: ReadableSpan) => s.kind;
const parentOf = (s: ReadableSpan) => s.parentSpanContext?.spanId;
const inTrace = (tid: string) => spans.getFinishedSpans().filter((s) => s.spanContext().traceId === tid);
/** Span SERVER của bff (không tính span SERVER của core giả, path /readyz). */
const bffServer = (tid: string) =>
  inTrace(tid).filter((s) => kindOf(s) === SpanKind.SERVER && s.attributes['url.path'] !== '/readyz');
const bffClient = (tid: string) => inTrace(tid).filter((s) => kindOf(s) === SpanKind.CLIENT);

/** s có phải con/cháu của span ancestorId trong cùng trace không. */
function isUnder(s: ReadableSpan, ancestorId: string): boolean {
  const all = new Map(spans.getFinishedSpans().map((x) => [x.spanContext().spanId, x]));
  let cur: ReadableSpan | undefined = s;
  for (let i = 0; cur && i < 50; i++) {
    const p = parentOf(cur);
    if (p === ancestorId) return true;
    cur = p ? all.get(p) : undefined;
  }
  return false;
}

beforeAll(async () => {
  const silent = createLogger('fatal', { write: () => {} });
  sdk = createOtelSdk({
    env: {},
    logger: silent,
    spanProcessors: [new SimpleSpanProcessor(spans)],
    metricReaders: [reader],
  });
  sdk?.start();

  coreServer = createServer((req, res) => {
    coreHits.push(req.headers);
    if (coreMode === 'hang') return;
    res.writeHead(coreMode === 'ok' ? 200 : 503, { 'content-type': 'application/json' });
    res.end(coreMode === 'ok' ? '{"status":"ok"}' : '{"status":"unavailable"}');
  });
  await new Promise<void>((r) => coreServer.listen(0, '127.0.0.1', r));
  coreBase = `http://127.0.0.1:${(coreServer.address() as AddressInfo).port}`;

  // Import động sau sdk.start(): fastify + node:http nạp khi instrumentation đã bật.
  const { buildApp } = await import('./app.js');
  const logger = createLogger('info', {
    write: (s: string) => {
      lines.push(JSON.parse(s) as LogLine);
    },
  });
  app = await buildApp({
    config: loadConfig({ MONGODB_URI: 'mongodb://127.0.0.1:1/snaptix', CORE_BASE_URL: coreBase }),
    logger,
    pinger: { ping: async () => {} },
    coreTimeoutMs: 150,
  });
  await app.listen({ port: 0, host: '127.0.0.1' });
  bffBase = `http://127.0.0.1:${(app.server.address() as AddressInfo).port}`;
});

afterAll(async () => {
  await app?.close();
  coreServer?.closeAllConnections();
  await new Promise((r) => coreServer?.close(r));
  await sdk?.shutdown();
});

beforeEach(() => {
  coreMode = 'ok';
  coreHits.length = 0;
});

describe('trace bff (SDK + InMemorySpanExporter)', () => {
  it('span server GET /healthz: đúng 1 span SERVER, có http.route, parent là span id trong traceparent', async () => {
    const tid = hex(16);
    const sid = hex(8);
    const res = await get('/healthz', { traceparent: `00-${tid}-${sid}-01` });
    expect(res.status).toBe(200);
    await vi.waitFor(() => expect(bffServer(tid)).toHaveLength(1));
    const [srv] = bffServer(tid);
    expect(srv?.name).toBe('GET /healthz');
    expect(srv?.attributes['http.route']).toBe('/healthz');
    expect(srv?.attributes['http.response.status_code']).toBe(200);
    expect(parentOf(srv!)).toBe(sid);
    expect(srv?.status.code).not.toBe(SpanStatusCode.ERROR);
    expect(bffClient(tid)).toHaveLength(0);
  });

  it('deep=1: outbound tới core gửi traceparent cùng trace ID, parent là span client (con của span server)', async () => {
    const tid = hex(16);
    const res = await get('/healthz?deep=1', { traceparent: `00-${tid}-${hex(8)}-01` });
    expect(res.status).toBe(200);
    expect(JSON.parse(res.body)).toEqual({ status: 'ok', core: 'ok' });
    expect(coreHits).toHaveLength(1);
    const tp = String(coreHits[0]?.traceparent ?? '');
    const m = /^00-([0-9a-f]{32})-([0-9a-f]{16})-01$/.exec(tp);
    expect(m, tp).not.toBeNull();
    expect(m?.[1]).toBe(tid);

    await vi.waitFor(() => {
      expect(bffServer(tid)).toHaveLength(1);
      expect(bffClient(tid)).toHaveLength(1);
    });
    const [srv] = bffServer(tid);
    const [cli] = bffClient(tid);
    expect(cli?.spanContext().spanId).toBe(m?.[2]);
    expect(isUnder(cli!, srv!.spanContext().spanId)).toBe(true);
    expect(cli?.attributes['http.request.method']).toBe('GET');
    expect(String(cli?.attributes['url.full'])).toBe(`${coreBase}/readyz`);
    expect(cli?.attributes['http.response.status_code']).toBe(200);
    expect(cli?.status.code).not.toBe(SpanStatusCode.ERROR);
  });

  it('deep=1, core trả 503 → bff 503, span server và span client status ERROR', async () => {
    coreMode = '503';
    const tid = hex(16);
    const res = await get('/healthz?deep=1', { traceparent: `00-${tid}-${hex(8)}-01` });
    expect(res.status).toBe(503);
    expect(JSON.parse(res.body)).toEqual({ status: 'unavailable', core: 'unavailable' });
    expect(coreHits).toHaveLength(1);
    await vi.waitFor(() => {
      expect(bffServer(tid)).toHaveLength(1);
      expect(bffClient(tid)).toHaveLength(1);
    });
    expect(bffServer(tid)[0]?.attributes['http.response.status_code']).toBe(503);
    expect(bffServer(tid)[0]?.status.code).toBe(SpanStatusCode.ERROR);
    expect(bffClient(tid)[0]?.attributes['http.response.status_code']).toBe(503);
    expect(bffClient(tid)[0]?.status.code).toBe(SpanStatusCode.ERROR);
  });

  it('deep=1, core treo → 503 trong timeout ngắn (150 ms), span client ERROR', async () => {
    coreMode = 'hang';
    const tid = hex(16);
    const t0 = performance.now();
    const res = await get('/healthz?deep=1', { traceparent: `00-${tid}-${hex(8)}-01` });
    const elapsed = performance.now() - t0;
    expect(res.status).toBe(503);
    expect(elapsed).toBeGreaterThanOrEqual(140);
    expect(elapsed).toBeLessThan(1500);
    await vi.waitFor(() => expect(bffClient(tid)).toHaveLength(1));
    expect(bffClient(tid)[0]?.status.code).toBe(SpanStatusCode.ERROR);
    expect(bffServer(tid)[0]?.status.code).toBe(SpanStatusCode.ERROR);
  });

  it('không có traceparent → span server là root, outbound mang trace mới của bff', async () => {
    const res = await get('/healthz?deep=1', { 'x-request-id': 'no-tp' });
    expect(res.status).toBe(200);
    const done = lines.find((l) => l.request_id === 'no-tp' && l.msg === 'request completed');
    const tid = String(done?.trace_id);
    expect(tid).toMatch(/^[0-9a-f]{32}$/);
    expect(tid).not.toBe('0'.repeat(32));
    await vi.waitFor(() => expect(bffServer(tid)).toHaveLength(1));
    expect(parentOf(bffServer(tid)[0]!)).toBeUndefined();
    expect(String(coreHits[0]?.traceparent)).toContain(`00-${tid}-`);
  });

  it('traceparent không sample (flag 00) → không export span nhưng vẫn truyền trace ID sang core và vào log', async () => {
    const tid = hex(16);
    const res = await get('/healthz?deep=1', { traceparent: `00-${tid}-${hex(8)}-00`, 'x-request-id': 'unsampled' });
    expect(res.status).toBe(200);
    expect(String(coreHits[0]?.traceparent)).toMatch(new RegExp(`^00-${tid}-[0-9a-f]{16}-00$`));
    await new Promise((r) => setTimeout(r, 50));
    expect(inTrace(tid)).toHaveLength(0);
    const mine = lines.filter((l) => l.request_id === 'unsampled');
    expect(mine.length).toBeGreaterThanOrEqual(2);
    for (const l of mine) expect(l.trace_id).toBe(tid);
  });

  it('không ghi header nhạy cảm (authorization, cookie) vào thuộc tính span', async () => {
    const tid = hex(16);
    await get('/healthz?deep=1', {
      traceparent: `00-${tid}-${hex(8)}-01`,
      authorization: 'Bearer tok-SECRET',
      cookie: 'sid=ck-SECRET',
    });
    await vi.waitFor(() => expect(bffServer(tid)).toHaveLength(1));
    const text = JSON.stringify(inTrace(tid).map((s) => s.attributes));
    expect(text).not.toContain('SECRET');
  });
});

describe('log pino có trace_id/span_id khớp span', () => {
  it('incoming request, request completed và warn của deep=1 có trace_id = trace của request, span_id là span bff trong trace', async () => {
    coreMode = '503';
    const tid = hex(16);
    await get('/healthz?deep=1', { traceparent: `00-${tid}-${hex(8)}-01`, 'x-request-id': 'log-tid' });
    const mine = lines.filter((l) => l.request_id === 'log-tid');
    expect(mine.map((l) => l.msg)).toEqual(['incoming request', 'healthz: core unavailable', 'request completed']);
    await vi.waitFor(() => expect(bffServer(tid)).toHaveLength(1));
    const bffIds = new Set(
      inTrace(tid)
        .filter((s) => s.attributes['url.path'] !== '/readyz')
        .map((s) => s.spanContext().spanId),
    );
    for (const l of mine) {
      expect(l.trace_id).toBe(tid);
      expect(l.span_id).toMatch(/^[0-9a-f]{16}$/);
      expect(bffIds.has(String(l.span_id)), String(l.span_id)).toBe(true);
    }
  });

  it('không có traceparent → mọi dòng của một request cùng một trace_id', async () => {
    await get('/healthz', { 'x-request-id': 'log-plain' });
    const ids = new Set(lines.filter((l) => l.request_id === 'log-plain').map((l) => l.trace_id));
    expect(ids.size).toBe(1);
    expect(String([...ids][0])).toMatch(/^[0-9a-f]{32}$/);
  });

  it('dòng log ngoài request không có trace_id/span_id', () => {
    const own: LogLine[] = [];
    createLogger('info', { write: (s: string) => void own.push(JSON.parse(s) as LogLine) }).info('ngoai request');
    expect(own[0]).not.toHaveProperty('trace_id');
    expect(own[0]).not.toHaveProperty('span_id');
  });
});

describe('metric http.server.request.duration', () => {
  it('histogram đơn vị giây (s) có http.route; route 404 không mang path thô', async () => {
    await get('/healthz');
    await get('/readyz');
    await get('/khong-co-123');
    await vi.waitFor(async () => {
      await reader.forceFlush();
      const found = metricExporter
        .getMetrics()
        .flatMap((rm) => rm.scopeMetrics.flatMap((sm) => sm.metrics))
        .filter((m) => m.descriptor.name === 'http.server.request.duration');
      expect(found.length).toBeGreaterThan(0);
    });
    const hist = metricExporter
      .getMetrics()
      .flatMap((rm) => rm.scopeMetrics.flatMap((sm) => sm.metrics))
      .filter((m) => m.descriptor.name === 'http.server.request.duration') as HistogramMetricData[];
    const last = hist[hist.length - 1]!;
    expect(last.descriptor.unit).toBe('s');
    const routes = new Set(last.dataPoints.map((p) => p.attributes['http.route']));
    expect(routes.has('/healthz')).toBe(true);
    expect(routes.has('/readyz')).toBe(true);
    expect([...routes].some((r) => String(r).includes('khong-co'))).toBe(false);
    const healthz = last.dataPoints.find((p) => p.attributes['http.route'] === '/healthz' && p.attributes['http.response.status_code'] === 200);
    expect(healthz?.attributes['http.request.method']).toBe('GET');
    const v = healthz!.value;
    expect(v.count).toBeGreaterThan(0);
    // Đơn vị giây: request cục bộ ≪ 1 s.
    expect(v.sum).toBeGreaterThan(0);
    expect((v.sum ?? Infinity) / v.count).toBeLessThan(1);
  });
});
