import type { FastifyPluginAsync } from 'fastify';

/** Hạn của /readyz, giống core (ReadyTimeout = 2 s). */
export const READY_TIMEOUT_MS = 2000;

/** Thứ /readyz cần: ping được DB trong hạn. Khai báo ở phía dùng. */
export interface Pinger {
  ping(timeoutMs: number): Promise<void>;
}

export interface HealthOptions {
  pinger: Pinger;
  readyTimeoutMs?: number;
}

export class TimeoutError extends Error {
  constructor(ms: number) {
    super(`timeout after ${ms}ms`);
    this.name = 'TimeoutError';
  }
}

/**
 * Chờ `fn()` tối đa `ms`. Promise.race đã gắn handler cho promise gốc, nên nó
 * reject muộn cũng không thành unhandled rejection. Lỗi ném đồng bộ trong `fn`
 * cũng thành reject.
 */
export async function withTimeout<T>(fn: () => Promise<T>, ms: number): Promise<T> {
  let timer: NodeJS.Timeout | undefined;
  const timeout = new Promise<never>((_, reject) => {
    timer = setTimeout(() => reject(new TimeoutError(ms)), ms);
  });
  try {
    return await Promise.race([Promise.resolve().then(fn), timeout]);
  } finally {
    clearTimeout(timer);
  }
}

const statusSchema = {
  type: 'object',
  properties: { status: { type: 'string' } },
  required: ['status'],
} as const;

export const healthRoutes: FastifyPluginAsync<HealthOptions> = async (app, opts) => {
  const timeoutMs = opts.readyTimeoutMs ?? READY_TIMEOUT_MS;

  // Liveness: process còn phục vụ được, không chạm DB.
  app.get('/healthz', { schema: { response: { 200: statusSchema } } }, async () => ({ status: 'ok' }));

  // Readiness: ping được MongoDB trong hạn.
  app.get(
    '/readyz',
    { schema: { response: { 200: statusSchema, 503: statusSchema } } },
    async (req, reply) => {
      try {
        await withTimeout(() => opts.pinger.ping(timeoutMs), timeoutMs);
        return { status: 'ok' };
      } catch (err) {
        // Chỉ ghi tên + message: object lỗi của driver có thể mang cả topology.
        // Không dùng khoá `err` (serializer err của pino sẽ ghi đè thành type/stack).
        const e = err instanceof Error ? err : new Error(String(err));
        req.log.warn({ error_type: e.name, error: e.message }, 'readyz: database unavailable');
        return reply.code(503).send({ status: 'unavailable' });
      }
    },
  );
};
