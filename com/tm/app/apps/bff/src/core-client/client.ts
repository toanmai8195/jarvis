/**
 * Client gọi core qua `fetch` global của Node (undici). Span CLIENT và header
 * `traceparent` do `@opentelemetry/instrumentation-undici` thêm (nạp ở
 * `src/instrumentation.ts`), code ở đây không đụng OTel.
 *
 * Mỗi lời gọi đúng một request, có hạn. Keep-alive agent riêng, token dịch vụ
 * và API nội bộ để phase 2 (P2-T05).
 */

/** Core trả mã ngoài 2xx. Chỉ mang mã và path, không mang URL đầy đủ hay body. */
export class CoreStatusError extends Error {
  readonly status: number;

  constructor(path: string, status: number) {
    super(`core ${path} returned ${status}`);
    this.name = 'CoreStatusError';
    this.status = status;
  }
}

export interface CoreClient {
  /** `GET {base}/readyz` của core (có ping PG); resolve khi 2xx, reject khi lỗi/quá hạn. */
  ready(timeoutMs: number): Promise<void>;
}

export interface CoreClientOptions {
  /** Gốc URL core, đã validate (không userinfo). */
  baseUrl: string;
  /** Thay `fetch` (test). Mặc định `globalThis.fetch`. */
  fetch?: typeof fetch;
}

export function createCoreClient(opts: CoreClientOptions): CoreClient {
  const base = opts.baseUrl.replace(/\/+$/, '');
  const doFetch = opts.fetch ?? globalThis.fetch;

  return {
    async ready(timeoutMs) {
      const path = '/readyz';
      // AbortSignal.timeout huỷ cả kết nối lẫn đọc body khi quá hạn.
      const res = await doFetch(`${base}${path}`, {
        method: 'GET',
        headers: { accept: 'application/json' },
        signal: AbortSignal.timeout(timeoutMs),
      });
      // Đọc hết body để trả kết nối về pool của undici.
      await res.arrayBuffer().catch(() => undefined);
      if (!res.ok) {
        throw new CoreStatusError(path, res.status);
      }
    },
  };
}
