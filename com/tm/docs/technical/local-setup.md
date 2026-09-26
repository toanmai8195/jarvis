# Chạy local

> Đang phát triển — hướng dẫn sẽ cập nhật theo từng phase.

## Yêu cầu

- Docker + Docker Compose
- Go 1.23+, Bazelisk (đọc phiên bản Bazel từ `.bazelversion`)
- Node.js 22+ và pnpm
- Công cụ: `goose`, `sqlc`

## Các bước

```bash
git clone <repo-url> snaptix && cd snaptix

# 1. Hạ tầng: postgres (core + analytics), mongodb, redis, observability
docker compose -f deploy/docker-compose.yml up -d --wait

# 2. Migration
goose -dir com/tm/server/db/core/migrations postgres "$CORE_DATABASE_URL" up
goose -dir com/tm/server/db/analytics/migrations postgres "$ANALYTICS_DATABASE_URL" up

# 3. Server (Go) — chạy trực tiếp bằng go khi dev
cd com/tm/server
go run ./services/core/cmd/server
go run ./services/core/cmd/worker          # terminal khác
go run ./services/stats-worker/cmd/worker  # terminal khác

# 4. App (Node + React)
cd com/tm/app
pnpm install
pnpm dev   # chạy song song bff, web-client, web-admin
```

## Biến môi trường

Sao chép `.env.example` thành `.env` ở mỗi app/service.

| Biến | Dùng bởi | Ví dụ |
|---|---|---|
| `CORE_DATABASE_URL` | core | `postgres://snaptix:snaptix@localhost:5432/core` |
| `ANALYTICS_DATABASE_URL` | stats-worker, bff | `postgres://snaptix:snaptix@localhost:5433/analytics` |
| `REDIS_URL` | core, bff | `redis://localhost:6379` |
| `MONGODB_URI` | bff | `mongodb://localhost:27017/snaptix` |
| `CORE_BASE_URL` | bff | `http://localhost:8080` |
| `CORE_SERVICE_TOKEN` | bff, core | chuỗi ngẫu nhiên |
| `AUTH_GOOGLE_ID`, `AUTH_GOOGLE_SECRET` | bff | từ Google Cloud Console |
| `SESSION_SECRET` | bff | `openssl rand -base64 32` |
| `VITE_BFF_URL` | web-client, web-admin | `http://localhost:3000` |

## Cổng mặc định

| Service | Cổng |
|---|---|
| bff | 3000 |
| web-client (Vite) | 5173 |
| web-admin (Vite) | 5174 |
| core | 8080 |
| PostgreSQL core / analytics | 5432 / 5433 |
| MongoDB | 27017 |
| Redis | 6379 |
| OTLP gRPC / HTTP (otel-collector) | 4317 / 4318 |
| Prometheus | 9090 |
| Tempo | 3200 |
| Grafana (đăng nhập ẩn danh, quyền Admin) | 3100 |

## Build & kiểm thử

```bash
# Server
cd com/tm/server
go test ./...                 # vòng dev nhanh
bazel run //:gazelle          # sau khi thêm/xoá file Go hoặc import
bazel test //...              # như CI
bazel build //services/core/cmd/server:image   # image OCI

# App
cd com/tm/app
pnpm test
pnpm build

# Load test
k6 run loadtest/booking-peak.js
```
