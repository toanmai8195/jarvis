/**
 * Entry OpenTelemetry của bff, nạp TRƯỚC `server.ts` bằng cờ `--import`:
 *   node --import ./dist/instrumentation.js dist/server.js
 *   tsx watch --import ./src/instrumentation.ts src/server.ts
 *
 * Phải là entry riêng: tsup gom `server.ts` + `app.ts` thành một bundle và
 * `import` tĩnh được hoist, nên nếu `sdk.start()` nằm chung bundle thì fastify
 * và node:http đã nạp xong trước khi instrumentation kịp patch.
 *
 * File này không import fastify hay node:http.
 */
import { createLogger } from './logger.js';
import { DEFAULT_LOG_LEVEL, LOG_LEVELS, type LogLevel } from './plugins/config.js';
import { createOtelSdk, otelEnvProblems } from './plugins/otel.js';

const rawLevel = process.env.BFF_LOG_LEVEL ?? '';
// BFF_LOG_LEVEL sai do server.ts báo lỗi; ở đây chỉ dùng mức mặc định.
const level: LogLevel = (LOG_LEVELS as readonly string[]).includes(rawLevel) ? (rawLevel as LogLevel) : DEFAULT_LOG_LEVEL;
const log = createLogger(level);

const problems = otelEnvProblems(process.env);
if (problems.length > 0) {
  // Giống lỗi config của server.ts: log JSON fatal nêu tên biến, không in giá trị, exit 1.
  log.fatal(
    { invalid: problems.map((p) => p.name), problems },
    `invalid config: ${problems.map((p) => `${p.name}: ${p.reason}`).join('; ')}`,
  );
  process.exit(1);
}

// Không bắt SIGTERM / flush khi dừng ở đây: graceful shutdown của bff là P0-T13.
createOtelSdk({ env: process.env, logger: log })?.start();
