# Kiến trúc

## Tổng quan

```mermaid
flowchart LR
    subgraph Web["Web (React + Vite + TS)"]
        C[web-client]
        A[web-admin]
    end

    subgraph BFF["BackToFront (Node.js + Fastify + TS)"]
        B[bff]
        M[(MongoDB)]
    end

    subgraph Core["Core Server (Go)"]
        S[core]
        P[(PostgreSQL core)]
        R[(Redis)]
    end

    subgraph Stats["Thống kê"]
        W[stats-worker]
        SP[(PostgreSQL analytics)]
    end

    G[Google OAuth]
    PG[Cổng thanh toán]

    C --> B
    A --> B
    B --> G
    B --> M
    B -->|HTTP/JSON nội bộ| S
    S --> P
    S --> R
    S <-->|webhook| PG
    P -->|outbox| W
    W --> SP
    B -->|truy vấn báo cáo| SP
```

## Các tầng

| Tầng | Thành phần | Vai trò |
|---|---|---|
| **Web** | `web-client`, `web-admin` | Giao diện SPA (build tĩnh, phục vụ qua CDN). Chỉ gọi BFF. |
| **BackToFront** | `bff` | Xác thực Google, quản lý session, phân quyền admin, gom/định dạng dữ liệu cho UI, rate limit, cache. Lưu dữ liệu mềm (profile, session, preference, audit log) trong MongoDB. |
| **Core** | `core` | **Nguồn sự thật** cho tồn kho ghế, đơn hàng, vé, ví. Mọi logic ảnh hưởng tính đúng đắn nằm ở đây. |
| **Thống kê** | `stats-worker` | Tiêu thụ sự kiện từ outbox của core, ghi vào PostgreSQL analytics dạng fact/aggregate. |

## Nguyên tắc thiết kế

1. **Core là modular monolith** — một service, chia module theo nghiệp vụ với ranh giới chặt. Xem [ADR-0001](adr/0001-modular-monolith-core.md).
2. **Một nguồn sự thật cho tiền và ghế** — chỉ `core` được ghi vào dữ liệu ghế, đơn hàng, ledger. BFF không bao giờ tự tính tiền.
3. **Web không gọi thẳng core** — core không public ra internet; BFF là cổng duy nhất.
4. **Tách OLTP / OLAP** — báo cáo đọc từ analytics DB, không đọc từ core DB.
5. **Idempotent mọi nơi** — mọi lệnh ghi có tác động tiền/ghế đều mang idempotency key.
6. **Sự kiện qua outbox** — ghi sự kiện cùng transaction với dữ liệu nghiệp vụ, không publish trực tiếp.

## Công nghệ

| Thành phần | Stack |
|---|---|
| Web | React, Vite, TypeScript, React Router, shadcn/ui (Tailwind CSS), TanStack Query, React Hook Form + Zod |
| BackToFront | Node.js, Fastify, TypeScript, MongoDB, @fastify/oauth2 (Google), undici, Zod |
| Core | Go, PostgreSQL, pgx, sqlc, goose, chi |
| Thống kê | Go (worker), PostgreSQL (partitioning, materialized view) |
| Cache / lock | Redis |
| Build | Bazel + Gazelle (Go, `com/tm/server`), pnpm workspace (TS, `com/tm/app`) |
| Hạ tầng | Docker Compose, k6, OpenTelemetry, Prometheus, Grafana |

## Yêu cầu phi chức năng

| Chỉ số | Mục tiêu production | Nghiệm thu máy dev (phase 7) |
|---|---|---|
| Thông lượng tìm chuyến | ≥ 5.000 req/s | ≥ 2.000 req/s |
| Thông lượng đặt vé (cao điểm) | ≥ 1.000 booking/s | ≥ 500 booking/s |
| Latency p99 đặt vé | < 300 ms | < 300 ms |
| Latency p99 tìm chuyến | < 150 ms | < 150 ms |
| Chạy dài (bộ nhớ, bloat ổn định) | 24h | 1h |
| Bán trùng ghế | **0** | **0** |
| Sai lệch tiền | **0 đồng** | **0 đồng** |
| Độ trễ dữ liệu thống kê | < 5 phút | < 5 phút |
| Availability | 99,9% | — |

Máy dev: Docker 8 GB RAM / 8 CPU, k6 chạy cùng máy. Cột "Nghiệm thu máy dev" là điều kiện đóng phase 7; cột production là mục tiêu thiết kế.

## Observability

- **Tracing**: OpenTelemetry, trace ID truyền từ BFF sang core qua header `traceparent`.
- **Metrics**: Prometheus — RED metrics cho mỗi endpoint, số ghế đang hold, độ trễ outbox, pool connection PG.
- **Logging**: JSON có cấu trúc, luôn kèm `trace_id`, `user_id`, `request_id`. Core: middleware trong `services/core/internal/httpx` nhận/sinh `X-Request-ID`, handler slog lấy `request_id`/`trace_id`/`span_id` từ context, mỗi request một dòng access log (`method`, `route`, `path`, `status`, `latency_ms`, `bytes`).
- **Dashboard**: Grafana (datasource Prometheus + Tempo provision sẵn). Dashboard **RED theo service** (uid `snaptix-red`, thư mục `snaptix`) provisioning từ `deploy/observability/grafana/dashboards/red.json`: biến `service` = label `job` (`<service.namespace>/<service.name>`), Rate (theo service và theo `http_route`), Errors (tỉ lệ 5xx), Duration (p50/p95/p99) trên metric OTel `http.server.request.duration` (Prometheus: `http_server_request_duration_seconds_*`).
- **Luồng local**: app → OTLP (4317/4318) → otel-collector → Tempo (trace) / Prometheus (metric) → Grafana. Cấu hình trong `deploy/observability/`.
  Metric vào Prometheus bằng OTLP push: collector đẩy tới `/api/v1/otlp` của Prometheus (cờ `--web.enable-otlp-receiver`), không qua exporter để scrape. Prometheus chỉ scrape chính nó, self-telemetry của collector (:8888) và `/metrics` của service (từ P0-T07). Metric RED (`http.server.request.duration`) của service Go **chỉ** đi qua OTLP (label `job` = `<service.namespace>/<service.name>`); `/metrics` scrape (job `core`) chỉ có metric runtime Go/process, không có histogram request, để dashboard RED không thấy một service hai lần (quyết định P0-T08).
