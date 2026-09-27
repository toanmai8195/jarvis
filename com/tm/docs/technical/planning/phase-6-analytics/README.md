# Phase 6 — Thống kê (Go + PG analytics)

> Chặng A — Go + PostgreSQL (core) · Công nghệ: **Go + PostgreSQL**

## Mục tiêu

Outbox relay, stats-worker, star schema, aggregate tăng dần, partition, SQL phân tích.

**Mốc demo**: Đặt vé → aggregate trong PG analytics cập nhật < 5 phút; replay không lệch số

## Kiến thức trọng tâm

Go: worker at-least-once, idempotent consumer · PG: star schema, partition, BRIN, window function, materialized view

## Phạm vi

- **Trong**: Relay, stats-worker, schema analytics, truy vấn báo cáo.
- **Ngoài**: API báo cáo ở BFF, dashboard admin.

## Workstream

`core / db` · `analytics` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P6-FR1 | FR | Mọi sự kiện trong outbox được giao cho stats-worker ít nhất một lần |
| P6-FR2 | FR | Stats-worker xử lý mỗi sự kiện đúng một lần về mặt kết quả |
| P6-FR3 | FR | Rebuild aggregate từ fact khi đổi công thức |
| P6-NFR1 | NFR | Độ trễ từ lúc đặt vé đến dashboard < 5 phút |
| P6-NFR2 | NFR | Báo cáo 1 năm < 2s trên 100 triệu dòng fact |
| P6-NFR3 | NFR | Xoá dữ liệu cũ bằng detach partition, không `DELETE` hàng loạt |
| P6-NFR4 | NFR | Log có `trace_id`; trace request core hiển thị trên Grafana kèm span PG |

## Task

### infra / db (chuyển từ Phase 0)
- [ ] **P6-T01** Thêm PostgreSQL analytics (5433) vào compose; `scripts/migrate.sh analytics`, migration đầu tiên

### core / db
- [ ] **P6-T02** Outbox relay trong `core worker`: đọc `FOR UPDATE SKIP LOCKED` theo batch, đánh dấu `published_at` (tái sử dụng pattern G4) _(trước đây P6-T01)_
- [ ] **P6-T03** Partition theo tháng cho `ledger_entries`, `outbox_events`; job tạo partition trước và detach partition cũ `[P8]` _(trước đây P6-T02)_

### analytics
- [ ] **P6-T04** Thiết kế star schema: `dim_date`, `dim_route`, `dim_trip`, `fact_bookings`, `fact_cancellations`, `fact_topups`; xác định grain `[A1]` _(trước đây P6-T03)_
- [ ] **P6-T05** `processed_events` + ghi fact trong cùng transaction `[A2]` _(trước đây P6-T04)_
- [ ] **P6-T06** Upsert `agg_revenue_hourly`, `agg_trip_occupancy` theo sự kiện; job rollup `agg_revenue_daily` `[A3]` _(trước đây P6-T05)_
- [ ] **P6-T07** Partition fact theo tháng, BRIN index cột thời gian `[A4]` _(trước đây P6-T06)_
- [ ] **P6-T08** Truy vấn báo cáo: window function, CTE, `GROUPING SETS`; `mv_top_routes_30d` refresh concurrently `[A5]` _(trước đây P6-T07)_
- [ ] **P6-T09** Lệnh rebuild aggregate vào bảng mới rồi swap (versioning) `[A6]` _(trước đây P6-T08)_
- [ ] **P6-T10** Seed 100 triệu dòng fact để đo hiệu năng _(trước đây P6-T09)_

### qa
- [ ] **P6-T11** Test replay toàn bộ outbox 2 lần → số liệu không đổi _(trước đây P6-T12)_
- [ ] **P6-T12** Đối chiếu tổng doanh thu analytics với tổng ledger `REVENUE` ở core _(trước đây P6-T13)_

### observability cho core (chuyển từ Phase 0)
- [ ] **P6-T13** Thêm otel-collector, Tempo, Prometheus, Grafana vào compose (profile `observability`), datasource provision sẵn
- [ ] **P6-T14** OpenTelemetry SDK trong `pkg/otelx` (OTLP HTTP), otelhttp middleware, trace truy vấn PG khi có span cha; log có `trace_id`
- [ ] **P6-T15** Dashboard Grafana RED provision từ file

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

## Definition of Done

- [ ] Replay outbox không làm lệch số liệu
- [ ] Doanh thu analytics khớp ledger core
- [ ] Báo cáo 1 năm < 2s trên dữ liệu seed
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] ADR: grain của fact, chiến lược partition
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
