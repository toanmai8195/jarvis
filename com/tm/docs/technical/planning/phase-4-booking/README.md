# Phase 4 — Giữ chỗ & đặt vé

## Mục tiêu

Luồng cốt lõi: giữ ghế có thời hạn, đặt vé thanh toán bằng ví trong một transaction, không bán trùng dưới tranh chấp cao; sơ đồ ghế realtime trên web.

**Mốc demo**: hai trình duyệt cùng chọn một ghế — chỉ một người giữ được, người kia thấy ghế đổi trạng thái ngay; đặt vé, nhận vé QR, số dư giảm đúng.

## Phạm vi

- **Trong**: `seat_holds`, `bookings`, `tickets`, `outbox_events` (chỉ ghi, chưa relay); API hold/booking; worker hết hạn hold; SSE trạng thái ghế; UI chọn ghế, checkout, vé của tôi.
- **Ngoài**: huỷ vé (phase 8), relay outbox sang analytics (phase 6), tối ưu cho tải cực lớn (phase 7).

## Workstream

`db` · `core` · `bff` · `web` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P4-FR1 | FR | Giữ 1..N ghế của một chuyến trong 10 phút; tất cả hoặc không ghế nào |
| P4-FR2 | FR | Trả ghế chủ động; hold hết hạn tự trả ghế |
| P4-FR3 | FR | Đặt vé từ hold: kiểm tra hold, tính giá phía server, trừ ví, phát hành vé, ghi outbox — **một transaction** |
| P4-FR4 | FR | Số dư không đủ → `INSUFFICIENT_BALANCE`, hold vẫn giữ nguyên |
| P4-FR5 | FR | Sơ đồ ghế cập nhật realtime cho mọi người đang xem chuyến |
| P4-FR6 | FR | Vé của tôi: danh sách, chi tiết, mã QR |
| P4-NFR1 | NFR | 0 ghế bán trùng dưới mọi mức tranh chấp |
| P4-NFR2 | NFR | 500 booking/s trên máy dev, p99 < 300ms (mục tiêu 1.000/s ở phase 7) |
| P4-NFR3 | NFR | Bấm đặt vé nhiều lần → một đơn, một lần trừ tiền |

## Task

### db
- [ ] **P4-T01** Migration `seat_holds`, `bookings`, `tickets`, `outbox_events`, `refunds` (bảng rỗng cho phase 8)
- [ ] **P4-T02** Unique index `tickets (trip_id, seat_id) WHERE status <> 'CANCELLED'`; partial index hold ACTIVE `[P1]`

### core
- [ ] **P4-T03** Use case hold: update có điều kiện, khoá theo thứ tự `seat_id`, kiểm tra số dòng `[P1][P3]`
- [ ] **P4-T04** Triển khai 3 chiến lược khoá sau interface (`FOR UPDATE`, `SKIP LOCKED`, optimistic version) để benchmark `[P2]`
- [ ] **P4-T05** Use case booking trong một transaction (`WithTx`, idempotency từ phase 3); `booking` gọi `wallet` qua interface `debiter` khai báo phía booking `[P1][G5]`
- [ ] **P4-T06** Cập nhật `trips.available_seats` trong cùng transaction
- [ ] **P4-T07** Giới hạn concurrency vào DB (semaphore theo kích thước pool), trả 503 nhanh khi quá tải `[G1]`
- [ ] **P4-T08** Worker hết hạn hold chạy nhiều instance bằng `SKIP LOCKED` + `errgroup` + backoff `[G4]`
- [ ] **P4-T09** Phát thay đổi trạng thái ghế (LISTEN/NOTIFY hoặc Redis pub/sub) để BFF đẩy SSE
- [ ] **P4-T10** Retry có giới hạn khi deadlock (`40P01`) / serialization (`40001`) `[P3]`

### bff
- [ ] **P4-T11** Route `/api/holds`, `/api/bookings*`
- [ ] **P4-T12** SSE `/api/trips/{id}/seats/stream`: fan-out theo chuyến, heartbeat, dọn kết nối khi client rời, backpressure `[N6]`

### web
- [ ] **P4-T13** Component sơ đồ ghế: mỗi ghế memo hoá, store theo ghế, cập nhật từ SSE `[R1]`
- [ ] **P4-T14** Optimistic chọn ghế, rollback khi `SEAT_UNAVAILABLE` `[R2]`
- [ ] **P4-T15** Checkout bằng state machine: chọn ghế → hành khách → thanh toán → kết quả; đồng hồ đếm ngược; khôi phục khi refresh `[R3]`
- [ ] **P4-T16** Invalidate query ví, vé sau thanh toán `[R9]`
- [ ] **P4-T17** Trang vé của tôi + mã QR

### qa
- [ ] **P4-T18** Test N goroutine tranh 1 ghế → đúng 1 thành công `[G9][P1]`
- [ ] **P4-T19** k6: 1.000 user tranh 40 ghế; 5.000 user đặt trên 500 chuyến `[G1][P2]`
- [ ] **P4-T20** Benchmark 3 chiến lược khoá, ghi ADR `[P2]`

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| G1 | Golang | Xử lý hàng nghìn request đặt vé đồng thời | Mở bán Tết | Goroutine per request, giới hạn concurrency bằng semaphore, pgxpool đúng kích thước | k6 500 booking/s, p99 < 300ms, không leak goroutine | ⬜ |
| G4 | Golang | Worker chạy nhiều instance an toàn | Hết hạn hold, relay outbox, sinh slot | `FOR UPDATE SKIP LOCKED`, `errgroup`, backoff, leader election (advisory lock) cho job cần duy nhất | 3 instance worker, không xử lý trùng, không bỏ sót | ⬜ |
| P1 | PostgreSQL core | Không bán trùng ghế | `trip_seats` là bảng nóng nhất | Update có điều kiện, khoá theo thứ tự, unique index chốt chặn | Load test tranh ghế: 0 vé trùng | ⬜ |
| P2 | PostgreSQL core | Chọn chiến lược khoá | Nhiều cách cùng đúng, khác hiệu năng | So sánh `FOR UPDATE`, `SKIP LOCKED`, optimistic version, advisory lock | ADR có số liệu throughput/latency từng cách | ⬜ |
| P3 | PostgreSQL core | Deadlock | Nhiều user chọn nhiều ghế chồng nhau | Khoá theo thứ tự `seat_id`, transaction ngắn, retry có giới hạn | Không có deadlock trong log khi load test | ⬜ |
| N6 | Node.js | Streaming realtime | Sơ đồ ghế cập nhật | SSE, backpressure, dọn kết nối khi client rời | 10.000 kết nối SSE đồng thời ổn định bộ nhớ | ⬜ |
| R1 | React | Sơ đồ ghế realtime hiệu năng cao | Toa tàu hàng trăm ghế, cập nhật liên tục | Memo hoá từng ghế, cập nhật state theo ghế, virtualize khi lớn | Không render lại toàn bộ sơ đồ khi 1 ghế đổi trạng thái | ⬜ |
| R2 | React | Optimistic UI & rollback | Chọn ghế, huỷ vé | TanStack Query mutation, rollback khi `SEAT_UNAVAILABLE` | UI luôn khớp server sau lỗi | ⬜ |
| R3 | React | Luồng checkout nhiều bước | Chọn ghế → hành khách → thanh toán | State machine (useReducer / XState), đồng hồ đếm ngược hold | Refresh/quay lại không mất trạng thái, không thanh toán 2 lần | ⬜ |
| R9 | React | Dữ liệu không cũ sau thanh toán | Số dư, vé của tôi | Chiến lược `staleTime`, invalidate query theo key sau mutation | Không bao giờ hiển thị số dư/vé cũ sau thanh toán | ⬜ |

## Definition of Done

- [ ] Demo hai trình duyệt tranh một ghế hoạt động đúng
- [ ] k6 tranh ghế: 0 vé trùng, không deadlock trong log
- [ ] Đạt P4-NFR2 trên máy dev
- [ ] ADR so sánh chiến lược khoá có số liệu; chọn một chiến lược mặc định
- [ ] Mỗi booking có đúng một sự kiện `booking.confirmed` trong `outbox_events`
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [services](../../services.md) (luồng đặt vé), [key-problems](../../key-problems.md) mục chống bán trùng
- [ ] ADR: chiến lược khoá ghế; cơ chế phát trạng thái ghế realtime
- [ ] Lưu kết quả k6 vào `loadtest/results/phase-4/`
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
