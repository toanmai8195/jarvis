# Phase 3 — Ví

## Mục tiêu

Xây ví nội bộ theo mô hình double-entry ledger, nạp tiền qua cổng thanh toán (giả lập), idempotency cho mọi lệnh ghi tiền.

**Mốc demo**: nạp tiền trên web, số dư cập nhật sau webhook; lịch sử giao dịch chính xác; bấm nạp nhiều lần không bị ghi trùng.

## Phạm vi

- **Trong**: `accounts`, `ledger_*`, `topups`, `idempotency_keys`; cổng thanh toán giả lập (mock provider có webhook ký HMAC); API ví; UI ví.
- **Ngoài**: trả tiền vé (phase 4), hoàn tiền (phase 8), đối soát đầy đủ (phase 8).

## Workstream

`db` · `core` · `bff` · `web` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P3-FR1 | FR | Tạo lệnh nạp tiền, nhận `payment_url` |
| P3-FR2 | FR | Webhook thành công → cộng tiền vào ví đúng một lần |
| P3-FR3 | FR | Xem số dư và lịch sử giao dịch (phân trang cursor) |
| P3-FR4 | FR | Lệnh nạp PENDING quá 30 phút → EXPIRED |
| P3-FR5 | FR | Mọi lệnh ghi tiền nhận `Idempotency-Key`; retry trả lại đúng response cũ |
| P3-NFR1 | NFR | Tổng mọi `ledger_entries.amount` luôn = 0 |
| P3-NFR2 | NFR | Số dư ví user không bao giờ âm, kể cả khi có bug ở code |
| P3-NFR3 | NFR | Không dùng số thực cho tiền ở bất kỳ tầng nào |

## Task

### db
- [ ] **P3-T01** Migration `accounts`, `ledger_transactions`, `ledger_entries`, `topups`, `idempotency_keys`; seed tài khoản hệ thống `CASH_IN`, `REVENUE`, `REFUND_EXPENSE` `[P5]`
- [ ] **P3-T02** Constraint: `CHECK` số dư ≥ 0 cho ví user, unique `idempotency_key`, unique `provider_ref` `[P5][P6]`
- [ ] **P3-T03** Chặn UPDATE/DELETE trên `ledger_entries` (quyền DB hoặc trigger) `[P5]`

### core
- [ ] **P3-T04** Value object `Money` (int64), phép cộng/trừ kiểm tra overflow; lint cấm float trong package wallet `[G6]`
- [ ] **P3-T05** `pkg/postgres.WithTx` (kèm retry `40001`/`40P01`); service mở transaction, hàm store nhận `pgx.Tx` `[G5]`
- [ ] **P3-T06** Service `Ledger.Post(tx, entries)`: kiểm tra tổng = 0, cập nhật `balance` + `balance_after` `[P5]`
- [ ] **P3-T07** Middleware idempotency: lưu key + request hash + response trong cùng transaction `[P6]`
- [ ] **P3-T08** Tái hiện lost update khi hai giao dịch cập nhật cùng ví ở READ COMMITTED; chọn cách chặn (`SELECT ... FOR UPDATE` / update có điều kiện / SERIALIZABLE + retry) `[P4]`
- [ ] **P3-T09** Mock payment provider: trang thanh toán giả, gửi webhook ký HMAC, có chế độ gửi lặp / trễ / mất
- [ ] **P3-T10** API topup, webhook, số dư, lịch sử
- [ ] **P3-T11** Worker hết hạn topup PENDING

### bff
- [ ] **P3-T12** Route `/api/wallet*`, truyền `Idempotency-Key` từ client xuống core

### web
- [ ] **P3-T13** Trang ví: số dư, nạp tiền, lịch sử (infinite scroll); sinh `Idempotency-Key` một lần mỗi lần bấm

### qa
- [ ] **P3-T14** Integration test song song nhiều giao dịch cùng ví với testcontainers + `-race` `[G9]`
- [ ] **P3-T15** Property test cho `Money` và `Ledger.Post` `[G6]`
- [ ] **P3-T16** Script kiểm tra bất biến ledger (dùng lại ở phase 8)

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| G5 | Golang | Ranh giới transaction giữa các module | Use case đặt vé đụng ghế + ví + outbox | `postgres.WithTx`, truyền `pgx.Tx` qua interface phía dùng; logic tính toán tách thành hàm thuần | Một use case = một transaction; logic giá/hoàn tiền test không cần DB | ⬜ |
| G6 | Golang | Kiểu tiền an toàn | Toàn bộ ví, giá vé | Value object `Money` (int64), cấm phép tính float, kiểm tra overflow | Lint/test chặn float; property test cho cộng/trừ | ⬜ |
| G9 | Golang | Test với DB thật | Logic khoá ghế, ledger | testcontainers-go, test song song race condition, `go test -race` | Test N goroutine tranh 1 ghế → đúng 1 thành công | ⬜ |
| P4 | PostgreSQL core | Isolation level & anomaly | Ví: lost update, write skew | Hiểu RC / RR / Serializable; tái hiện anomaly bằng test; retry `40001` | Có test tái hiện và test chứng minh đã chặn | ⬜ |
| P5 | PostgreSQL core | Ledger chính xác tuyệt đối | Nạp, trả, hoàn tiền | Double-entry, `CHECK` constraint, entry bất biến, snapshot balance | Job đối soát luôn ra chênh lệch 0 | ⬜ |
| P6 | PostgreSQL core | Idempotency | Retry, double click, webhook lặp | Bảng `idempotency_keys` ghi cùng transaction, `INSERT ... ON CONFLICT` | Gửi 1 request 100 lần → 1 lần trừ tiền | ⬜ |

## Definition of Done

- [ ] Nạp tiền end-to-end qua mock provider trên web
- [ ] Webhook gửi lặp 10 lần → cộng tiền 1 lần
- [ ] 100 request cùng `Idempotency-Key` song song → 1 giao dịch
- [ ] Script bất biến: tổng entries = 0, `balance` = tổng entries cho mọi tài khoản
- [ ] Có test tái hiện lost update và test chứng minh đã chặn
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [database](../../database.md) (phần ví), [key-problems](../../key-problems.md) mục tính tiền
- [ ] ADR: cách chặn lost update trên ví (kèm so sánh), thiết kế idempotency
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
