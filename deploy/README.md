# deploy

Hạ tầng local dùng chung cho `com/tm/server` và `com/tm/app` (P0-T02). Xem thêm [project-structure](../com/tm/docs/technical/project-structure.md) và [local-setup](../com/tm/docs/technical/local-setup.md).

```bash
# từ gốc repo
docker compose -f deploy/docker-compose.yml up -d --wait   # trả về khi mọi service healthy
docker compose -f deploy/docker-compose.yml ps
docker compose -f deploy/docker-compose.yml down           # giữ dữ liệu (volume)
docker compose -f deploy/docker-compose.yml down -v        # xoá cả dữ liệu
```

## Service

| Service | Image | Cổng host | Healthcheck |
|---|---|---|---|
| `postgres-core` | `postgres:18.6-alpine3.23` | 5432 | `pg_isready -h 127.0.0.1` |
| `postgres-analytics` | `postgres:18.6-alpine3.23` | 5433 | `pg_isready -h 127.0.0.1` |
| `mongodb` | `mongo:8.0.32` | 27017 | `mongosh ... ping` |
| `redis` | `redis:8.8.3-alpine` | 6379 | `redis-cli ping` |
| `otel-collector` | build `observability/otel-collector` (contrib 0.161.0 trên `alpine:3.24.2`) | 4317 (gRPC), 4318 (HTTP) | `wget` extension `health_check` :13133 |
| `prometheus` | build `observability/prometheus` (`prom/prometheus:v3.15.0`) | 9090 | `wget /-/ready` |
| `tempo` | build `observability/tempo` (`grafana/tempo:3.0.3`) | 3200 | `/tempo -health` (gọi `/ready`) |
| `grafana` | build `observability/grafana` (`grafana/grafana:13.2.2`) | 3100 | `wget /api/health` |

- Tên project compose: `snaptix` (network `snaptix_default`, volume `snaptix_*`).
- PostgreSQL: user/password `snaptix`/`snaptix`, DB `core` và `analytics` (hai instance riêng). MongoDB không bật auth, DB `snaptix`. Redis không mật khẩu.
- Grafana: đăng nhập ẩn danh quyền Admin; datasource Prometheus (uid `prometheus`) và Tempo (uid `tempo`) được provision sẵn.
- Luồng observability: app → OTLP 4317/4318 → `otel-collector` → trace sang Tempo (OTLP gRPC), metric sang Prometheus (OTLP HTTP vào `/api/v1/otlp`, cờ `--web.enable-otlp-receiver`). Prometheus scrape chính nó và self-telemetry của collector (:8888).

## Cấu hình observability

`observability/<service>/` chứa cấu hình và một `Dockerfile` `FROM` image gốc đã pin rồi `COPY` cấu hình vào. Compose build các image này local (`pull_policy: build`, tag `snaptix/<service>:<phiên bản gốc>`), không bind mount file từ repo:

- Docker Desktop trên macOS treo khi bind mount thư mục trong `~/Documents` mà chưa được cấp quyền truy cập (TCC). Build context được CLI đọc phía host nên không bị ảnh hưởng.
- Sửa cấu hình rồi chạy lại `up -d --wait`: compose build lại (có cache) và tạo lại container có image mới.
- `docker compose pull` bỏ qua các image build local.

## Biến môi trường

Compose chạy được khi **chưa có** `deploy/.env`: mọi biến có mặc định bằng giá trị trong [`.env.example`](.env.example) (commit). Muốn đổi (vd cổng bị trùng) thì `cp deploy/.env.example deploy/.env` rồi sửa; `.env` bị `.gitignore`.

## Kiểm tra tĩnh

`bash scripts/check-compose.sh` kiểm image pin phiên bản (kể cả `FROM` trong Dockerfile) và mọi service có healthcheck không phải dạng giả. Test: `bash scripts/check-compose_test.sh`.
