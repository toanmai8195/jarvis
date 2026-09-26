# Phase 6 — Thống kê

## Mục tiêu

Relay outbox tin cậy, stats-worker ghi vào PostgreSQL analytics theo star schema, dashboard thống kê gần realtime cho admin.

**Mốc demo**: đặt vé trên web client → trong vòng 5 phút dashboard doanh thu và lấp đầy thay đổi tương ứng; replay toàn bộ sự kiện không làm lệch số liệu.

## Phạm vi

- **Trong**: outbox relay, stats-worker, schema analytics (dim, fact, aggregate, materialized view), partition bảng lớn ở cả core và analytics, API thống kê, dashboard, xuất CSV.
- **Ngoài**: tối ưu tải cực lớn (phase 7).

## Workstream

`core` · `db` · `analytics` · `bff` · `admin` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P6-FR1 | FR | Mọi sự kiện trong outbox được giao cho stats-worker ít nhất một lần |
| P6-FR2 | FR | Stats-worker xử lý mỗi sự kiện đúng một lần về mặt kết quả |
| P6-FR3 | FR | Dashboard: doanh thu, số vé, tỉ lệ lấp đầy, top tuyến, nạp tiền (theo [admin guide](../../../user-guide/admin.md#5-thống-kê)) |
| P6-FR4 | FR | Bộ lọc thời gian, tuyến, loại phương tiện; xuất CSV |
| P6-FR5 | FR | Rebuild aggregate từ fact khi đổi công thức |
| P6-NFR1 | NFR | Độ trễ từ lúc đặt vé đến dashboard < 5 phút |
| P6-NFR2 | NFR | Báo cáo 1 năm < 2s trên 100 triệu dòng fact |
| P6-NFR3 | NFR | Xoá dữ liệu cũ bằng detach partition, không `DELETE` hàng loạt |

## Task

### core / db
- [ ] **P6-T01** Outbox relay trong `core worker`: đọc `FOR UPDATE SKIP LOCKED` theo batch, đánh dấu `published_at` (tái sử dụng pattern G4)
- [ ] **P6-T02** Partition theo tháng cho `ledger_entries`, `outbox_events`; job tạo partition trước và detach partition cũ `[P8]`

### analytics
- [ ] **P6-T03** Thiết kế star schema: `dim_date`, `dim_route`, `dim_trip`, `fact_bookings`, `fact_cancellations`, `fact_topups`; xác định grain `[A1]`
- [ ] **P6-T04** `processed_events` + ghi fact trong cùng transaction `[A2]`
- [ ] **P6-T05** Upsert `agg_revenue_hourly`, `agg_trip_occupancy` theo sự kiện; job rollup `agg_revenue_daily` `[A3]`
- [ ] **P6-T06** Partition fact theo tháng, BRIN index cột thời gian `[A4]`
- [ ] **P6-T07** Truy vấn báo cáo: window function, CTE, `GROUPING SETS`; `mv_top_routes_30d` refresh concurrently `[A5]`
- [ ] **P6-T08** Lệnh rebuild aggregate vào bảng mới rồi swap (versioning) `[A6]`
- [ ] **P6-T09** Seed 100 triệu dòng fact để đo hiệu năng

### bff
- [ ] **P6-T10** Route `/api/admin/stats/*`, xuất CSV dạng stream

### admin
- [ ] **P6-T11** Dashboard: KPI tile, biểu đồ thời gian, bảng top tuyến; lazy load chart, skeleton `[R6]`

### qa
- [ ] **P6-T12** Test replay toàn bộ outbox 2 lần → số liệu không đổi
- [ ] **P6-T13** Đối chiếu tổng doanh thu analytics với tổng ledger `REVENUE` ở core

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| P8 | PostgreSQL core | Bảng tăng không giới hạn | `ledger_entries`, `outbox_events` | Declarative partitioning theo tháng, detach/archive partition cũ | Xoá dữ liệu cũ không cần `DELETE` lớn | ⬜ |
| A1 | PostgreSQL analytics | Thiết kế mô hình phân tích | Dashboard doanh thu, lấp đầy | Star schema, chọn grain cho fact | Mọi chỉ số dashboard truy vấn từ ≤ 2 bảng | ⬜ |
| A2 | PostgreSQL analytics | Ghi đúng một lần | Sự kiện at-least-once | `processed_events` cùng transaction với fact | Replay toàn bộ outbox → số liệu không đổi | ⬜ |
| A3 | PostgreSQL analytics | Tổng hợp tăng dần | Không quét lại toàn bộ fact | Upsert aggregate theo giờ, rollup ngày | Cập nhật dashboard trễ < 5 phút | ⬜ |
| A4 | PostgreSQL analytics | Truy vấn thời gian trên dữ liệu lớn | Hàng trăm triệu dòng fact | Partition theo tháng, BRIN index, partition pruning | Báo cáo 1 năm < 2s | ⬜ |
| A5 | PostgreSQL analytics | SQL phân tích nâng cao | Top tuyến, tăng trưởng, cohort | Window function, CTE, `GROUPING SETS`, materialized view `CONCURRENTLY` | Các báo cáo trong [admin guide](../../../user-guide/admin.md#5-thống-kê) chạy bằng SQL thuần | ⬜ |
| A6 | PostgreSQL analytics | Rebuild khi sai logic | Sửa công thức tính chỉ số | Backfill từ fact/outbox, versioning aggregate | Rebuild aggregate không ảnh hưởng dashboard đang chạy | ⬜ |
| R6 | React | Dashboard biểu đồ | Thống kê | Thư viện chart, lazy load, skeleton | Dashboard hiển thị < 2s | ⬜ |

## Definition of Done

- [ ] Đặt vé → dashboard cập nhật < 5 phút
- [ ] Replay outbox không làm lệch số liệu
- [ ] Doanh thu analytics khớp ledger core
- [ ] Đạt P6-NFR2 trên dữ liệu seed
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [database](../../database.md) phần analytics, [services](../../services.md) phần sự kiện
- [ ] ADR: grain của fact, chiến lược partition, polling outbox vs logical replication/CDC
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
