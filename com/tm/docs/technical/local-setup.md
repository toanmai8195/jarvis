# Chạy local

> Đang phát triển — hướng dẫn sẽ cập nhật theo từng phase.

## Yêu cầu

- Docker + Docker Compose v2 (đã kiểm với Docker 29.8, Compose v5.5). macOS: nếu repo nằm trong `~/Documents`, cấp quyền cho Docker Desktop ở System Settings → Privacy & Security → Files and Folders (hạ tầng không bind mount file từ repo nên không cần, nhưng container mount mã nguồn sẽ treo nếu thiếu quyền)
- Go 1.27.1+ (theo `go` trong `com/tm/server/go.mod`; Bazel tự tải đúng SDK này), Bazelisk (đọc phiên bản Bazel từ `.bazelversion`)
- Node.js 22+ và pnpm (bất kỳ bản nào ≥ 9.7; trong `com/tm/app` pnpm tự chuyển sang bản pin ở `packageManager` = `pnpm@11.18.0` — kiểm bằng `pnpm -v`)
- Công cụ: `goose`, `sqlc`
  - `goose` pin **v3.28.0**, cài trên host (không chạy trong container, vì bind mount repo từ `~/Documents` làm Docker Desktop treo):
    ```bash
    go install github.com/pressly/goose/v3/cmd/goose@v3.28.0
    export PATH="$(go env GOPATH)/bin:$PATH"   # nếu chưa có; thêm vào ~/.zshrc
    goose -version                              # goose version: v3.28.0
    ```
    Migration là file SQL đánh số tuần tự (`00001_init.sql`, tạo mới: `goose -s -dir <thư mục> create <tên> sql`), bảng version mặc định `public.goose_db_version`.
  - `golangci-lint` pin **v2.14.0** (cùng bản CI dùng trong `.github/workflows/ci.yml`; cấu hình `com/tm/server/.golangci.yml`):
    ```bash
    brew install golangci-lint            # hoặc bản đúng pin:
    curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v2.14.0
    golangci-lint --version               # golangci-lint has version 2.14.0 ...
    cd com/tm/server && golangci-lint run ./...
    ```

## Các bước

Các lệnh `make` dùng `Makefile` ở gốc repo (GNU Make 3.81 có sẵn trên macOS là đủ; `make` không đối số chỉ in hướng dẫn). Chạy được từ thư mục khác bằng `make -C <repo> <target>`.

```bash
git clone <repo-url> snaptix && cd snaptix

# 1. Hạ tầng: postgres (core + analytics), mongodb, redis, observability — chờ tới khi mọi service healthy
#    Không cần deploy/.env (mặc định khớp bảng dưới). Chi tiết: deploy/README.md
make up
#    = docker compose -f deploy/docker-compose.yml up -d --wait   (idempotent: chạy lại không tạo lại container)
#    Tắt: docker compose -f deploy/docker-compose.yml down   (thêm -v để xoá cả dữ liệu)

# 2. Migration PG core + PG analytics (goose trên host, scripts/migrate.sh)
make migrate
#    = goose -dir com/tm/server/db/core/migrations postgres "$CORE_DATABASE_URL" up
#      goose -dir com/tm/server/db/analytics/migrations postgres "$ANALYTICS_DATABASE_URL" up
#    - Biến chưa đặt → mặc định theo bảng "Biến môi trường"; đặt biến để trỏ DB khác.
#    - goose tìm trong PATH, rồi `go env GOBIN`, `$(go env GOPATH)/bin` — không cần sửa PATH.
#      Không có goose → lệnh dừng và in lệnh cài: go install github.com/pressly/goose/v3/cmd/goose@v3.28.0
#    - Migrate cả hai DB kể cả khi một DB lỗi; có DB lỗi → exit 1, in tên DB (core/analytics).
#      URL chưa có `connect_timeout` được thêm `connect_timeout=10` (đổi bằng MIGRATE_CONNECT_TIMEOUT).

# 3. Server (Go) — chạy trực tiếp bằng go khi dev
cd com/tm/server
cp services/core/.env.example services/core/.env   # lần đầu; sửa nếu cần
set -a; . services/core/.env; set +a                # nạp CORE_* vào shell (core đọc env, không tự đọc file .env)
go run ./services/core/cmd/server                   # hoặc: bazel run //services/core/cmd/server:server
#    core nghe :8080 — curl localhost:8080/healthz (sống), /readyz (ping được PG core), /metrics (Prometheus).
#    Core khởi động được cả khi PG chưa lên (pool lười): /readyz trả 503 tới khi PG sẵn sàng.
#    Prometheus (job `core`) scrape host.docker.internal:8080/metrics — xem http://localhost:9090/targets.
#    Trace + metric RED gửi OTLP/HTTP tới otel-collector localhost:4318 (biến OTEL_*, bảng dưới) —
#    xem trace trong Grafana Explore (datasource tempo), dashboard RED với service = snaptix/core.
go run ./services/core/cmd/worker          # terminal khác — hiện là skeleton (chưa có job, không cần env): log JSON, Ctrl+C → exit 0
go run ./services/stats-worker/cmd/worker  # terminal khác

# 4. App (Node + React)
cd com/tm/app
pnpm install --frozen-lockfile   # cài đúng theo pnpm-lock.yaml (lockfile lệch → ERR_PNPM_OUTDATED_LOCKFILE)
pnpm dev                         # chạy song song script dev của mọi app (bff, web-client, web-admin)
pnpm --filter bff dev            # chỉ một app

# BFF (Fastify) — bff không tự đọc .env, nạp env vào shell trước (như core)
cp apps/bff/.env.example apps/bff/.env   # lần đầu; sửa nếu cần
set -a; . apps/bff/.env; set +a          # nạp MONGODB_URI, BFF_* vào shell
pnpm --filter bff dev                    # tsx watch src/server.ts — sửa file trong src/ là tự chạy lại
#    bff nghe 0.0.0.0:3000 — curl localhost:3000/healthz (sống), localhost:3000/readyz (ping MongoDB, hạn 2 s).
#    bff khởi động được khi MongoDB chưa lên (client lười): /readyz trả 503 tới khi Mongo sẵn sàng.
#    Log: pino JSON một dòng một object (time ISO, level chữ, msg, request_id), kể cả khi dev.
pnpm --filter bff build                  # tsc --noEmit + tsup → apps/bff/dist/server.js (ESM)
pnpm --filter bff start                  # node dist/server.js (cần env như trên)
```

## Biến môi trường

Sao chép `.env.example` thành `.env` ở mỗi app/service.

Hạ tầng (`deploy/docker-compose.yml`) chạy được khi chưa có `deploy/.env`: mọi biến có mặc định. Muốn đổi cổng/mật khẩu thì `cp deploy/.env.example deploy/.env` rồi sửa (biến: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_CORE_DB`, `POSTGRES_ANALYTICS_DB`, `POSTGRES_CORE_PORT`, `POSTGRES_ANALYTICS_PORT`, `MONGODB_PORT`, `REDIS_PORT`, `OTLP_GRPC_PORT`, `OTLP_HTTP_PORT`, `PROMETHEUS_PORT`, `TEMPO_PORT`, `GRAFANA_PORT`).

| Biến | Dùng bởi | Ví dụ |
|---|---|---|
| `CORE_DATABASE_URL` | core (bắt buộc; thiếu/rỗng/sai → core thoát với log ERROR nêu tên biến) | `postgres://snaptix:snaptix@localhost:5432/core?sslmode=disable` |
| `CORE_HTTP_ADDR` | core (mặc định `:8080`, dạng `host:port`) | `:8080` |
| `CORE_LOG_LEVEL` | core (mặc định `info`; `debug` \| `info` \| `warn` \| `error`) | `info` |
| `CORE_SHUTDOWN_TIMEOUT` | core (mặc định `10s`; định dạng Go duration `time.ParseDuration`, vd `10s`, `1500ms`; `0`/âm/thiếu đơn vị → lỗi cấu hình). Hạn chờ request đang chạy khi nhận SIGTERM/SIGINT, hết hạn thì đóng cưỡng bức, exit `1` | `10s` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | core (`pkg/otelx`, OTLP/HTTP; mặc định `http://localhost:4318`; phải là URL `http://`/`https://` có host, sai → core thoát với log ERROR; collector không tới được → core vẫn chạy, log WARN). Không hỗ trợ `OTEL_EXPORTER_OTLP_PROTOCOL` (luôn HTTP/protobuf) | `http://localhost:4318` |
| `OTEL_SERVICE_NAME` | core (mặc định `core`; thắng `service.name` trong `OTEL_RESOURCE_ATTRIBUTES`) | `core` |
| `OTEL_RESOURCE_ATTRIBUTES` | core (mặc định rỗng; `key=value,...` thêm/đè thuộc tính resource, vd `service.namespace` — mặc định `snaptix`) | `deployment.environment.name=local` |
| `OTEL_TRACES_SAMPLER`, `OTEL_TRACES_SAMPLER_ARG` | core (mặc định `parentbased_always_on`) | `parentbased_traceidratio`, `0.1` |
| `OTEL_SDK_DISABLED` | core (mặc định `false`; `true` → không export trace/metric, log vẫn có `trace_id` khi request có `traceparent`) | `false` |
| `OTEL_BSP_SCHEDULE_DELAY`, `OTEL_METRIC_EXPORT_INTERVAL`, `OTEL_EXPORTER_OTLP_TIMEOUT` | core (mặc định `5000`, `60000`, `10000` ms: chu kỳ gửi span, chu kỳ gửi metric, hạn mỗi lần export) | `500`, `2000`, `1000` |
| `ANALYTICS_DATABASE_URL` | stats-worker, bff | `postgres://snaptix:snaptix@localhost:5433/analytics` |
| `REDIS_URL` | core, bff | `redis://localhost:6379` |
| `MONGODB_URI` | bff (bắt buộc, URL `mongodb://` hoặc `mongodb+srv://`; thiếu/rỗng/sai → bff thoát `1` với log JSON `fatal` nêu tên biến, không in giá trị) | `mongodb://localhost:27017/snaptix` |
| `BFF_PORT` | bff (mặc định `3000`; số nguyên `1..65535`, sai → lỗi cấu hình) | `3000` |
| `BFF_HOST` | bff (mặc định `0.0.0.0`) | `0.0.0.0` |
| `BFF_LOG_LEVEL` | bff (mặc định `info`; `trace` \| `debug` \| `info` \| `warn` \| `error` \| `fatal`, sai → lỗi cấu hình) | `info` |
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

Dashboard RED: <http://localhost:3100/d/snaptix-red> (Rate / Errors 5xx / Duration p50-p95-p99 theo service, chọn service bằng biến `service`). Dashboard là code: sửa `deploy/observability/grafana/dashboards/red.json` rồi `make up` (image grafana build lại). Lưu/xoá trên UI bị chặn (`allowUiUpdates: false`, `disableDeletion: true`). Kiểm tĩnh: `bash deploy/observability/grafana/check-dashboards.sh`.

## Build & kiểm thử

```bash
# Toàn bộ test (từ gốc repo, không cần stack Docker) — scripts/test-all.sh:
make test
#   scripts: bash scripts/*_test.sh, bash scripts/check-structure.sh, bash scripts/check-compose.sh
#   server (com/tm/server): go vet ./... ; go test -race ./... ; bazel test //...
#   app (com/tm/app): pnpm install --frozen-lockfile ; pnpm lint ; pnpm typecheck ; pnpm test ; pnpm build
#   Mọi bộ đều chạy, có bước lỗi → exit 1 và in danh sách bước lỗi ở cuối.
#   Chỉ một bộ: bash scripts/test-all.sh server   (scripts | server | app)
#   Không gồm golangci-lint và gazelle (xem dưới).

# Server
cd com/tm/server
go test ./...                 # vòng dev nhanh
bazel run //:gazelle          # sau khi thêm/xoá file Go hoặc import
bazel test //...              # mọi target (CI chỉ test target bị ảnh hưởng, xem dưới; target image bị SKIP trên macOS)

# Image OCI (macro com_tm_go_image: <name>, <name>_image, <name>_docker, <name>_push)
bazel run --config=linux-arm64 //tools/smoke:smoke_docker     # Apple Silicon → nạp com.tm.go.smoke:v1.0.0 vào Docker
docker run --rm com.tm.go.smoke:v1.0.0                        # in "snaptix smoke ok"
bazel build --config=linux-amd64 //tools/smoke:smoke_image    # server x86

# Image core (attr image = "core-server" / "core-worker" trong BUILD → tag com.tm.go.core-server|core-worker:v1.0.0)
bazel run --config=linux-arm64 //services/core/cmd/server:server_docker   # → com.tm.go.core-server:v1.0.0, entrypoint /app/server
bazel run --config=linux-arm64 //services/core/cmd/worker:worker_docker   # → com.tm.go.core-worker:v1.0.0, entrypoint /app/worker
#   Kiểm kết quả thật bằng `docker image ls | grep com.tm.go.core-`, không tin riêng exit code của `bazel run`.
#   Server x86: --config=linux-amd64. Image distroless nonroot (uid 65532), không có shell.
# Chạy core trong mạng compose (sau `make up`; compose không khai báo networks → mạng snaptix_default):
docker run -d --name core --network snaptix_default -p 127.0.0.1:18080:8080 \
  -e CORE_DATABASE_URL='postgres://snaptix:snaptix@postgres-core:5432/core?sslmode=disable' \
  -e OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318 \
  com.tm.go.core-server:v1.0.0
curl localhost:18080/readyz          # 200 khi ping được postgres-core
#   Không dùng mạng compose: CORE_DATABASE_URL=...@host.docker.internal:5432/..., OTLP http://host.docker.internal:4318
# Dừng graceful: docker stop -t phải > CORE_SHUTDOWN_TIMEOUT + 1 s (hạn HTTP + tối thiểu 1 s đóng pool/flush telemetry),
#   nếu không Docker gửi SIGKILL (exit 137) giữa chừng. Mặc định CORE_SHUTDOWN_TIMEOUT=10s → dùng -t 12 trở lên:
docker stop -t 12 core && docker rm core   # log shutdown_step signal → http_stopped → pool_closed → done, exit 0
# Worker (skeleton, chưa có job): không cần env, không mở cổng; dừng bằng SIGTERM → exit 0
docker run -d --name core-worker com.tm.go.core-worker:v1.0.0
docker stop -t 12 core-worker && docker rm core-worker

# App
cd com/tm/app
pnpm lint && pnpm typecheck && pnpm test && pnpm build   # mọi package, thứ tự topo; package thiếu script được bỏ qua
pnpm --filter bff test                                   # một app
# Như CI: chỉ package thay đổi so với origin/main + package phụ thuộc vào chúng
# (không dùng `pnpm ... lint test build` — pnpm chỉ chạy lint, xem project-structure.md mục CI)
pnpm --filter "...[origin/main]" --filter '!snaptix-app' --if-present run '/^(lint|test|build)$/'

# Chạy lại job CI cục bộ (cùng script mà .github/workflows/ci.yml gọi; từ gốc repo).
# BASE = origin/main (như PR); cần `git fetch origin` để origin/main mới.
bash scripts/ci-changes.sh origin/main HEAD   # server=true|false, app=true|false
bash scripts/ci-affected.sh origin/main HEAD  # target Bazel test bị ảnh hưởng (//... = tất cả)
bash scripts/ci-server.sh origin/main         # golangci-lint + bazel test target bị ảnh hưởng
bash scripts/ci-app.sh origin/main            # pnpm install --frozen-lockfile + lint/test/build + chốt lockfile

# Load test
k6 run loadtest/booking-peak.js
```
