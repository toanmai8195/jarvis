import { randomBytes } from 'node:crypto';
import type { IncomingMessage } from 'node:http';

/** api.md "Quy ước chung": giữ X-Request-ID của client nếu khớp mẫu này. */
export const REQUEST_ID_PATTERN = /^[A-Za-z0-9._:-]{1,128}$/;

/**
 * UUID v7 (RFC 9562): 48 bit Unix ms + version 7 + variant 10 + ngẫu nhiên.
 * Sắp theo thời gian, nên id sinh sau lớn hơn theo từng ms.
 */
export function uuidv7(nowMs: number = Date.now()): string {
  const b = randomBytes(16);
  b.writeUIntBE(nowMs, 0, 6);
  b[6] = ((b[6] ?? 0) & 0x0f) | 0x70;
  b[8] = ((b[8] ?? 0) & 0x3f) | 0x80;
  const h = b.toString('hex');
  return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`;
}

/**
 * `genReqId` của Fastify, dùng cùng `requestIdHeader: false`: tự đọc header
 * `x-request-id`; hợp lệ thì giữ, còn lại (thiếu, rỗng, sai ký tự, > 128,
 * header lặp — Node gộp thành "a, b") sinh UUID v7.
 */
export function requestIdFrom(req: Pick<IncomingMessage, 'headers'>): string {
  const h = req.headers['x-request-id'];
  if (typeof h === 'string' && REQUEST_ID_PATTERN.test(h)) {
    return h;
  }
  return uuidv7();
}
