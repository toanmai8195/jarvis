# Phase 5 — Admin

## Mục tiêu

Web admin cho nhà vận hành: setup catalog, lịch chạy/slot, giá vé; quản lý đơn và người dùng; phân quyền theo vai trò và audit log bất biến.

**Mốc demo**: operator tạo trạm → sơ đồ ghế → phương tiện → tuyến → giá → lịch chạy, slot sinh ra và bán được trên web client; support tra cứu đơn; mọi thao tác có trong audit log.

## Phạm vi

- **Trong**: API admin CRUD ở core, RBAC ở BFF, audit log MongoDB, sinh slot từ lịch, web-admin.
- **Ngoài**: huỷ chuyến + hoàn tiền hàng loạt và hoàn tiền thủ công (phase 8), dashboard thống kê (phase 6).

## Workstream

`core` · `bff` · `admin` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P5-FR1 | FR | CRUD trạm, sơ đồ ghế (có phiên bản), phương tiện, tuyến + trạm dừng |
| P5-FR2 | FR | Cấu hình giá theo đoạn, hạng ghế, hiệu lực thời gian; không ghi đè giá cũ |
| P5-FR3 | FR | Tạo lịch chạy lặp, xem trước slot, xác nhận sinh slot |
| P5-FR4 | FR | Tạm dừng/mở bán slot, đổi phương tiện (kiểm tra tương thích), khoá ghế |
| P5-FR5 | FR | Tra cứu đơn hàng, người dùng; khoá/mở khoá tài khoản |
| P5-FR6 | FR | 4 vai trò: super_admin, operator, support, analyst |
| P5-FR7 | FR | Audit log cho mọi thao tác ghi của admin |
| P5-NFR1 | NFR | Sinh 1 năm slot cho một lịch hằng ngày < 5s |
| P5-NFR2 | NFR | Bảng đơn hàng 100.000 dòng thao tác mượt |

## Task

### core
- [ ] **P5-T01** API `/internal/v1/admin/*` cho catalog, fare, schedule, trip
- [ ] **P5-T02** Sơ đồ ghế dùng cho chuyến đã bán → chỉ tạo phiên bản mới
- [ ] **P5-T03** Sinh slot từ `schedules.recurrence`: batch insert `trips` + `trip_seats` (COPY / `unnest`), idempotent theo (schedule, ngày)
- [ ] **P5-T04** Đổi phương tiện: kiểm tra mọi ghế đã bán tồn tại ở sơ đồ mới
- [ ] **P5-T05** API tìm kiếm đơn hàng có filter + cursor pagination

### bff
- [ ] **P5-T06** Collection `admin_accounts`; RBAC plugin: khai báo quyền theo route, mặc định từ chối `[N4]`
- [ ] **P5-T07** Collection `audit_logs` append-only: hook `onResponse` ghi before/after; user DB chỉ có quyền insert `[M4]`
- [ ] **P5-T08** Route `/api/admin/*`

### admin
- [ ] **P5-T09** Layout admin, điều hướng theo vai trò `[R10 tái sử dụng]`
- [ ] **P5-T10** Form tuyến (field array trạm dừng), form giá, form lịch chạy + preview bằng React Hook Form + Zod dùng chung schema với BFF `[R4]`
- [ ] **P5-T11** Trình vẽ sơ đồ ghế kéo thả `[R4]`
- [ ] **P5-T12** Bảng đơn hàng, người dùng: filter/sort phía server, virtualized rows `[R5]`
- [ ] **P5-T13** Trang audit log

### qa
- [ ] **P5-T14** Test ma trận quyền: mỗi vai trò × mỗi route `[N4]`
- [ ] **P5-T15** E2E setup tuyến → slot → đặt vé trên web client

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| N4 | Node.js | Phân quyền admin | 4 vai trò | RBAC middleware, kiểm tra quyền ở server, audit log | Test: mỗi vai trò chỉ gọi được endpoint của mình | ⬜ |
| M4 | MongoDB | Audit log bất biến, tăng nhanh | Mọi thao tác admin | Append-only, phân quyền chỉ insert, archive theo thời gian | Không có đường nào sửa/xoá audit log | ⬜ |
| R4 | React | Form phức tạp | Setup sơ đồ ghế, lịch chạy, giá | React Hook Form + Zod, field array, kéo thả | Validate đồng nhất client/server | ⬜ |
| R5 | React | Bảng dữ liệu lớn | Đơn hàng, người dùng ở admin | Server-side pagination/filter, virtualized table | Bảng 100.000 dòng cuộn mượt | ⬜ |

## Definition of Done

- [ ] Demo luồng setup đầy đủ → slot bán được trên web client
- [ ] Ma trận quyền pass 100%
- [ ] Không có API nào sửa/xoá audit log; user DB không có quyền update/delete
- [ ] Đạt P5-NFR1, P5-NFR2
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [admin guide](../../../user-guide/admin.md) theo UI thực tế
- [ ] Cập nhật [api](../../api.md) phần admin
- [ ] ADR: mô hình RBAC, cách sinh slot hàng loạt
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
