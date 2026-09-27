import type { FastifyPluginAsync } from 'fastify';
import fp from 'fastify-plugin';
import { MongoClient } from 'mongodb';

/** Hạn chọn server; phải ≤ hạn ping của /readyz (mặc định driver là 30 s). */
export const SERVER_SELECTION_TIMEOUT_MS = 2000;

export interface MongoPluginOptions {
  uri: string;
  serverSelectionTimeoutMS?: number;
}

declare module 'fastify' {
  interface FastifyInstance {
    mongo: MongoClient;
  }
}

/**
 * Client MongoDB lười: tạo client nhưng không connect, nên bff khởi động được
 * khi Mongo chưa lên. Đóng client ở `onClose` (vòng đời plugin).
 */
const mongo: FastifyPluginAsync<MongoPluginOptions> = async (app, opts) => {
  const client = new MongoClient(opts.uri, {
    serverSelectionTimeoutMS: opts.serverSelectionTimeoutMS ?? SERVER_SELECTION_TIMEOUT_MS,
  });
  app.decorate('mongo', client);
  app.addHook('onClose', async () => {
    await client.close();
  });
};

export const mongoPlugin = fp(mongo, { name: 'mongo', fastify: '5.x' });

/**
 * Ping MongoDB có hạn. Gọi `connect()` trước mỗi lần: idempotent khi đã nối,
 * và mở lại topology sau khi lần chọn server trước thất bại — driver 7 đóng
 * hẳn topology khi auto-connect lỗi, mọi lệnh sau đó trả
 * MongoTopologyClosedError kể cả khi Mongo đã lên lại.
 */
export async function pingMongo(client: MongoClient, timeoutMs: number): Promise<void> {
  await client.connect();
  await client.db().command({ ping: 1 }, { timeoutMS: timeoutMs });
}
