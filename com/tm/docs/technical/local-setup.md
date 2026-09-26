# Chạy local

> Đang phát triển — hướng dẫn sẽ cập nhật theo từng phase.

## Yêu cầu

- Docker + Docker Compose
- Go 1.27.1+ (theo `go` trong `com/tm/server/go.mod`; Bazel tự tải đúng SDK này), Bazelisk (đọc phiên bản Bazel từ `.bazelversion`)
- Node.js 22+ và pnpm (bất kỳ bản nào ≥ 9.7; trong `com/tm/app` pnpm tự chuyển sang bản pin ở `packageManager` = `pnpm@11.18.0` — kiểm bằng `pnpm -v`)
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
pnpm install --frozen-lockfile   # cài đúng theo pnpm-lock.yaml (lockfile lệch → ERR_PNPM_OUTDATED_LOCKFILE)
pnpm dev                         # chạy song song script dev của mọi app (bff, web-client, web-admin)
pnpm --filter bff dev            # chỉ một app
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
bazel test //...              # như CI (target image bị SKIP trên macOS, xem dưới)

# Image OCI (macro com_tm_go_image: <name>, <name>_image, <name>_docker, <name>_push)
bazel run --config=linux-arm64 //tools/smoke:smoke_docker     # Apple Silicon → nạp com.tm.go.smoke:v1.0.0 vào Docker
docker run --rm com.tm.go.smoke:v1.0.0                        # in "snaptix smoke ok"
bazel build --config=linux-amd64 //tools/smoke:smoke_image    # server x86

# App
cd com/tm/app
pnpm lint && pnpm typecheck && pnpm test && pnpm build   # mọi package, thứ tự topo; package thiếu script được bỏ qua
pnpm --filter bff test                                   # một app
# Như CI: chỉ package thay đổi so với origin/main + package phụ thuộc vào chúng
# (không dùng `pnpm ... lint test build` — pnpm chỉ chạy lint, xem project-structure.md mục CI)
pnpm --filter "...[origin/main]" --filter '!snaptix-app' --if-present run '/^(lint|test|build)$/'

# Load test
k6 run loadtest/booking-peak.js
```
