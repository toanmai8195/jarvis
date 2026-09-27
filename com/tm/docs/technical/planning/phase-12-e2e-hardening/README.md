# Phase 12 — E2E & hardening

> Chặng D — Tổng hợp · Công nghệ: **Toàn hệ thống**

## Mục tiêu

Bộ E2E đầy đủ và regression toàn bộ test nghiệm thu.

**Mốc demo**: E2E luồng chính chạy xanh trong CI < 10 phút

## Kiến thức trọng tâm

Playwright, test chiến lược toàn hệ thống

## Phạm vi

- **Trong**: E2E, regression.
- **Ngoài**: —

## Workstream

`qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P12-NFR1 | NFR | E2E toàn bộ luồng chính chạy trong CI < 10 phút |

## Task

### qa
- [ ] **P12-T01** E2E setup tuyến → slot → đặt vé trên web client _(trước đây P5-T15)_
- [ ] **P12-T02** Bộ Playwright E2E: đăng nhập → nạp tiền → đặt vé → huỷ vé; admin setup → bán → huỷ chuyến `[R8]` _(trước đây P8-T11)_
- [ ] **P12-T03** Chạy lại toàn bộ test nghiệm thu các phase trước (regression) _(trước đây P8-T13)_

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|

## Definition of Done

- [ ] E2E luồng người dùng và admin xanh trong CI
- [ ] Regression mọi test nghiệm thu pass
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật toàn bộ docs technical theo code (bỏ ghi chú "bản thiết kế")
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
