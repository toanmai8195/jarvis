import { pino, type DestinationStream, type Logger } from 'pino';

import type { LogLevel } from './plugins/config.js';

/**
 * Logger JSON một dòng một object, đọc thống nhất với slog của core:
 * `time` ISO 8601, `level` nhãn chữ thường, `msg`. Không pino-pretty.
 */
export function createLogger(level: LogLevel, dest?: DestinationStream): Logger {
  const opts = {
    level,
    timestamp: pino.stdTimeFunctions.isoTime,
    formatters: {
      level: (label: string) => ({ level: label }),
    },
  };
  return dest ? pino(opts, dest) : pino(opts);
}
