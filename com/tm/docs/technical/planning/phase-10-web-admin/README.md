# Phase 10 — Web admin (React)

> Chặng C — React (web) · Công nghệ: **React + TypeScript**

## Mục tiêu

React nâng cao: form phức tạp, kéo thả, bảng dữ liệu lớn, dashboard biểu đồ.

**Mốc demo**: Operator setup tuyến → slot trên web admin; dashboard thống kê

## Kiến thức trọng tâm

React: React Hook Form + Zod, field array, kéo thả, virtualized table, chart

## Phạm vi

- **Trong**: Skeleton web-admin, mọi màn hình admin.
- **Ngoài**: —

## Workstream

`admin`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P10-FR1 | FR | Dashboard: doanh thu, số vé, tỉ lệ lấp đầy, top tuyến, nạp tiền (theo [admin guide](../../../user-guide/admin.md#5-thống-kê)) |
| P10-NFR1 | NFR | Bảng đơn hàng 100.000 dòng thao tác mượt |

## Task

### admin
- [ ] **P10-T01** Skeleton `web-admin` bằng Vite + React + TS + Tailwind + shadcn/ui
- [ ] **P10-T02** Layout admin, điều hướng theo vai trò `[R10 tái sử dụng]` _(trước đây P5-T09)_
- [ ] **P10-T03** Form tuyến (field array trạm dừng), form giá, form lịch chạy + preview bằng React Hook Form + Zod dùng chung schema với BFF `[R4]` _(trước đây P5-T10)_
- [ ] **P10-T04** Trình vẽ sơ đồ ghế kéo thả `[R4]` _(trước đây P5-T11)_
- [ ] **P10-T05** Bảng đơn hàng, người dùng: filter/sort phía server, virtualized rows `[R5]` _(trước đây P5-T12)_
- [ ] **P10-T06** Trang audit log _(trước đây P5-T13)_
- [ ] **P10-T07** Dashboard: KPI tile, biểu đồ thời gian, bảng top tuyến; lazy load chart, skeleton `[R6]` _(trước đây P6-T11)_
- [ ] **P10-T08** UI huỷ chuyến + tiến độ; UI hoàn tiền thủ công

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| R4 | React | Form phức tạp | Setup sơ đồ ghế, lịch chạy, giá | React Hook Form + Zod, field array, kéo thả | Validate đồng nhất client/server | ⬜ |
| R5 | React | Bảng dữ liệu lớn | Đơn hàng, người dùng ở admin | Server-side pagination/filter, virtualized table | Bảng 100.000 dòng cuộn mượt | ⬜ |
| R6 | React | Dashboard biểu đồ | Thống kê | Thư viện chart, lazy load, skeleton | Dashboard hiển thị < 2s | ⬜ |

## Definition of Done

- [ ] Setup đầy đủ trên web admin → slot bán được trên web client
- [ ] Bảng 100.000 dòng thao tác mượt
- [ ] Dashboard hiển thị < 2s
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [admin guide](../../../user-guide/admin.md) theo UI thực tế
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
