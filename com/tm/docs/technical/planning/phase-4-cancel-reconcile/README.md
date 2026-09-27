# Phase 4 — Huỷ/hoàn vé & đối soát (core)

> Chặng A — Go + PostgreSQL (core) · Công nghệ: **Go + PostgreSQL**

## Mục tiêu

Hoàn thiện vòng đời vé phía core: huỷ, hoàn tiền, huỷ chuyến hàng loạt, đối soát tiền, migration không downtime.

**Mốc demo**: `curl` huỷ vé → tiền hoàn đúng chính sách; job đối soát báo chênh lệch 0

## Kiến thức trọng tâm

Go: job batch có checkpoint · PG: migration expand–contract, lock_timeout, CREATE INDEX CONCURRENTLY

## Phạm vi

- **Trong**: Chính sách hoàn, huỷ vé, huỷ chuyến, hoàn thủ công, đối soát, migration dưới tải.
- **Ngoài**: API public và giao diện huỷ vé.

## Workstream

`core` · `db`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P4-FR1 | FR | Xem trước số tiền hoàn theo chính sách (≥24h: 90%, 4–24h: 50%, <4h: 0%) |
| P4-FR2 | FR | Huỷ vé: vé CANCELLED, ghế AVAILABLE, hoàn tiền vào ví, outbox — một transaction |
| P4-FR3 | FR | Huỷ một phần vé trong đơn → đơn PARTIALLY_CANCELLED |
| P4-FR4 | FR | Admin huỷ chuyến → hoàn 100% mọi vé, chạy theo batch, tiếp tục được nếu gián đoạn |
| P4-FR5 | FR | Support hoàn tiền thủ công có lý do, ghi audit log |
| P4-FR6 | FR | Job đối soát hằng giờ: bất biến ledger, balance, topup vs provider; cảnh báo khi lệch |
| P4-NFR1 | NFR | Migration trên bảng lớn chạy dưới tải không khoá bảng |

## Task

### core
- [ ] **P4-T01** Chính sách hoàn tiền dạng cấu hình, tính theo thời điểm huỷ so với giờ khởi hành _(trước đây P8-T01)_
- [ ] **P4-T02** Use case huỷ vé (một hoặc nhiều vé), idempotent _(trước đây P8-T02)_
- [ ] **P4-T03** Huỷ chuyến: job batch có checkpoint, idempotent theo ticket _(trước đây P8-T03)_
- [ ] **P4-T04** Hoàn tiền thủ công (giới hạn không vượt số đã trả) _(trước đây P8-T04)_
- [ ] **P4-T05** Job đối soát: bất biến ledger, `balance` vs tổng entries, topup SUCCEEDED vs báo cáo mock provider; xuất metric + cảnh báo Grafana _(trước đây P8-T05)_

### db
- [ ] **P4-T06** Thực hành expand–contract: thêm cột mới vào `tickets` (ví dụ `cancel_reason`), backfill theo batch, `CREATE INDEX CONCURRENTLY`, chạy dưới k6 `[P10]` _(trước đây P8-T06)_
- [ ] **P4-T07** Kiểm tra lock bằng `lock_timeout`, `statement_timeout` cho migration `[P10]` _(trước đây P8-T07)_

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| P10 | PostgreSQL core | Migration không downtime | Thêm cột/index trên bảng lớn | `CREATE INDEX CONCURRENTLY`, expand–contract, backfill theo batch | Migration chạy dưới tải không khoá bảng | ⬜ |

## Definition of Done

- [ ] Huỷ vé, huỷ chuyến, hoàn thủ công chạy qua API core
- [ ] Job đối soát chạy, báo chênh lệch 0
- [ ] Migration expand–contract dưới tải không lỗi do lock
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] ADR: quy trình migration không downtime
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
