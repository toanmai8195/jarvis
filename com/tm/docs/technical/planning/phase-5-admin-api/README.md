# Phase 5 — Admin API (core)

> Chặng A — Go + PostgreSQL (core) · Công nghệ: **Go + PostgreSQL**

## Mục tiêu

API quản trị ở core: CRUD catalog, giá, lịch chạy, sinh slot hàng loạt, tra cứu đơn.

**Mốc demo**: `curl` tạo tuyến → lịch chạy → slot sinh ra và tìm được

## Kiến thức trọng tâm

Go: validate, batch · PG: COPY / unnest insert hàng loạt, cursor pagination, versioning dữ liệu

## Phạm vi

- **Trong**: API `/internal/v1/admin/*` ở core.
- **Ngoài**: Phân quyền, audit log (Phase 8), giao diện admin (Phase 10).

## Workstream

`core`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P5-FR1 | FR | CRUD trạm, sơ đồ ghế (có phiên bản), phương tiện, tuyến + trạm dừng |
| P5-FR2 | FR | Cấu hình giá theo đoạn, hạng ghế, hiệu lực thời gian; không ghi đè giá cũ |
| P5-FR3 | FR | Tạo lịch chạy lặp, xem trước slot, xác nhận sinh slot |
| P5-FR4 | FR | Tạm dừng/mở bán slot, đổi phương tiện (kiểm tra tương thích), khoá ghế |
| P5-FR5 | FR | Tra cứu đơn hàng, người dùng; khoá/mở khoá tài khoản |
| P5-NFR1 | NFR | Sinh 1 năm slot cho một lịch hằng ngày < 5s |

## Task

### core
- [ ] **P5-T01** API `/internal/v1/admin/*` cho catalog, fare, schedule, trip _(trước đây P5-T01)_
- [ ] **P5-T02** Sơ đồ ghế dùng cho chuyến đã bán → chỉ tạo phiên bản mới _(trước đây P5-T02)_
- [ ] **P5-T03** Sinh slot từ `schedules.recurrence`: batch insert `trips` + `trip_seats` (COPY / `unnest`), idempotent theo (schedule, ngày) _(trước đây P5-T03)_
- [ ] **P5-T04** Đổi phương tiện: kiểm tra mọi ghế đã bán tồn tại ở sơ đồ mới _(trước đây P5-T04)_
- [ ] **P5-T05** API tìm kiếm đơn hàng có filter + cursor pagination _(trước đây P5-T05)_

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|

## Definition of Done

- [ ] Setup tuyến → lịch chạy → slot bằng API, slot tìm được qua API tìm chuyến
- [ ] Sinh 1 năm slot < 5s
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [api](../../api.md) phần admin
- [ ] ADR: cách sinh slot hàng loạt
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
