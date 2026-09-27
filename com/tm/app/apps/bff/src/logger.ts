import { context, isSpanContextValid, trace } from '@opentelemetry/api';
import { pino, type DestinationStream, type Logger } from 'pino';

import type { LogLevel } from './plugins/config.js';

/**
 * `trace_id`/`span_id` của span đang active (khoá giống log slog của core).
 * Không có span hợp lệ (ngoài request, SDK tắt) → không thêm khoá nào.
 * Chỉ dùng `@opentelemetry/api`: SDK chưa start thì API là no-op.
 */
export function traceFields(): Record<string, string> {
  const span = trace.getSpan(context.active());
  if (!span) return {};
  const sc = span.spanContext();
  if (!isSpanContextValid(sc)) return {};
  return { trace_id: sc.traceId, span_id: sc.spanId };
}

/**
 * Logger JSON một dòng một object, đọc thống nhất với slog của core:
 * `time` ISO 8601, `level` nhãn chữ thường, `msg`, và `trace_id`/`span_id`
 * khi có span active. Không pino-pretty.
 */
export function createLogger(level: LogLevel, dest?: DestinationStream): Logger {
  const opts = {
    level,
    timestamp: pino.stdTimeFunctions.isoTime,
    formatters: {
      level: (label: string) => ({ level: label }),
    },
    mixin: traceFields,
  };
  return dest ? pino(opts, dest) : pino(opts);
}
