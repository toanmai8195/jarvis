# Phase 8 — Huỷ/hoàn vé, đối soát, hardening

## Mục tiêu

Hoàn thiện vòng đời vé (huỷ, hoàn tiền theo chính sách, huỷ chuyến hàng loạt), đối soát tiền tự động, migration không downtime, accessibility và bộ E2E đầy đủ.

**Mốc demo**: huỷ vé → tiền hoàn vào ví theo chính sách; admin huỷ chuyến → toàn bộ khách được hoàn 100%; job đối soát báo chênh lệch 0; E2E đầy đủ chạy xanh trong CI.

## Phạm vi

- **Trong**: huỷ vé, refund quote, hoàn tiền thủ công, huỷ chuyến, job đối soát, migration expand–contract dưới tải, accessibility, E2E Playwright.
- **Ngoài**: rút tiền khỏi ví, tích hợp cổng thanh toán thật.

## Workstream

`core` · `db` · `bff` · `web` · `admin` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P8-FR1 | FR | Xem trước số tiền hoàn theo chính sách (≥24h: 90%, 4–24h: 50%, <4h: 0%) |
| P8-FR2 | FR | Huỷ vé: vé CANCELLED, ghế AVAILABLE, hoàn tiền vào ví, outbox — một transaction |
| P8-FR3 | FR | Huỷ một phần vé trong đơn → đơn PARTIALLY_CANCELLED |
| P8-FR4 | FR | Admin huỷ chuyến → hoàn 100% mọi vé, chạy theo batch, tiếp tục được nếu gián đoạn |
| P8-FR5 | FR | Support hoàn tiền thủ công có lý do, ghi audit log |
| P8-FR6 | FR | Job đối soát hằng giờ: bất biến ledger, balance, topup vs provider; cảnh báo khi lệch |
| P8-NFR1 | NFR | Migration trên bảng lớn chạy dưới tải không khoá bảng |
| P8-NFR2 | NFR | Web client đạt Lighthouse accessibility ≥ 95 |
| P8-NFR3 | NFR | E2E toàn bộ luồng chính chạy trong CI < 10 phút |

## Task

### core
- [ ] **P8-T01** Chính sách hoàn tiền dạng cấu hình, tính theo thời điểm huỷ so với giờ khởi hành
- [ ] **P8-T02** Use case huỷ vé (một hoặc nhiều vé), idempotent
- [ ] **P8-T03** Huỷ chuyến: job batch có checkpoint, idempotent theo ticket
- [ ] **P8-T04** Hoàn tiền thủ công (giới hạn không vượt số đã trả)
- [ ] **P8-T05** Job đối soát: bất biến ledger, `balance` vs tổng entries, topup SUCCEEDED vs báo cáo mock provider; xuất metric + cảnh báo Grafana

### db
- [ ] **P8-T06** Thực hành expand–contract: thêm cột mới vào `tickets` (ví dụ `cancel_reason`), backfill theo batch, `CREATE INDEX CONCURRENTLY`, chạy dưới k6 `[P10]`
- [ ] **P8-T07** Kiểm tra lock bằng `lock_timeout`, `statement_timeout` cho migration `[P10]`

### bff
- [ ] **P8-T08** Route huỷ vé, refund quote, admin refund, admin huỷ chuyến

### web / admin
- [ ] **P8-T09** UI huỷ vé với refund quote; UI huỷ chuyến + tiến độ; UI hoàn tiền thủ công
- [ ] **P8-T10** Accessibility: sơ đồ ghế điều hướng bằng bàn phím, ARIA, focus management trong checkout và dialog `[R7]`

### qa
- [ ] **P8-T11** Bộ Playwright E2E: đăng nhập → nạp tiền → đặt vé → huỷ vé; admin setup → bán → huỷ chuyến `[R8]`
- [ ] **P8-T12** Test Testing Library cho component quan trọng (sơ đồ ghế, checkout, ví) `[R8]`
- [ ] **P8-T13** Chạy lại toàn bộ test nghiệm thu các phase trước (regression)

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| P10 | PostgreSQL core | Migration không downtime | Thêm cột/index trên bảng lớn | `CREATE INDEX CONCURRENTLY`, expand–contract, backfill theo batch | Migration chạy dưới tải không khoá bảng | ⬜ |
| R7 | React | Accessibility | Chọn ghế bằng bàn phím, screen reader | ARIA, focus management | Lighthouse accessibility ≥ 95 | ⬜ |
| R8 | React | Test UI | Luồng đặt vé | Testing Library, Playwright E2E | E2E đặt vé → huỷ vé chạy trong CI | ⬜ |

Ngoài ra phase này củng cố lại **P5** (ledger) và **P6** (idempotency) cho luồng hoàn tiền.

## Definition of Done

- [ ] Huỷ vé, huỷ chuyến, hoàn tiền thủ công hoạt động trên UI
- [ ] Job đối soát chạy hằng giờ, báo chênh lệch 0 sau toàn bộ test
- [ ] Migration expand–contract chạy dưới tải, không có request lỗi do lock
- [ ] Đạt P8-NFR2, P8-NFR3
- [ ] Regression toàn bộ phase trước pass
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [client guide](../../../user-guide/client.md), [admin guide](../../../user-guide/admin.md) phần huỷ/hoàn
- [ ] Cập nhật toàn bộ docs technical theo code thực tế (bỏ ghi chú "bản thiết kế")
- [ ] ADR: quy trình migration không downtime
- [ ] Viết [lessons-learned.md](lessons-learned.md) và tổng kết toàn dự án
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
