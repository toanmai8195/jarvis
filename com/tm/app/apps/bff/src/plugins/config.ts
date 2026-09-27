import type { FastifyPluginAsync } from 'fastify';
import fp from 'fastify-plugin';

export const LOG_LEVELS = ['trace', 'debug', 'info', 'warn', 'error', 'fatal'] as const;
export type LogLevel = (typeof LOG_LEVELS)[number];

export interface Config {
  mongodbUri: string;
  port: number;
  host: string;
  logLevel: LogLevel;
}

export const DEFAULT_PORT = 3000;
export const DEFAULT_HOST = '0.0.0.0';
export const DEFAULT_LOG_LEVEL: LogLevel = 'info';

/** Một biến sai: chỉ có tên và lý do, không bao giờ chứa giá trị (có thể là mật khẩu). */
export interface ConfigProblem {
  name: string;
  reason: string;
}

export class ConfigError extends Error {
  readonly problems: readonly ConfigProblem[];

  constructor(problems: readonly ConfigProblem[]) {
    super(`invalid config: ${problems.map((p) => `${p.name}: ${p.reason}`).join('; ')}`);
    this.name = 'ConfigError';
    this.problems = problems;
  }

  get names(): string[] {
    return this.problems.map((p) => p.name);
  }
}

type Env = Readonly<Record<string, string | undefined>>;

const MONGO_SCHEMES = ['mongodb://', 'mongodb+srv://'];

function isLogLevel(v: string): v is LogLevel {
  return (LOG_LEVELS as readonly string[]).includes(v);
}

/**
 * Đọc và validate env của bff. Validate hết mọi biến rồi mới báo lỗi, để một
 * lần chạy thấy đủ các biến sai. Chuỗi rỗng coi như không đặt.
 */
export function loadConfig(env: Env): Config {
  const problems: ConfigProblem[] = [];

  const uri = env.MONGODB_URI ?? '';
  if (uri === '') {
    problems.push({ name: 'MONGODB_URI', reason: 'required' });
  } else if (!MONGO_SCHEMES.some((s) => uri.startsWith(s) && uri.length > s.length)) {
    problems.push({ name: 'MONGODB_URI', reason: 'must be a mongodb:// or mongodb+srv:// URL' });
  }

  let port = DEFAULT_PORT;
  const rawPort = env.BFF_PORT ?? '';
  if (rawPort !== '') {
    const n = /^\d{1,5}$/.test(rawPort) ? Number(rawPort) : NaN;
    if (!Number.isInteger(n) || n < 1 || n > 65535) {
      problems.push({ name: 'BFF_PORT', reason: 'must be an integer in 1..65535' });
    } else {
      port = n;
    }
  }

  const host = env.BFF_HOST || DEFAULT_HOST;

  let logLevel = DEFAULT_LOG_LEVEL;
  const rawLevel = env.BFF_LOG_LEVEL ?? '';
  if (rawLevel !== '') {
    if (isLogLevel(rawLevel)) {
      logLevel = rawLevel;
    } else {
      problems.push({ name: 'BFF_LOG_LEVEL', reason: `must be one of ${LOG_LEVELS.join(', ')}` });
    }
  }

  if (problems.length > 0) {
    throw new ConfigError(problems);
  }
  return { mongodbUri: uri, port, host, logLevel };
}

declare module 'fastify' {
  interface FastifyInstance {
    config: Config;
  }
}

/** Gắn config đã validate vào instance (`app.config`), không bị encapsulate. */
const config: FastifyPluginAsync<{ config: Config }> = async (app, opts) => {
  app.decorate('config', opts.config);
};

export const configPlugin = fp(config, { name: 'config', fastify: '5.x' });
