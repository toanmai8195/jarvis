import { describe, expect, it } from 'vitest';

import { requestIdFrom, uuidv7 } from './request-id.js';

const UUID7 = /^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

describe('uuidv7', () => {
  it('đúng định dạng version 7, variant 10', () => {
    for (let i = 0; i < 100; i++) {
      expect(uuidv7()).toMatch(UUID7);
    }
  });

  it('48 bit đầu là Unix ms, sắp theo thời gian', () => {
    const t = 1_790_000_000_123;
    const id = uuidv7(t);
    expect(parseInt(id.slice(0, 8) + id.slice(9, 13), 16)).toBe(t);
    expect(uuidv7(t + 1) > id).toBe(true);
  });
});

describe('requestIdFrom (X-Request-ID)', () => {
  const from = (h?: string | string[]) =>
    requestIdFrom({ headers: h === undefined ? {} : { 'x-request-id': h } });

  it('X-Request-ID hợp lệ được giữ nguyên', () => {
    expect(from('tc12.ok_A:1-z')).toBe('tc12.ok_A:1-z');
    expect(from('a'.repeat(128))).toBe('a'.repeat(128));
  });

  it.each([
    ['thiếu', undefined],
    ['rỗng', ''],
    ['ký tự lạ', 'bad id!'],
    ['dài > 128', 'a'.repeat(129)],
    ['header lặp đã gộp', 'a, b'],
    ['mảng', ['a', 'b']],
  ])('X-Request-ID %s → sinh UUID v7', (_, h) => {
    expect(from(h)).toMatch(UUID7);
  });
});
