import type { FastifyPluginAsync } from 'fastify';

/** Hạn của /readyz, giống core (ReadyTimeout = 2 s). */
export const READY_TIMEOUT_MS = 2000;

/** Thứ /readyz cần: ping được DB trong hạn. Khai báo ở phía dùng. */
export interface Pinger {
  ping(timeoutMs: number): Promise<void>;
}

/** Thứ `/healthz?deep=1` cần: core sẵn sàng (core `/readyz`) trong hạn. Khai báo ở phía dùng. */
export interface CoreReadiness {
  ready(timeoutMs: number): Promise<void>;
}

export interface HealthOptions {
  pinger: Pinger;
  readyTimeoutMs?: number;
  core: CoreReadiness;
  /** Hạn gọi core của `/healthz?deep=1`; mặc định CORE_TIMEOUT_MS (2 s). */
  coreTimeoutMs?: number;
}

/** Hạn gọi core của `/healthz?deep=1`, bằng hạn /readyz. */
export const CORE_TIMEOUT_MS = 2000;

/** Trường log của một lỗi: tên, message, mã nguyên nhân (vd ECONNREFUSED). Không log stack/object thô. */
function errorFields(err: unknown): Record<string, string> {
  const e = err instanceof Error ? err : new Error(String(err));
  const fields: Record<string, string> = { error_type: e.name, error: e.message };
  const cause = e.cause as { code?: unknown; name?: unknown } | undefined;
  if (cause && typeof cause === 'object') {
    const c = typeof cause.code === 'string' ? cause.code : typeof cause.name === 'string' ? cause.name : undefined;
    if (c) fields.error_cause = c;
  }
  return fields;
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

const healthSchema = {
  type: 'object',
  properties: { status: { type: 'string' }, core: { type: 'string' } },
  required: ['status'],
} as const;

export const healthRoutes: FastifyPluginAsync<HealthOptions> = async (app, opts) => {
  const timeoutMs = opts.readyTimeoutMs ?? READY_TIMEOUT_MS;
  const coreTimeoutMs = opts.coreTimeoutMs ?? CORE_TIMEOUT_MS;

  // Liveness: process còn phục vụ được, không chạm DB.
  // `?deep=1`: gọi thêm core `/readyz` đúng một lần trong hạn, để kiểm trace
  // xuyên service bff → core → PG. Giá trị khác của `deep` coi như không có.
  app.get<{ Querystring: { deep?: unknown } }>(
    '/healthz',
    { schema: { response: { 200: healthSchema, 503: healthSchema } } },
    async (req, reply) => {
      if (req.query.deep !== '1') {
        return { status: 'ok' };
      }
      try {
        await withTimeout(() => opts.core.ready(coreTimeoutMs), coreTimeoutMs);
        return { status: 'ok', core: 'ok' };
      } catch (err) {
        req.log.warn(errorFields(err), 'healthz: core unavailable');
        return reply.code(503).send({ status: 'unavailable', core: 'unavailable' });
      }
    },
  );

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
