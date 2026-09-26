# Phase 1 — Catalog & tìm chuyến

## Mục tiêu

Xây domain catalog (trạm, tuyến, phương tiện, sơ đồ ghế), chuyến và giá trong core Go dưới dạng module của modular monolith, viết Go đúng chất Go (không mang thói quen Java); API tìm chuyến nhanh trên dữ liệu lớn.

**Mốc demo**: seed 10 triệu `trip_seats`, gọi `GET /internal/v1/trips/search` trả kết quả đúng với p99 query < 20ms.

## Phạm vi

- **Trong**: schema catalog + trips + fares + trip_seats, repository sqlc, API đọc (search, trip detail, seats), seed script, chuẩn lỗi.
- **Ngoài**: CRUD admin (phase 5), hold/booking (phase 4), auth (phase 2).

## Workstream

`db` · `core` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P1-FR1 | FR | Tìm chuyến theo điểm đi, điểm đến, ngày, loại phương tiện; hỗ trợ đoạn giữa tuyến (đi từ trạm 2 đến trạm 4) |
| P1-FR2 | FR | Kết quả có giờ đi/đến dự kiến, giá thấp nhất, số ghế trống |
| P1-FR3 | FR | Xem chi tiết chuyến và giá theo hạng ghế |
| P1-FR4 | FR | Xem sơ đồ ghế kèm trạng thái |
| P1-FR5 | FR | Lỗi trả theo định dạng chuẩn trong [API](../../api.md#lỗi) |
| P1-NFR1 | NFR | p99 query tìm chuyến < 20ms trên 10 triệu `trip_seats` |
| P1-NFR2 | NFR | Mọi truy vấn DB tôn trọng deadline của request |

## Task

### db
- [ ] **P1-T01** Migration: `stations`, `routes`, `route_stops`, `seat_layouts`, `layout_seats`, `vehicles` theo [database](../../database.md)
- [ ] **P1-T02** Migration: `fares`, `schedules`, `trips`, `trip_seats`
- [ ] **P1-T03** Index cho tìm chuyến: `trips (route_id, departure_at)`, index tìm tuyến theo cặp trạm `[P7]`
- [ ] **P1-T04** Seed script sinh dữ liệu lớn: 200 trạm, 500 tuyến, 1 năm lịch chạy, 10 triệu `trip_seats`
- [ ] **P1-T05** Viết query search bằng sqlc; đọc `EXPLAIN (ANALYZE, BUFFERS)`, tối ưu đến khi đạt NFR `[P7]`

### core
- [ ] **P1-T06** Module `internal/catalog` theo vertical slice (types, hàm thuần, service, store, http); `internal/httpx`; theo [project-structure](../../project-structure.md#module-trong-core-modular-monolith) `[G11]`
- [ ] **P1-T06a** Wiring thủ công trong `cmd/server/main.go` `[G12]`
- [ ] **P1-T06b** Kiểm tra ranh giới module tự động (Bazel visibility hoặc test import) theo [ADR-0001](../../adr/0001-modular-monolith-core.md) `[G11]`
- [ ] **P1-T07** Domain error + mapping sang HTTP/`error.code` `[G7]`
- [ ] **P1-T08** Truyền `context.Context` từ handler → use case → repository, đặt timeout per-request `[G2]`
- [ ] **P1-T09** API `GET /internal/v1/trips/search`, `/trips/{id}`, `/trips/{id}/seats`
- [ ] **P1-T10** Tính giá thấp nhất theo đoạn và hạng ghế từ `fares` hợp lệ tại thời điểm hiện tại
- [ ] **P1-T11** OpenAPI `com/tm/server/api/core.openapi.yaml` cho các endpoint trên

### qa
- [ ] **P1-T12** Unit test domain (tính giờ đến, chọn giá)
- [ ] **P1-T13** Integration test repository với testcontainers
- [ ] **P1-T14** Benchmark query search trên dữ liệu seed

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| G2 | Golang | Timeout & huỷ lan truyền | Client đóng tab giữa chừng, PG chậm | `context.Context` xuyên suốt handler → use case → repository; deadline cho mọi I/O | Không còn query chạy sau khi request bị huỷ | ⬜ |
| G7 | Golang | Lỗi nghiệp vụ rõ ràng | Map sang HTTP code/`error.code` | Sentinel error + `errors.Is/As`, wrap có ngữ cảnh | Mọi lỗi trả ra đúng code trong [API](../../api.md#lỗi) | ⬜ |
| P7 | PostgreSQL core | Truy vấn tìm chuyến nhanh | 5.000 req/s | Composite index, denormalize `available_seats`, đọc `EXPLAIN (ANALYZE, BUFFERS)` | p99 query < 20ms trên 10 triệu `trip_seats` | ⬜ |
| G11 | Golang | Package theo nghiệp vụ, interface phía dùng | Thói quen Java: chia tầng, `IFoo` + `FooImpl` | Vertical slice trong module; interface nhỏ khai báo ở package dùng nó; nhận interface, trả struct | Không có package `service/`, `repository/`; không interface nào chỉ có một implement mà không có consumer cần | ⬜ |
| G12 | Golang | Wiring thủ công, không DI framework | Thói quen Dagger/Guice | Khởi tạo và nối dependency trong `main.go`; constructor nhận đúng thứ cần | `main.go` đọc từ trên xuống thấy toàn bộ đồ thị dependency | ⬜ |

## Definition of Done

- [ ] Tất cả API trong phạm vi hoạt động, khớp OpenAPI
- [ ] Đạt P1-NFR1 trên dữ liệu seed, có output `EXPLAIN` lưu trong ADR
- [ ] Logic tính toán (giờ đến, chọn giá) là hàm thuần, test không cần DB
- [ ] Không có package kiểu `service/`, `repository/`, `utils/`; review lại theo bảng "Go cho người từ Java" trong [project-structure](../../project-structure.md#go-cho-người-từ-java)
- [ ] Request bị huỷ → query PG bị huỷ theo (kiểm chứng bằng `pg_stat_activity`)
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [database](../../database.md), [api](../../api.md) nếu schema/API khác thiết kế
- [ ] ADR: thiết kế index tìm chuyến (kèm số liệu)
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
