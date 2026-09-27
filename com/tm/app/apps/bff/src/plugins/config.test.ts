import Fastify from 'fastify';
import { describe, expect, it } from 'vitest';

import { ConfigError, configPlugin, loadConfig } from './config.js';

const URI = 'mongodb://localhost:27017/snaptix';

function problemsOf(env: Record<string, string | undefined>): ConfigError {
  try {
    loadConfig(env);
  } catch (err) {
    if (err instanceof ConfigError) return err;
    throw err;
  }
  throw new Error('expected ConfigError');
}

describe('loadConfig', () => {
  it('giá trị mặc định: BFF_PORT 3000, BFF_HOST 0.0.0.0, BFF_LOG_LEVEL info', () => {
    expect(loadConfig({ MONGODB_URI: URI })).toEqual({
      mongodbUri: URI,
      port: 3000,
      host: '0.0.0.0',
      logLevel: 'info',
    });
  });

  it('đọc đủ biến khi hợp lệ (mongodb+srv://, BFF_PORT, BFF_HOST, BFF_LOG_LEVEL)', () => {
    expect(
      loadConfig({
        MONGODB_URI: 'mongodb+srv://u:p@cluster.example.net/db',
        BFF_PORT: '65535',
        BFF_HOST: '127.0.0.1',
        BFF_LOG_LEVEL: 'debug',
      }),
    ).toEqual({
      mongodbUri: 'mongodb+srv://u:p@cluster.example.net/db',
      port: 65535,
      host: '127.0.0.1',
      logLevel: 'debug',
    });
  });

  it.each([
    ['thiếu', {}],
    ['rỗng', { MONGODB_URI: '' }],
    ['sai scheme http://', { MONGODB_URI: 'http://x:1/db' }],
    ['chỉ có scheme', { MONGODB_URI: 'mongodb://' }],
  ])('MONGODB_URI %s → lỗi nêu tên MONGODB_URI', (_, env) => {
    expect(problemsOf(env).names).toEqual(['MONGODB_URI']);
  });

  it.each(['abc', '0', '70000', '-1', '80.5', '1e3', ' 80'])(
    'BFF_PORT=%j không phải số nguyên 1..65535 → lỗi BFF_PORT',
    (port) => {
      expect(problemsOf({ MONGODB_URI: URI, BFF_PORT: port }).names).toEqual(['BFF_PORT']);
    },
  );

  it('BFF_PORT biên 1 và 65535 hợp lệ', () => {
    expect(loadConfig({ MONGODB_URI: URI, BFF_PORT: '1' }).port).toBe(1);
    expect(loadConfig({ MONGODB_URI: URI, BFF_PORT: '65535' }).port).toBe(65535);
  });

  it.each(['trace', 'debug', 'info', 'warn', 'error', 'fatal'])('BFF_LOG_LEVEL=%s được chấp nhận', (lvl) => {
    expect(loadConfig({ MONGODB_URI: URI, BFF_LOG_LEVEL: lvl }).logLevel).toBe(lvl);
  });

  it.each(['verbose', 'INFO', 'silent'])('BFF_LOG_LEVEL lạ %j → lỗi BFF_LOG_LEVEL', (lvl) => {
    expect(problemsOf({ MONGODB_URI: URI, BFF_LOG_LEVEL: lvl }).names).toEqual(['BFF_LOG_LEVEL']);
  });

  it('nhiều biến sai (MONGODB_URI, BFF_PORT, BFF_LOG_LEVEL) → nêu tất cả trong một lần', () => {
    const e = problemsOf({ BFF_PORT: 'abc', BFF_LOG_LEVEL: 'verbose' });
    expect(e.names).toEqual(['MONGODB_URI', 'BFF_PORT', 'BFF_LOG_LEVEL']);
  });

  it('lỗi MONGODB_URI nêu tên biến nhưng không chứa giá trị/mật khẩu', () => {
    const e = problemsOf({ MONGODB_URI: 'http://tcuser:tc-pw-SECRET@x:1/db', BFF_PORT: 'p0rt-SECRET' });
    const text = `${e.message} ${JSON.stringify(e.problems)}`;
    expect(text).toContain('MONGODB_URI');
    expect(text).toContain('BFF_PORT');
    expect(text).not.toContain('tc-pw-SECRET');
    expect(text).not.toContain('tcuser');
    expect(text).not.toContain('p0rt-SECRET');
  });
});

describe('configPlugin', () => {
  it('decorate app.config, không bị encapsulate (fastify-plugin)', async () => {
    const app = Fastify({ logger: false });
    const config = loadConfig({ MONGODB_URI: URI });
    await app.register(configPlugin, { config });
    await app.ready();
    expect(app.config).toBe(config);
    await app.close();
  });
});
