# Phase 2 — Ví (core)

> Chặng A — Go + PostgreSQL (core) · Công nghệ: **Go + PostgreSQL**

## Mục tiêu

Bài toán ghi dữ liệu chính xác: transaction, isolation, lost update, ledger kế toán kép, idempotency.

**Mốc demo**: `curl` nạp tiền qua mock provider, số dư và lịch sử đúng; retry không ghi trùng

## Kiến thức trọng tâm

Go: value type Money, WithTx, test song song với -race · PG: transaction, isolation, lock, constraint, trigger

## Phạm vi

- **Trong**: Tạo user + ví ở core, xác thực service token, ledger, topup, webhook, idempotency, mock payment provider.
- **Ngoài**: BFF, giao diện ví.

## Workstream

`db` · `core` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P2-FR1 | FR | Tạo lệnh nạp tiền, nhận `payment_url` |
| P2-FR2 | FR | Webhook thành công → cộng tiền vào ví đúng một lần |
| P2-FR3 | FR | Xem số dư và lịch sử giao dịch (phân trang cursor) |
| P2-FR4 | FR | Lệnh nạp PENDING quá 30 phút → EXPIRED |
| P2-FR5 | FR | Mọi lệnh ghi tiền nhận `Idempotency-Key`; retry trả lại đúng response cũ |
| P2-NFR1 | NFR | Tổng mọi `ledger_entries.amount` luôn = 0 |
| P2-NFR2 | NFR | Số dư ví user không bao giờ âm, kể cả khi có bug ở code |
| P2-NFR3 | NFR | Không dùng số thực cho tiền ở bất kỳ tầng nào |
| P2-FR6 | FR | `POST /internal/v1/users` idempotent: tạo user + ví trong một transaction |
| P2-FR7 | FR | API nội bộ chỉ nhận request có service token hợp lệ; đọc `X-User-Id`, `X-User-Role` |

## Task

### db
- [ ] **P2-T01** Migration `accounts`, `ledger_transactions`, `ledger_entries`, `topups`, `idempotency_keys`; seed tài khoản hệ thống `CASH_IN`, `REVENUE`, `REFUND_EXPENSE` `[P5]` _(trước đây P3-T01)_
- [ ] **P2-T02** Constraint: `CHECK` số dư ≥ 0 cho ví user, unique `idempotency_key`, unique `provider_ref` `[P5][P6]` _(trước đây P3-T02)_
- [ ] **P2-T03** Chặn UPDATE/DELETE trên `ledger_entries` (quyền DB hoặc trigger) `[P5]` _(trước đây P3-T03)_

### core
- [ ] **P2-T04** Value object `Money` (int64), phép cộng/trừ kiểm tra overflow; lint cấm float trong package wallet `[G6]` _(trước đây P3-T04)_
- [ ] **P2-T05** `pkg/postgres.WithTx` (kèm retry `40001`/`40P01`); service mở transaction, hàm store nhận `pgx.Tx` `[G5]` _(trước đây P3-T05)_
- [ ] **P2-T06** Service `Ledger.Post(tx, entries)`: kiểm tra tổng = 0, cập nhật `balance` + `balance_after` `[P5]` _(trước đây P3-T06)_
- [ ] **P2-T07** `POST /internal/v1/users` idempotent: tạo `users` + `accounts` (ví) trong một transaction _(trước đây P2-T10)_
- [ ] **P2-T08** Middleware xác thực service token, đọc `X-User-Id`, `X-User-Role` _(trước đây P2-T11)_
- [ ] **P2-T09** Middleware idempotency: lưu key + request hash + response trong cùng transaction `[P6]` _(trước đây P3-T07)_
- [ ] **P2-T10** Tái hiện lost update khi hai giao dịch cập nhật cùng ví ở READ COMMITTED; chọn cách chặn (`SELECT ... FOR UPDATE` / update có điều kiện / SERIALIZABLE + retry) `[P4]` _(trước đây P3-T08)_
- [ ] **P2-T11** Mock payment provider: trang thanh toán giả, gửi webhook ký HMAC, có chế độ gửi lặp / trễ / mất _(trước đây P3-T09)_
- [ ] **P2-T12** API topup, webhook, số dư, lịch sử _(trước đây P3-T10)_
- [ ] **P2-T13** Worker hết hạn topup PENDING _(trước đây P3-T11)_

### qa
- [ ] **P2-T14** Integration test song song nhiều giao dịch cùng ví với testcontainers + `-race` `[G9]` _(trước đây P3-T14)_
- [ ] **P2-T15** Property test cho `Money` và `Ledger.Post` `[G6]` _(trước đây P3-T15)_
- [ ] **P2-T16** Script kiểm tra bất biến ledger (dùng lại ở Phase 4) _(trước đây P3-T16)_

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| P5 | PostgreSQL core | Ledger chính xác tuyệt đối | Nạp, trả, hoàn tiền | Double-entry, `CHECK` constraint, entry bất biến, snapshot balance | Job đối soát luôn ra chênh lệch 0 | ⬜ |
| P6 | PostgreSQL core | Idempotency | Retry, double click, webhook lặp | Bảng `idempotency_keys` ghi cùng transaction, `INSERT ... ON CONFLICT` | Gửi 1 request 100 lần → 1 lần trừ tiền | ⬜ |
| G6 | Golang | Kiểu tiền an toàn | Toàn bộ ví, giá vé | Value object `Money` (int64), cấm phép tính float, kiểm tra overflow | Lint/test chặn float; property test cho cộng/trừ | ⬜ |
| G5 | Golang | Ranh giới transaction giữa các module | Use case đặt vé đụng ghế + ví + outbox | `postgres.WithTx`, truyền `pgx.Tx` qua interface phía dùng; logic tính toán tách thành hàm thuần | Một use case = một transaction; logic giá/hoàn tiền test không cần DB | ⬜ |
| P4 | PostgreSQL core | Isolation level & anomaly | Ví: lost update, write skew | Hiểu RC / RR / Serializable; tái hiện anomaly bằng test; retry `40001` | Có test tái hiện và test chứng minh đã chặn | ⬜ |
| G9 | Golang | Test với DB thật | Logic khoá ghế, ledger | testcontainers-go, test song song race condition, `go test -race` | Test N goroutine tranh 1 ghế → đúng 1 thành công | ⬜ |

## Definition of Done

- [ ] Nạp tiền end-to-end qua mock provider bằng `curl`
- [ ] Webhook gửi lặp 10 lần → cộng tiền 1 lần
- [ ] 100 request cùng `Idempotency-Key` song song → 1 giao dịch
- [ ] Script bất biến ledger: tổng entries = 0, balance khớp
- [ ] Có test tái hiện lost update và test chứng minh đã chặn
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [database](../../database.md) phần ví
- [ ] ADR: cách chặn lost update, thiết kế idempotency
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
