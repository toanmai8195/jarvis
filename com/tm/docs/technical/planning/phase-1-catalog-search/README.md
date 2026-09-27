# Phase 1 — Catalog & tìm chuyến

> Chặng A — Go + PostgreSQL (core) · Công nghệ: **Go + PostgreSQL**

## Mục tiêu

Làm quen Go và PostgreSQL qua bài toán đọc dữ liệu: schema quan hệ, index, sqlc, module Go theo nghiệp vụ, context, lỗi nghiệp vụ.

**Mốc demo**: `curl` API core tìm được chuyến trên 10 triệu `trip_seats`, p99 < 20ms

## Kiến thức trọng tâm

Go: package theo nghiệp vụ, interface phía dùng, context, error value, table-driven test · PG: schema quan hệ, index, EXPLAIN, sqlc

## Phạm vi

- **Trong**: Schema catalog + chuyến + giá, seed dữ liệu lớn, API đọc của core, chuẩn lỗi.
- **Ngoài**: BFF, web, CRUD admin.

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
| P1-NFR3 | NFR | Mọi dòng log của request có `request_id`; handler panic không làm sập server |

## Task

### core — HTTP nền (chuyển từ Phase 0)
- [ ] **P1-T01** Middleware request ID (`X-Request-ID` hợp lệ hoặc sinh mới) + gắn vào log
- [ ] **P1-T02** Middleware recover (panic → 500 JSON, log stack) và access log (method, route, status, duration)

### db
- [ ] **P1-T03** Migration: `stations`, `routes`, `route_stops`, `seat_layouts`, `layout_seats`, `vehicles` theo [database](../../database.md) _(trước đây P1-T01)_
- [ ] **P1-T04** Migration: `fares`, `schedules`, `trips`, `trip_seats` _(trước đây P1-T02)_
- [ ] **P1-T05** Index cho tìm chuyến: `trips (route_id, departure_at)`, index tìm tuyến theo cặp trạm `[P7]` _(trước đây P1-T03)_
- [ ] **P1-T06** Seed script sinh dữ liệu lớn: 200 trạm, 500 tuyến, 1 năm lịch chạy, 10 triệu `trip_seats` _(trước đây P1-T04)_
- [ ] **P1-T07** Viết query search bằng sqlc; đọc `EXPLAIN (ANALYZE, BUFFERS)`, tối ưu đến khi đạt NFR `[P7]` _(trước đây P1-T05)_

### core
- [ ] **P1-T08** Module `internal/catalog` theo vertical slice (types, hàm thuần, service, store, http); `internal/httpx`; theo [project-structure](../../project-structure.md#module-trong-core-modular-monolith) `[G11]` _(trước đây P1-T06)_
- [ ] **P1-T09** Wiring thủ công trong `cmd/server/main.go` `[G12]` _(trước đây P1-T06a)_
- [ ] **P1-T10** Kiểm tra ranh giới module tự động (Bazel visibility hoặc test import) theo [ADR-0001](../../adr/0001-modular-monolith-core.md) `[G11]` _(trước đây P1-T06b)_
- [ ] **P1-T11** Domain error + mapping sang HTTP/`error.code` `[G7]` _(trước đây P1-T07)_
- [ ] **P1-T12** Truyền `context.Context` từ handler → use case → repository, đặt timeout per-request `[G2]` _(trước đây P1-T08)_
- [ ] **P1-T13** API `GET /internal/v1/trips/search`, `/trips/{id}`, `/trips/{id}/seats` _(trước đây P1-T09)_
- [ ] **P1-T14** Tính giá thấp nhất theo đoạn và hạng ghế từ `fares` hợp lệ tại thời điểm hiện tại _(trước đây P1-T10)_
- [ ] **P1-T15** OpenAPI `com/tm/server/api/core.openapi.yaml` cho các endpoint trên _(trước đây P1-T11)_

### qa
- [ ] **P1-T16** Unit test domain (tính giờ đến, chọn giá) _(trước đây P1-T12)_
- [ ] **P1-T17** Integration test repository với testcontainers _(trước đây P1-T13)_
- [ ] **P1-T18** Benchmark query search trên dữ liệu seed _(trước đây P1-T14)_

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| P7 | PostgreSQL core | Truy vấn tìm chuyến nhanh | 5.000 req/s | Composite index, denormalize `available_seats`, đọc `EXPLAIN (ANALYZE, BUFFERS)` | p99 query < 20ms trên 10 triệu `trip_seats` | ⬜ |
| G11 | Golang | Package theo nghiệp vụ, interface phía dùng | Thói quen Java: chia tầng, `IFoo` + `FooImpl` | Vertical slice trong module; interface nhỏ khai báo ở package dùng nó; nhận interface, trả struct | Không có package `service/`, `repository/`; không interface nào chỉ có một implement mà không có consumer cần | ⬜ |
| G12 | Golang | Wiring thủ công, không DI framework | Thói quen Dagger/Guice | Khởi tạo và nối dependency trong `main.go`; constructor nhận đúng thứ cần | `main.go` đọc từ trên xuống thấy toàn bộ đồ thị dependency | ⬜ |
| G7 | Golang | Lỗi nghiệp vụ rõ ràng | Map sang HTTP code/`error.code` | Sentinel error + `errors.Is/As`, wrap có ngữ cảnh | Mọi lỗi trả ra đúng code trong [API](../../api.md#lỗi) | ⬜ |
| G2 | Golang | Timeout & huỷ lan truyền | Client đóng tab giữa chừng, PG chậm | `context.Context` xuyên suốt handler → use case → repository; deadline cho mọi I/O | Không còn query chạy sau khi request bị huỷ | ⬜ |

## Definition of Done

- [ ] Mọi API trong phạm vi hoạt động, khớp OpenAPI; kiểm bằng `curl`
- [ ] Đạt NFR p99 < 20ms, có output `EXPLAIN` trong ADR
- [ ] Logic tính toán (giờ đến, chọn giá) là hàm thuần, test không cần DB
- [ ] Request bị huỷ → query PG bị huỷ theo
- [ ] Không có package kiểu `service/`, `repository/`, `utils/`
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [database](../../database.md), [api](../../api.md) nếu khác thiết kế
- [ ] ADR: thiết kế index tìm chuyến (kèm số liệu)
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
