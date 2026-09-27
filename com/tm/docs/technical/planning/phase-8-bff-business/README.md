# Phase 8 — BFF nghiệp vụ & admin

> Chặng B — Node.js + MongoDB (BFF) · Công nghệ: **Node.js + MongoDB**

## Mục tiêu

BFF cho ví, đặt vé, huỷ vé, admin: SSE realtime, RBAC, audit log, xuất CSV dạng stream.

**Mốc demo**: Toàn bộ API public và admin dùng được bằng `curl`; SSE sơ đồ ghế realtime

## Kiến thức trọng tâm

Node: SSE, backpressure, stream · Mongo: append-only, phân quyền, index

## Phạm vi

- **Trong**: Route ví, đặt vé, SSE, huỷ vé, admin + RBAC + audit, thống kê/CSV.
- **Ngoài**: Giao diện.

## Workstream

`bff` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P8-FR1 | FR | Sơ đồ ghế cập nhật realtime cho mọi người đang xem chuyến |
| P8-FR2 | FR | 4 vai trò: super_admin, operator, support, analyst |
| P8-FR3 | FR | Audit log cho mọi thao tác ghi của admin |
| P8-FR4 | FR | Bộ lọc thời gian, tuyến, loại phương tiện; xuất CSV |

## Task

### bff
- [ ] **P8-T01** Route `/api/wallet*`, truyền `Idempotency-Key` từ client xuống core _(trước đây P3-T12)_
- [ ] **P8-T02** Route `/api/holds`, `/api/bookings*` _(trước đây P4-T11)_
- [ ] **P8-T03** SSE `/api/trips/{id}/seats/stream`: fan-out theo chuyến, heartbeat, dọn kết nối khi client rời, backpressure `[N6]` _(trước đây P4-T12)_
- [ ] **P8-T04** Route huỷ vé, refund quote, admin refund, admin huỷ chuyến _(trước đây P8-T08)_
- [ ] **P8-T05** Collection `admin_accounts`; RBAC plugin: khai báo quyền theo route, mặc định từ chối `[N4]` _(trước đây P5-T06)_
- [ ] **P8-T06** Collection `audit_logs` append-only: hook `onResponse` ghi before/after; user DB chỉ có quyền insert `[M4]` _(trước đây P5-T07)_
- [ ] **P8-T07** Route `/api/admin/*` _(trước đây P5-T08)_
- [ ] **P8-T08** Route `/api/admin/stats/*`, xuất CSV dạng stream _(trước đây P6-T10)_

### qa
- [ ] **P8-T09** Test ma trận quyền: mỗi vai trò × mỗi route `[N4]` _(trước đây P5-T14)_

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| N6 | Node.js | Streaming realtime | Sơ đồ ghế cập nhật | SSE, backpressure, dọn kết nối khi client rời | 10.000 kết nối SSE đồng thời ổn định bộ nhớ | ⬜ |
| N4 | Node.js | Phân quyền admin | 4 vai trò | RBAC middleware, kiểm tra quyền ở server, audit log | Test: mỗi vai trò chỉ gọi được endpoint của mình | ⬜ |
| M4 | MongoDB | Audit log bất biến, tăng nhanh | Mọi thao tác admin | Append-only, phân quyền chỉ insert, archive theo thời gian | Không có đường nào sửa/xoá audit log | ⬜ |

## Definition of Done

- [ ] Mọi route public và admin có trong OpenAPI BFF, gọi được
- [ ] Ma trận quyền 4 vai trò pass 100%
- [ ] 10.000 kết nối SSE ổn định bộ nhớ
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [api](../../api.md)
- [ ] ADR: mô hình RBAC, cơ chế SSE
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
