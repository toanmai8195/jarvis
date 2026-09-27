import { createServer, type IncomingMessage, type Server, type ServerResponse } from 'node:http';
import type { AddressInfo } from 'node:net';

import { afterEach, describe, expect, it } from 'vitest';

import { CoreStatusError, createCoreClient } from './client.js';

interface Hit {
  method?: string;
  url?: string;
}

const servers: Server[] = [];

/** Core giả trên 127.0.0.1:0; `handler` quyết định trả gì (hoặc không trả). */
async function fakeCore(handler: (req: IncomingMessage, res: ServerResponse) => void) {
  const hits: Hit[] = [];
  const srv = createServer((req, res) => {
    hits.push({ method: req.method, url: req.url });
    handler(req, res);
  });
  servers.push(srv);
  await new Promise<void>((r) => srv.listen(0, '127.0.0.1', r));
  const { port } = srv.address() as AddressInfo;
  return { base: `http://127.0.0.1:${port}`, hits };
}

afterEach(async () => {
  for (const s of servers.splice(0)) {
    s.closeAllConnections();
    await new Promise((r) => s.close(r));
  }
});

describe('createCoreClient.ready', () => {
  it('core 200 → resolve, gọi đúng 1 lần GET /readyz', async () => {
    const core = await fakeCore((_, res) => res.writeHead(200, { 'content-type': 'application/json' }).end('{"status":"ok"}'));
    await expect(createCoreClient({ baseUrl: core.base }).ready(2000)).resolves.toBeUndefined();
    expect(core.hits).toEqual([{ method: 'GET', url: '/readyz' }]);
  });

  it('base URL có "/" cuối → vẫn gọi /readyz (không "//readyz")', async () => {
    const core = await fakeCore((_, res) => res.writeHead(200).end());
    await createCoreClient({ baseUrl: `${core.base}/` }).ready(2000);
    expect(core.hits.map((h) => h.url)).toEqual(['/readyz']);
  });

  it('core 503 → reject CoreStatusError status 503, không gọi lại', async () => {
    const core = await fakeCore((_, res) => res.writeHead(503).end('{"status":"unavailable"}'));
    const err = await createCoreClient({ baseUrl: core.base })
      .ready(2000)
      .catch((e: unknown) => e);
    expect(err).toBeInstanceOf(CoreStatusError);
    expect((err as CoreStatusError).status).toBe(503);
    expect((err as Error).message).toBe('core /readyz returned 503');
    expect(core.hits).toHaveLength(1);
  });

  it('core không chạy (từ chối kết nối) → reject nhanh', async () => {
    const core = await fakeCore((_, res) => res.end());
    const base = core.base;
    const s = servers.pop();
    await new Promise((r) => s?.close(r));
    const t0 = performance.now();
    await expect(createCoreClient({ baseUrl: base }).ready(2000)).rejects.toThrow('fetch failed');
    expect(performance.now() - t0).toBeLessThan(1000);
  });

  it('core treo (không trả lời) → reject TimeoutError trong timeout', async () => {
    const core = await fakeCore(() => {});
    const t0 = performance.now();
    const err = await createCoreClient({ baseUrl: core.base })
      .ready(100)
      .catch((e: unknown) => e);
    const elapsed = performance.now() - t0;
    expect((err as Error).name).toBe('TimeoutError');
    expect(elapsed).toBeGreaterThanOrEqual(90);
    expect(elapsed).toBeLessThan(1000);
    expect(core.hits).toHaveLength(1);
  });

  it('dùng fetch truyền vào (không phải global) khi có option fetch', async () => {
    const urls: string[] = [];
    const fake = (async (input: string | URL | Request) => {
      urls.push(String(input));
      return new Response('{}', { status: 200 });
    }) as typeof fetch;
    await createCoreClient({ baseUrl: 'http://core.test:8080', fetch: fake }).ready(2000);
    expect(urls).toEqual(['http://core.test:8080/readyz']);
  });
});
