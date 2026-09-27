import Fastify, { LogController } from 'fastify';
import type { Logger } from 'pino';

import { configPlugin, type Config } from './plugins/config.js';
import { mongoPlugin, pingMongo } from './plugins/mongo.js';
import { requestIdFrom } from './request-id.js';
import { healthRoutes, type Pinger } from './routes/health.js';

export interface BuildOptions {
  config: Config;
  logger: Logger;
  /** Thay ping MongoDB (test). Không truyền → dùng plugin mongo thật. */
  pinger?: Pinger;
  /** Hạn của /readyz; mặc định READY_TIMEOUT_MS (2 s). */
  readyTimeoutMs?: number;
}

/** Dựng app Fastify (chưa listen) — wiring tay, dependency qua option/decorate. */
export async function buildApp(opts: BuildOptions) {
  const app = Fastify({
    loggerInstance: opts.logger,
    // Không để Fastify lấy nguyên header: genReqId tự đọc và kiểm X-Request-ID.
    requestIdHeader: false,
    genReqId: requestIdFrom,
    // Khoá log `request_id` thay cho `reqId` (option top-level requestIdLogLabel đã deprecated).
    logController: new LogController({ requestIdLogLabel: 'request_id' }),
  });

  // Hook ở root (không encapsulate) → mọi response, kể cả 404/503/500, có X-Request-ID.
  app.addHook('onRequest', async (req, reply) => {
    reply.header('x-request-id', req.id);
  });

  await app.register(configPlugin, { config: opts.config });

  let pinger = opts.pinger;
  if (!pinger) {
    await app.register(mongoPlugin, { uri: opts.config.mongodbUri });
    const client = app.mongo;
    pinger = { ping: (ms) => pingMongo(client, ms) };
  }

  await app.register(healthRoutes, { pinger, readyTimeoutMs: opts.readyTimeoutMs });
  return app;
}
