# Phase 13 — *(Tuỳ chọn)* Tách wallet thành service riêng

> Chặng D — Tổng hợp · Công nghệ: **Go + PG + gRPC**

## Mục tiêu

Kiểm chứng [ADR-0001](../../adr/0001-modular-monolith-core.md) bằng số liệu: tách module `wallet` khỏi core thành service riêng có DB riêng, chuyển luồng đặt vé sang saga, rồi so sánh với bản modular monolith.

**Mốc demo**: Saga core ↔ wallet; bảng so sánh với monolith

## Kiến thức trọng tâm

Go: gRPC, saga · PG: DB riêng, đối soát chéo

## Phạm vi

- **Trong**: Service `wallet` riêng, gRPC, saga đặt vé/huỷ vé, đo đạc.
- **Ngoài**: Tách catalog, booking.

## Workstream

`core / wallet` · `infra` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P13-FR1 | FR | `wallet` là service riêng, DB riêng; core không truy cập DB của wallet |
| P13-FR2 | FR | Đặt vé theo saga: hold → reserve tiền → xác nhận vé → capture tiền; lỗi ở bước nào thì bù trừ bước trước |
| P13-FR3 | FR | Huỷ vé và hoàn tiền hoạt động qua service mới |
| P13-FR4 | FR | Retry bất kỳ bước nào của saga không gây trừ/hoàn tiền trùng |
| P13-NFR1 | NFR | 0 vé trùng, 0 sai tiền dưới k6 + chaos (kill wallet giữa saga) |
| P13-NFR2 | NFR | Có báo cáo so sánh với baseline Phase 11 |

## Task

### core / wallet
- [ ] **P13-T01** Định nghĩa `wallet.proto`: `Reserve`, `Capture`, `Release`, `Refund`, `GetBalance`; quy tắc tương thích ngược `[S2]` _(trước đây P9-T01)_
- [ ] **P13-T02** Tạo `services/wallet`, chuyển code từ `core/internal/wallet`; migration DB riêng _(trước đây P9-T02)_
- [ ] **P13-T03** Thay implement `debiter` trong core bằng gRPC client — xác nhận interface phía dùng giúp đổi mà không sửa `booking` _(trước đây P9-T03)_
- [ ] **P13-T04** Saga đặt vé (orchestration trong core), trạng thái saga lưu trong PG core, bước bù trừ `[S1]` _(trước đây P9-T04)_
- [ ] **P13-T05** Idempotency key truyền xuyên service cho mọi bước saga `[S3]` _(trước đây P9-T05)_
- [ ] **P13-T06** Đối soát giữa ledger wallet và booking core `[S3]` _(trước đây P9-T06)_

### infra
- [ ] **P13-T07** Deploy wallet riêng, trace xuyên core → wallet _(trước đây P9-T07)_

### qa
- [ ] **P13-T08** Chaos: kill wallet sau Reserve, trước Capture; mạng chập chờn giữa core ↔ wallet _(trước đây P9-T08)_
- [ ] **P13-T09** Chạy lại bộ k6 phase 7, lập bảng so sánh `[S4]` _(trước đây P9-T09)_

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| S2 | Microservice | Hợp đồng gRPC tiến hoá | core và wallet deploy lệch nhau | Protobuf tương thích ngược, `buf breaking` trong CI | Deploy wallet mới trước core cũ vẫn chạy | ⬜ |
| S1 | Microservice | Saga thay cho transaction ACID | Đặt vé trải trên 2 DB | Orchestration, trạng thái saga bền vững, bước bù trừ, reserve/capture | Mọi điểm lỗi trong saga đều hội tụ về trạng thái đúng | ⬜ |
| S3 | Microservice | Tính đúng đắn xuyên service | Retry, timeout không rõ kết quả | Idempotency key xuyên service, đối soát chéo | Chaos test: 0 sai tiền | ⬜ |
| S4 | Microservice | Đo cái giá của việc tách | Quyết định kiến trúc cần số liệu | Cùng bộ k6, so sánh latency, throughput, code, case lỗi | Có bảng so sánh và kết luận trong ADR | ⬜ |

## Definition of Done

- [ ] Đặt vé, huỷ vé chạy qua wallet service
- [ ] Chaos test pass: 0 vé trùng, đối soát chênh lệch 0
- [ ] Báo cáo so sánh với phase 7 và cập nhật kết luận vào ADR-0001 (hoặc ADR mới thay thế)
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] ADR: kết luận monolith vs microservice cho snaptix
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
