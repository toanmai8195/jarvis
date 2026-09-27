import Fastify from 'fastify';
import { MongoClient } from 'mongodb';
import { describe, expect, it } from 'vitest';

import { mongoPlugin, pingMongo } from './mongo.js';

// Cổng 1 trên loopback: không có Mongo, connect bị từ chối ngay — không cần Docker.
const UNREACHABLE = 'mongodb://tcuser:tc-pw-SECRET@127.0.0.1:1/snaptix';

describe('mongoPlugin', () => {
  it('client lười: ready + close không cần Mongo chạy', async () => {
    const app = Fastify({ logger: false });
    await app.register(mongoPlugin, { uri: UNREACHABLE });
    const t0 = performance.now();
    await app.ready();
    expect(app.mongo).toBeInstanceOf(MongoClient);
    await app.close();
    expect(performance.now() - t0).toBeLessThan(1000);
  });

  it('URI sai định dạng → register lỗi (MongoParseError)', async () => {
    const app = Fastify({ logger: false });
    await expect(app.register(mongoPlugin, { uri: 'mongodb://a:b:c@/x?bad=%' })).rejects.toThrow();
    await app.close();
  });
});

describe('pingMongo', () => {
  it('Mongo không tới được → reject trong hạn serverSelectionTimeoutMS (timeout), lỗi không lộ mật khẩu', async () => {
    const app = Fastify({ logger: false });
    await app.register(mongoPlugin, { uri: UNREACHABLE, serverSelectionTimeoutMS: 200 });
    await app.ready();
    const t0 = performance.now();
    const err = await pingMongo(app.mongo, 200).then(
      () => undefined,
      (e: unknown) => e,
    );
    expect(err).toBeInstanceOf(Error);
    expect(performance.now() - t0).toBeLessThan(2000);
    expect(String((err as Error).message)).not.toContain('tc-pw-SECRET');
    // Lần thứ hai vẫn thử connect lại (không kẹt MongoTopologyClosedError).
    const err2 = await pingMongo(app.mongo, 200).then(
      () => undefined,
      (e: unknown) => e,
    );
    expect((err2 as Error).name).not.toBe('MongoTopologyClosedError');
    await app.close();
  });
});
