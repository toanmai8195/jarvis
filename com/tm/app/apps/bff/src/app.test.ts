import { afterEach, describe, expect, it } from 'vitest';

import { buildApp } from './app.js';
import { createLogger } from './logger.js';
import { loadConfig } from './plugins/config.js';
import type { CoreReadiness, Pinger } from './routes/health.js';

const UUID7 = /^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const config = loadConfig({ MONGODB_URI: 'mongodb://tcuser:tc-pw-SECRET@127.0.0.1:1/snaptix' });

type LogLine = Record<string, unknown>;

/** Pinger giả đếm số lần gọi; `impl` quyết định ok / lỗi / treo. */
function fakePinger(impl: () => Promise<void> = async () => {}): Pinger & { calls: number } {
  const p = {
    calls: 0,
    ping: () => {
      p.calls++;
      return impl();
    },
  };
  return p;
}

const apps: Array<Awaited<ReturnType<typeof buildApp>>> = [];

/** Core giả đếm số lần gọi `ready`; `impl` quyết định ok / lỗi / treo. */
function fakeCore(impl: () => Promise<void> = async () => {}): CoreReadiness & { calls: number; timeouts: number[] } {
  const c = {
    calls: 0,
    timeouts: [] as number[],
    ready: (ms: number) => {
      c.calls++;
      c.timeouts.push(ms);
      return impl();
    },
  };
  return c;
}

async function setup(
  pinger: Pinger,
  readyTimeoutMs?: number,
  extra: { core?: CoreReadiness; coreTimeoutMs?: number } = {},
) {
  const lines: LogLine[] = [];
  const logger = createLogger('info', {
    write: (s: string) => {
      lines.push(JSON.parse(s) as LogLine);
    },
  });
  const app = await buildApp({ config, logger, pinger, readyTimeoutMs, core: extra.core ?? fakeCore(), coreTimeoutMs: extra.coreTimeoutMs });
  apps.push(app);
  return { app, lines };
}

afterEach(async () => {
  await Promise.all(apps.splice(0).map((a) => a.close()));
});

describe('/healthz', () => {
  it('/healthz trả 200 {"status":"ok"} và không gọi pinger', async () => {
    const pinger = fakePinger();
    const { app } = await setup(pinger);
    const res = await app.inject({ method: 'GET', url: '/healthz' });
    expect(res.statusCode).toBe(200);
    expect(res.headers['content-type']).toMatch(/^application\/json/);
    expect(res.json()).toEqual({ status: 'ok' });
    expect(pinger.calls).toBe(0);
  });

  it('HEAD /healthz 200, POST /healthz 404', async () => {
    const { app } = await setup(fakePinger());
    expect((await app.inject({ method: 'HEAD', url: '/healthz' })).statusCode).toBe(200);
    expect((await app.inject({ method: 'POST', url: '/healthz' })).statusCode).toBe(404);
  });
});

describe('/healthz?deep=1 (gọi core)', () => {
  it('không deep, deep=0, deep=true, x=1 → 200 {"status":"ok"}, không gọi core', async () => {
    const core = fakeCore();
    const { app } = await setup(fakePinger(), undefined, { core });
    for (const url of ['/healthz', '/healthz?deep=0', '/healthz?deep=true', '/healthz?x=1', '/healthz?deep=']) {
      const res = await app.inject({ method: 'GET', url });
      expect(res.statusCode, url).toBe(200);
      expect(res.json(), url).toEqual({ status: 'ok' });
    }
    expect(core.calls).toBe(0);
  });

  it('deep=1, core ok → 200 {"status":"ok","core":"ok"}, gọi core đúng 1 lần với timeout 2000 ms mặc định', async () => {
    const core = fakeCore();
    const pinger = fakePinger();
    const { app } = await setup(pinger, undefined, { core });
    const res = await app.inject({ method: 'GET', url: '/healthz?deep=1' });
    expect(res.statusCode).toBe(200);
    expect(res.json()).toEqual({ status: 'ok', core: 'ok' });
    expect(core.calls).toBe(1);
    expect(core.timeouts).toEqual([2000]);
    expect(pinger.calls).toBe(0);
  });

  it('deep=1, core lỗi → 503 {"status":"unavailable","core":"unavailable"}, gọi 1 lần, log warn có request_id + loại lỗi', async () => {
    const core = fakeCore(async () => {
      throw new TypeError('fetch failed', { cause: Object.assign(new Error('connect ECONNREFUSED'), { code: 'ECONNREFUSED' }) });
    });
    const { app, lines } = await setup(fakePinger(), undefined, { core });
    const res = await app.inject({ method: 'GET', url: '/healthz?deep=1', headers: { 'x-request-id': 'deep-503' } });
    expect(res.statusCode).toBe(503);
    expect(res.json()).toEqual({ status: 'unavailable', core: 'unavailable' });
    expect(res.headers['x-request-id']).toBe('deep-503');
    expect(core.calls).toBe(1);
    const warn = lines.find((l) => l.level === 'warn');
    expect(warn).toMatchObject({
      request_id: 'deep-503',
      msg: 'healthz: core unavailable',
      error_type: 'TypeError',
      error: 'fetch failed',
      error_cause: 'ECONNREFUSED',
    });
    expect(JSON.stringify(warn)).not.toMatch(/"stack"/);
  });

  it('deep=1, core treo → 503 trong timeout ngắn truyền vào, không chờ core', async () => {
    const core = fakeCore(() => new Promise<void>(() => {}));
    const { app } = await setup(fakePinger(), undefined, { core, coreTimeoutMs: 50 });
    const t0 = performance.now();
    const res = await app.inject({ method: 'GET', url: '/healthz?deep=1' });
    const elapsed = performance.now() - t0;
    expect(res.statusCode).toBe(503);
    expect(res.json()).toEqual({ status: 'unavailable', core: 'unavailable' });
    expect(elapsed).toBeGreaterThanOrEqual(45);
    expect(elapsed).toBeLessThan(1000);
    expect(core.timeouts).toEqual([50]);
  });

  it('/readyz không gọi core (chỉ ping MongoDB)', async () => {
    const core = fakeCore();
    const { app } = await setup(fakePinger(), undefined, { core });
    expect((await app.inject({ method: 'GET', url: '/readyz' })).statusCode).toBe(200);
    expect(core.calls).toBe(0);
  });
});

describe('/readyz', () => {
  it('/readyz trả 200 {"status":"ok"} khi pinger ok', async () => {
    const pinger = fakePinger();
    const { app } = await setup(pinger);
    const res = await app.inject({ method: 'GET', url: '/readyz' });
    expect(res.statusCode).toBe(200);
    expect(res.json()).toEqual({ status: 'ok' });
    expect(pinger.calls).toBe(1);
  });

  it('/readyz trả 503 {"status":"unavailable"} khi pinger lỗi, log warn có request_id, không lộ mật khẩu', async () => {
    const { app, lines } = await setup(
      fakePinger(async () => {
        throw new Error('connect ECONNREFUSED 127.0.0.1:1');
      }),
    );
    const res = await app.inject({ method: 'GET', url: '/readyz', headers: { 'x-request-id': 'rz-1' } });
    expect(res.statusCode).toBe(503);
    expect(res.json()).toEqual({ status: 'unavailable' });
    const warn = lines.find((l) => l.level === 'warn');
    expect(warn).toMatchObject({ request_id: 'rz-1', msg: 'readyz: database unavailable' });
    expect(JSON.stringify(lines)).not.toContain('tc-pw-SECRET');
  });

  it('/readyz trả 503 khi pinger ném lỗi đồng bộ', async () => {
    const { app } = await setup({
      ping: () => {
        throw new Error('boom');
      },
    });
    expect((await app.inject({ method: 'GET', url: '/readyz' })).statusCode).toBe(503);
  });

  it('/readyz trả 503 trong hạn khi pinger treo quá timeout', async () => {
    const { app } = await setup(
      fakePinger(() => new Promise<void>(() => {})),
      50,
    );
    const t0 = performance.now();
    const res = await app.inject({ method: 'GET', url: '/readyz' });
    const elapsed = performance.now() - t0;
    expect(res.statusCode).toBe(503);
    expect(res.json()).toEqual({ status: 'unavailable' });
    expect(elapsed).toBeGreaterThanOrEqual(45);
    expect(elapsed).toBeLessThan(1000);
  });

  it('/readyz: pinger reject muộn sau timeout không gây unhandled rejection', async () => {
    const unhandled: unknown[] = [];
    const onUnhandled = (r: unknown) => unhandled.push(r);
    process.on('unhandledRejection', onUnhandled);
    try {
      const { app } = await setup(
        fakePinger(() => new Promise<void>((_, reject) => setTimeout(() => reject(new Error('late')), 80))),
        20,
      );
      expect((await app.inject({ method: 'GET', url: '/readyz' })).statusCode).toBe(503);
      await new Promise((r) => setTimeout(r, 150));
      expect(unhandled).toEqual([]);
    } finally {
      process.off('unhandledRejection', onUnhandled);
    }
  });
});

describe('X-Request-ID và log', () => {
  it('X-Request-ID hợp lệ được trả lại nguyên văn và làm request_id trong log', async () => {
    const { app, lines } = await setup(fakePinger());
    const res = await app.inject({ method: 'GET', url: '/healthz', headers: { 'x-request-id': 'tc12.ok_A:1-z' } });
    expect(res.headers['x-request-id']).toBe('tc12.ok_A:1-z');
    const mine = lines.filter((l) => l.request_id === 'tc12.ok_A:1-z').map((l) => l.msg);
    expect(mine).toEqual(['incoming request', 'request completed']);
    expect(lines.some((l) => 'reqId' in l)).toBe(false);
  });

  it.each([
    ['sai ký tự', 'bad id!'],
    ['dài > 128', 'a'.repeat(129)],
  ])('X-Request-ID %s → UUID v7 (không 400)', async (_, h) => {
    const { app } = await setup(fakePinger());
    const res = await app.inject({ method: 'GET', url: '/healthz', headers: { 'x-request-id': h } });
    expect(res.statusCode).toBe(200);
    expect(res.headers['x-request-id']).toMatch(UUID7);
  });

  it('thiếu X-Request-ID → mỗi request một UUID v7 khác nhau', async () => {
    const { app } = await setup(fakePinger());
    const a = (await app.inject({ method: 'GET', url: '/healthz' })).headers['x-request-id'];
    const b = (await app.inject({ method: 'GET', url: '/healthz' })).headers['x-request-id'];
    expect(a).toMatch(UUID7);
    expect(b).toMatch(UUID7);
    expect(a).not.toBe(b);
  });

  it('route lạ trả 404 JSON và vẫn có header X-Request-ID', async () => {
    const { app, lines } = await setup(fakePinger());
    const res = await app.inject({ method: 'GET', url: '/khong-co', headers: { 'x-request-id': 'tc12-404' } });
    expect(res.statusCode).toBe(404);
    expect(typeof res.json()).toBe('object');
    expect(res.headers['x-request-id']).toBe('tc12-404');
    const done = lines.find((l) => l.request_id === 'tc12-404' && l.msg === 'request completed');
    expect(done).toMatchObject({ res: { statusCode: 404 } });
  });

  it('503 của /readyz cũng có header X-Request-ID', async () => {
    const { app } = await setup(
      fakePinger(async () => {
        throw new Error('down');
      }),
    );
    const res = await app.inject({ method: 'GET', url: '/readyz', headers: { 'x-request-id': 'rz-503' } });
    expect(res.statusCode).toBe(503);
    expect(res.headers['x-request-id']).toBe('rz-503');
  });

  it('log JSON: time ISO 8601, level là nhãn chữ, có msg; không log header nhạy cảm', async () => {
    const { app, lines } = await setup(fakePinger());
    await app.inject({
      method: 'GET',
      url: '/healthz',
      headers: { authorization: 'Bearer tok-SECRET', cookie: 'sid=ck-SECRET' },
    });
    expect(lines.length).toBeGreaterThan(0);
    for (const l of lines) {
      expect(typeof l.time).toBe('string');
      expect(l.time).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z$/);
      expect(['trace', 'debug', 'info', 'warn', 'error', 'fatal']).toContain(l.level);
      expect(typeof l.msg).toBe('string');
    }
    const text = JSON.stringify(lines);
    expect(text).not.toContain('SECRET');
    expect(text).not.toMatch(/"headers"/);
  });

  it('app.config được decorate từ config đã validate', async () => {
    const { app } = await setup(fakePinger());
    expect(app.config.port).toBe(3000);
  });
});
