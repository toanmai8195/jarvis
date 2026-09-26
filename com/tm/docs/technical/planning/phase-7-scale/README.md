# Phase 7 — Chịu tải & tối ưu

## Mục tiêu

Đạt toàn bộ chỉ tiêu phi chức năng trong [architecture](../../architecture.md#yêu-cầu-phi-chức-năng) bằng đo đạc có hệ thống: profile, tuning PG, connection pooling, read replica, cache, rate limit, hàng đợi ảo — và chứng minh hệ thống vẫn đúng khi thành phần phụ trợ hỏng.

**Mốc demo**: kịch bản k6 "mở bán Tết" (traffic gấp 10 lần bình thường) chạy 30 phút: đạt chỉ tiêu, 0 vé trùng, 0 sai tiền; tắt Redis giữa chừng hệ thống vẫn đúng.

## Phạm vi

- **Trong**: load test, profiling Go và Node, PgBouncer, read replica, autovacuum, cache BFF, rate limit, Redis pre-check ghế, waiting room.
- **Ngoài**: tính năng nghiệp vụ mới.

## Workstream

`qa` · `core` · `db` · `bff` · `infra`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P7-NFR1 | NFR | Tìm chuyến ≥ 5.000 req/s, p99 < 150ms |
| P7-NFR2 | NFR | Đặt vé ≥ 1.000 booking/s, p99 < 300ms |
| P7-NFR3 | NFR | Scale core lên 10 instance không lỗi connection |
| P7-NFR4 | NFR | Chạy tải 24h: bộ nhớ Go/Node ổn định, bloat PG ổn định |
| P7-NFR5 | NFR | Traffic gấp 10 lần → hàng đợi ảo giữ p99 của người đã vào trong ngưỡng |
| P7-NFR6 | NFR | Redis chết → hệ thống chậm hơn nhưng 0 vé trùng, 0 sai tiền |
| P7-FR1 | FR | Rate limit theo bảng trong [api](../../api.md#rate-limit-bff), đồng nhất giữa các instance BFF |

## Task

### qa
- [ ] **P7-T01** Bộ kịch bản k6: search, booking thường, mở bán Tết, soak 24h; dashboard Grafana cho k6
- [ ] **P7-T02** Đo baseline, lưu `loadtest/results/phase-7/baseline`

### core
- [ ] **P7-T03** pprof CPU/heap/mutex/block dưới tải, tối ưu điểm nóng, ghi trước/sau `[G8]`
- [ ] **P7-T03a** Đặt giới hạn CPU/RAM cho container core; cấu hình `GOMAXPROCS`, `GOMEMLIMIT`; đọc `GODEBUG=gctrace=1` `[G13]`
- [ ] **P7-T04** Tách pool đọc (replica) / ghi (primary); định tuyến đọc-sau-ghi về primary `[P11]`
- [ ] **P7-T05** Redis pre-check trạng thái ghế, đồng bộ từ sự kiện; fallback về PG khi Redis lỗi `[D1][D3]`
- [ ] **P7-T06** Waiting room: sorted set cấp token theo tốc độ, endpoint kiểm tra vị trí `[D2]`

### db
- [ ] **P7-T07** PgBouncer transaction mode; tính kích thước pool tối ưu; kiểm tra tương thích prepared statement của pgx `[P9]`
- [ ] **P7-T08** Streaming replication primary → replica trong Docker Compose; đo replication lag `[P11]`
- [ ] **P7-T09** Tuning autovacuum, fillfactor cho `trip_seats`, `seat_holds`; theo dõi dead tuples `[P12]`
- [ ] **P7-T10** Tuning `shared_buffers`, `work_mem`, `max_connections` và ghi lý do

### bff
- [ ] **P7-T11** Đo event loop lag, chuyển xử lý nặng (xuất CSV, format lớn) khỏi main thread `[N1]`
- [ ] **P7-T12** Rate limit sliding window trên Redis `[N5]`
- [ ] **P7-T13** Cache tìm chuyến: TTL ngắn + stale-while-revalidate + single-flight chống stampede `[N7]`
- [ ] **P7-T14** Heap snapshot trước/sau soak test `[N9]`

### infra
- [ ] **P7-T15** Chạy nhiều instance core, bff sau load balancer (nginx/traefik) trong Compose
- [ ] **P7-T16** Kịch bản chaos: kill Redis, kill 1 instance core, làm chậm PG (toxiproxy)

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| G8 | Golang | Tìm nút thắt hiệu năng | Latency tăng khi tải cao | pprof (CPU, heap, mutex, block), trace, benchmark | Có báo cáo profile trước/sau tối ưu trong ADR | ⬜ |
| P9 | PostgreSQL core | Connection là tài nguyên khan hiếm | Nhiều instance core + worker | pgxpool + PgBouncer transaction mode, tính số connection tối ưu | Không lỗi `too many connections` khi scale 10 instance | ⬜ |
| P11 | PostgreSQL core | Mở rộng đọc | Tìm chuyến, lịch sử | Streaming replication, read replica, xử lý replication lag | Đọc từ replica, đọc-sau-ghi vẫn đúng | ⬜ |
| P12 | PostgreSQL core | Vacuum & bloat | `trip_seats`, `seat_holds` update liên tục | Tuning autovacuum theo bảng, fillfactor, theo dõi dead tuples | Bloat ổn định sau 24h load test | ⬜ |
| N1 | Node.js | Không chặn event loop | Gom dữ liệu, format JSON lớn | Tránh CPU-bound trên main thread, đo event loop lag, worker_threads khi cần | Event loop lag p99 < 50ms dưới tải | ⬜ |
| N5 | Node.js | Rate limiting phân tán | Nhiều instance BFF | Sliding window trên Redis | Vượt ngưỡng → 429 đồng nhất giữa các instance | ⬜ |
| N7 | Node.js | Cache đọc | Tìm chuyến, chi tiết chuyến | Cache ngắn + stale-while-revalidate, chống cache stampede | Giảm ≥ 70% request tìm chuyến tới core | ⬜ |
| N9 | Node.js | Memory leak | Process chạy lâu | Heap snapshot, `--inspect`, theo dõi RSS | RSS ổn định sau 24h | ⬜ |
| D1 | Redis | Giảm tải PG khi mở bán | Pre-check ghế trống | Bitmap/hash trạng thái ghế, đồng bộ với PG | PG nhận ít request tranh chấp hơn, dữ liệu không lệch | ⬜ |
| D2 | Redis | Hàng đợi ảo | Sự kiện mở bán lớn | Sorted set cấp token theo tốc độ | Hệ thống giữ p99 ổn định khi traffic gấp 10 lần | ⬜ |
| D3 | Redis | Redis không phải nguồn sự thật | Redis mất dữ liệu | Thiết kế để Redis chết → hệ thống chậm hơn nhưng vẫn đúng | Tắt Redis giữa load test: 0 vé trùng, 0 sai tiền | ⬜ |
| G13 | Golang | Go runtime trong container | Pod bị giới hạn CPU/RAM | `GOMAXPROCS` theo CPU quota, `GOMEMLIMIT`, đọc GC trace | Không bị throttle CPU/OOMKill dưới tải; có số liệu GC trước/sau | ⬜ |

## Definition of Done

- [ ] Đạt P7-NFR1..6, có báo cáo k6 lưu trong repo
- [ ] Mỗi tối ưu có số liệu trước/sau
- [ ] Kịch bản chaos pass: 0 vé trùng, script bất biến ledger = 0
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [architecture](../../architecture.md) (chỉ tiêu đạt được thực tế, sơ đồ có replica/PgBouncer)
- [ ] ADR: cấu hình pool, chiến lược cache, thiết kế waiting room
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
