# Phase 3 — Giữ chỗ & đặt vé (core)

> Chặng A — Go + PostgreSQL (core) · Công nghệ: **Go + PostgreSQL**

## Mục tiêu

Bài toán tranh chấp: giữ ghế, đặt vé trong một transaction, không bán trùng, deadlock, worker nhiều instance.

**Mốc demo**: k6 tranh ghế: 0 vé trùng; đặt vé bằng `curl` trừ ví đúng

## Kiến thức trọng tâm

Go: goroutine, errgroup, semaphore, worker · PG: update có điều kiện, FOR UPDATE / SKIP LOCKED, deadlock, LISTEN/NOTIFY

## Phạm vi

- **Trong**: Hold, booking, ticket, outbox (chỉ ghi), worker hết hạn hold, phát thay đổi trạng thái ghế.
- **Ngoài**: SSE (Phase 8), giao diện chọn ghế (Phase 9).

## Workstream

`db` · `core` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P3-FR1 | FR | Giữ 1..N ghế của một chuyến trong 10 phút; tất cả hoặc không ghế nào |
| P3-FR2 | FR | Trả ghế chủ động; hold hết hạn tự trả ghế |
| P3-FR3 | FR | Đặt vé từ hold: kiểm tra hold, tính giá phía server, trừ ví, phát hành vé, ghi outbox — **một transaction** |
| P3-FR4 | FR | Số dư không đủ → `INSUFFICIENT_BALANCE`, hold vẫn giữ nguyên |
| P3-NFR1 | NFR | 0 ghế bán trùng dưới mọi mức tranh chấp |
| P3-NFR2 | NFR | 500 booking/s trên máy dev, p99 < 300ms (mục tiêu 1.000/s ở Phase 11) |
| P3-NFR3 | NFR | Bấm đặt vé nhiều lần → một đơn, một lần trừ tiền |

## Task

### db
- [ ] **P3-T01** Migration `seat_holds`, `bookings`, `tickets`, `outbox_events`, `refunds` (`refunds` dùng ở Phase 4) _(trước đây P4-T01)_
- [ ] **P3-T02** Unique index `tickets (trip_id, seat_id) WHERE status <> 'CANCELLED'`; partial index hold ACTIVE `[P1]` _(trước đây P4-T02)_

### core
- [ ] **P3-T03** Use case hold: update có điều kiện, khoá theo thứ tự `seat_id`, kiểm tra số dòng `[P1][P3]` _(trước đây P4-T03)_
- [ ] **P3-T04** Triển khai 3 chiến lược khoá sau interface (`FOR UPDATE`, `SKIP LOCKED`, optimistic version) để benchmark `[P2]` _(trước đây P4-T04)_
- [ ] **P3-T05** Use case booking trong một transaction (`WithTx`, idempotency từ Phase 2); `booking` gọi `wallet` qua interface `debiter` khai báo phía booking `[P1][G5]` _(trước đây P4-T05)_
- [ ] **P3-T06** Cập nhật `trips.available_seats` trong cùng transaction _(trước đây P4-T06)_
- [ ] **P3-T07** Giới hạn concurrency vào DB (semaphore theo kích thước pool), trả 503 nhanh khi quá tải `[G1]` _(trước đây P4-T07)_
- [ ] **P3-T08** Worker hết hạn hold chạy nhiều instance bằng `SKIP LOCKED` + `errgroup` + backoff `[G4]` _(trước đây P4-T08)_
- [ ] **P3-T09** Phát thay đổi trạng thái ghế (LISTEN/NOTIFY hoặc Redis pub/sub) để BFF đẩy SSE _(trước đây P4-T09)_
- [ ] **P3-T10** Retry có giới hạn khi deadlock (`40P01`) / serialization (`40001`) `[P3]` _(trước đây P4-T10)_

### qa
- [ ] **P3-T11** Test N goroutine tranh 1 ghế → đúng 1 thành công `[G9][P1]` _(trước đây P4-T18)_
- [ ] **P3-T12** k6: 1.000 user tranh 40 ghế; 5.000 user đặt trên 500 chuyến `[G1][P2]` _(trước đây P4-T19)_
- [ ] **P3-T13** Benchmark 3 chiến lược khoá, ghi ADR `[P2]` _(trước đây P4-T20)_

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| P1 | PostgreSQL core | Không bán trùng ghế | `trip_seats` là bảng nóng nhất | Update có điều kiện, khoá theo thứ tự, unique index chốt chặn | Load test tranh ghế: 0 vé trùng | ⬜ |
| P3 | PostgreSQL core | Deadlock | Nhiều user chọn nhiều ghế chồng nhau | Khoá theo thứ tự `seat_id`, transaction ngắn, retry có giới hạn | Không có deadlock trong log khi load test | ⬜ |
| P2 | PostgreSQL core | Chọn chiến lược khoá | Nhiều cách cùng đúng, khác hiệu năng | So sánh `FOR UPDATE`, `SKIP LOCKED`, optimistic version, advisory lock | ADR có số liệu throughput/latency từng cách | ⬜ |
| G1 | Golang | Xử lý hàng nghìn request đặt vé đồng thời | Mở bán Tết | Goroutine per request, giới hạn concurrency bằng semaphore, pgxpool đúng kích thước | k6 1.000 booking/s, p99 < 300ms, không leak goroutine | ⬜ |
| G4 | Golang | Worker chạy nhiều instance an toàn | Hết hạn hold, relay outbox, sinh slot | `FOR UPDATE SKIP LOCKED`, `errgroup`, backoff, leader election (advisory lock) cho job cần duy nhất | 3 instance worker, không xử lý trùng, không bỏ sót | ⬜ |

## Definition of Done

- [ ] k6 tranh ghế: 0 vé trùng, không deadlock trong log
- [ ] Đạt 500 booking/s trên máy dev
- [ ] ADR so sánh chiến lược khoá có số liệu
- [ ] Mỗi booking có đúng một sự kiện `booking.confirmed` trong outbox
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [services](../../services.md), [key-problems](../../key-problems.md)
- [ ] ADR: chiến lược khoá ghế
- [ ] Lưu kết quả k6 vào `loadtest/results/phase-3/`
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
