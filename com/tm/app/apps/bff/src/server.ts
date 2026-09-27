import { buildApp } from './app.js';
import { createLogger } from './logger.js';
import { ConfigError, DEFAULT_LOG_LEVEL, loadConfig, type Config } from './plugins/config.js';

// Entry của bff: đọc config → dựng app → listen. Không tự đọc .env.
// Graceful shutdown (SIGTERM, drain) là P0-T13.

async function main(): Promise<number> {
  let config: Config;
  try {
    config = loadConfig(process.env);
  } catch (err) {
    // Chưa biết level → logger mặc định; lỗi chỉ nêu tên biến, không in giá trị.
    const log = createLogger(DEFAULT_LOG_LEVEL);
    if (err instanceof ConfigError) {
      log.fatal({ invalid: err.names, problems: err.problems }, err.message);
    } else {
      log.fatal({ err }, 'config: unexpected error');
    }
    return 1;
  }

  const logger = createLogger(config.logLevel);
  let app: Awaited<ReturnType<typeof buildApp>> | undefined;
  try {
    app = await buildApp({ config, logger });
    await app.listen({ port: config.port, host: config.host });
    return 0;
  } catch (err) {
    logger.fatal({ err }, 'bff: start failed');
    await app?.close().catch(() => undefined);
    return 1;
  }
}

const code = await main();
if (code !== 0) {
  process.exit(code);
}
