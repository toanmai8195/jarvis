# Ghi chú execute-all

Việc ngoài phạm vi phát hiện trong lúc chạy execute-all, để báo cáo cuối.

## P0-T01

- Gợi ý cho P0-T04 (CI) / P0-T05 (Makefile): gọi `bash scripts/check-structure_test.sh` và `bash scripts/check-structure.sh` để luật cấu trúc được kiểm tự động. Không làm trong P0-T01.
- TC08 của P0-T01 (không có `MODULE.bazel`, `go.mod`, `package.json`, `docker-compose.yml`, `Makefile`, `.github/workflows`, `com/tm/server/db`) chỉ đúng tại commit của P0-T01; không chạy lại khi đóng phase.

## P0-T01a

- P0-T04 (CI): cần Bazelisk (đọc `com/tm/server/.bazelversion` = 8.7.0) và chạy trong `com/tm/server`. `MODULE.bazel.lock` được sinh trên darwin/arm64; nếu CI linux báo lockfile lệch, cân nhắc `--lockfile_mode=error` chỉ sau khi xác nhận lockfile ổn định giữa hai OS (góp ý review TC18).
- P0-T10a (image core): target do Gazelle đặt theo tên thư mục → `services/core/cmd/server` sinh `//services/core/cmd/server:server_docker` với tag `com.tm.go.server:v1.0.0`, `cmd/worker` → `com.tm.go.worker`. Có thể cần đặt tên image rõ hơn (vd `# keep` + `name` hoặc thêm tham số image name vào macro) — quyết định ở P0-T10a.
- TC22: `grep -rn 'cmd/server:image' com/tm/docs/technical` chỉ còn khớp trong chính `tasks/P0-T01a/test-cases.md` (dòng định nghĩa TC22 và mục Review trích lại chuỗi) — không sửa được vì là test case đã duyệt; mọi docs khác đã sạch.
- `go build ./...` ghi binary `com/tm/server/smoke` khi module chỉ có một package main → đã thêm `/com/tm/server/smoke` vào `.gitignore`.
- `.github/workflows` và `Makefile` chưa có; khi P0-T04/P0-T05 thêm, nên gọi `bazel test //...` và `go test -race ./...` trong `com/tm/server`.

## P0-T01b

- **Đề xuất người dùng tự sửa** (TC14 đã chứng minh): lệnh `pnpm --filter "...[origin/main]" lint test build` ở `CLAUDE.md` (Bước 4 — App, và bảng thư mục dòng `com/tm/app`: `pnpm --filter <app> lint test build`) và dòng task **P0-T04** trong `planning/phase-0-foundation/README.md` chỉ chạy `lint` (pnpm coi `test build` là đối số của `lint`, exit 0 nên che lỗi). Lệnh đúng: `pnpm --filter "...[origin/main]" --filter '!snaptix-app' --if-present run '/^(lint|test|build)$/'` (một app: `pnpm --filter <app> run '/^(lint|test|build)$/'`). `project-structure.md` và `local-setup.md` đã sửa trong P0-T01b.
- P0-T04 (CI): chạy `pnpm install --frozen-lockfile` **trước** mọi script. pnpm 11 tự `install` (không frozen) trước khi `run` nếu `package.json` lệch deps, nên lockfile có thể bị viết lại âm thầm. Nên thêm `git diff --exit-code com/tm/app/pnpm-lock.yaml` sau bước chạy script.
- P0-T04: `...[origin/main]` cần `fetch-depth: 0` (hoặc fetch `origin/main`) trong `actions/checkout`, nếu không pnpm không so được thay đổi. Filter không khớp package nào → exit 0 (không dùng `--fail-if-no-match`).
- Khi chỉ `package.json` gốc / `pnpm-workspace.yaml` / lockfile đổi, `--filter '!snaptix-app'` làm CI không chạy package nào. P0-T04 cân nhắc chạy toàn bộ (`pnpm lint && pnpm test && pnpm build`) trong trường hợp này.

## P0-T02

- **Cần người dùng (không chặn)**: Docker Desktop trên máy dev treo khi bind mount thư mục trong `~/Documents` (repo nằm ở `~/Documents/code/jarvis`), macOS chưa cấp quyền Files and Folders → Documents (TCC). Treo xong thì mọi `docker run` sau đó cũng treo cho tới khi `docker desktop restart`. P0-T02 né bằng cách COPY cấu hình vào image build local (không bind mount). Task sau nếu mount mã nguồn/migration vào container (P0-T03 goose trong container, P0-T05 `make migrate`...) sẽ gặp lại lỗi này. Nên cấp quyền, hoặc chạy goose trên host.
- MongoDB 8.x không chạy trên kernel Docker Desktop hiện tại (7.0.12, ≥ 6.19, SERVER-121912). Compose đặt `GLIBC_TUNABLES=glibc.pthread.rseq=1` cho `mongodb`. Khi nâng image mongo, thử bỏ biến này.
- P0-T04 (CI) / P0-T05 (Makefile): gọi `bash scripts/check-compose_test.sh` và `bash scripts/check-compose.sh` (cần `docker compose` + `jq`). `make up` nên là `docker compose -f deploy/docker-compose.yml up -d --wait` (tự build image observability, `pull_policy: build`).
- P0-T06 (dashboard): thêm `deploy/observability/grafana/provisioning/dashboards/` (provider + JSON). Image grafana COPY cả thư mục `provisioning`, nên chỉ cần `up -d --wait` là có. Datasource uid cố định: `prometheus`, `tempo`.
- P0-T07+: target scrape của core thêm vào `deploy/observability/prometheus/prometheus.yml` (từ container gọi host qua `host.docker.internal:8080`; trên Linux cần `extra_hosts: host.docker.internal:host-gateway` cho service `prometheus`).
- Tempo chưa bật `metrics_generator` (service graph/span metrics). Nếu phase sau cần, cấu hình `metrics_generator` + remote write vào Prometheus (`--web.enable-remote-write-receiver`).

## P0-T03

- P0-T04 (CI) / P0-T05 (`make migrate`): cài goose đúng bản pin `go install github.com/pressly/goose/v3/cmd/goose@v3.28.0` (không `@latest`), chạy trên host (không mount repo vào container). `make migrate` = hai lệnh bước 2 của `local-setup.md`, cần `$(go env GOPATH)/bin` trong `PATH`.
- Máy dev: `$(go env GOPATH)/bin` (`~/go/bin`) chưa có trong `PATH` của shell; người dùng nên thêm vào `~/.zshrc` (đã ghi trong `local-setup.md`).
- Lệnh nguyên văn của vài TC đã duyệt có nhiễu không phải lỗi: TC02 `ls $CORE_DIR $AN_DIR | grep -v ...` in dòng tiêu đề thư mục của `ls` (tên file đều khớp); TC04 `grep -rn 'goose@latest' com/tm/docs` khớp chính dòng định nghĩa TC04 trong `tasks/P0-T03/test-cases.md`; TC16 `git status --porcelain com/tm/server/db` in `?? com/tm/server/db/` vì file của task chưa commit (không có file lạ, `-uall` chỉ ra 2 migration). Không sửa được vì là test case đã duyệt.
- macOS không có `timeout` (coreutils) — test đo thời gian bằng `date +%s`.

## P0-T04

Cần kiểm khi đóng phase 0 (chạy thật trên GitHub qua nhánh `ci-check/*` + draft PR, P0-AT04/AT05/AT11 và DoD "CI xanh trên `main`"):
- **Lockfile Bazel trên Linux**: `MODULE.bazel.lock` sinh trên darwin/arm64. Trên runner `ubuntu-24.04`, Bazel có thể thêm mục cho platform Linux. Workflow **không** đặt `--lockfile_mode=error`, nên lockfile bị cập nhật trong runner cũng không fail. Nếu log báo lockfile đổi, sinh lại trên Linux (hoặc `bazel mod deps --lockfile_mode=update` trên cả hai OS) và commit.
- **Target image trên Linux**: push lên `main` có đổi file toàn cục, hoặc `before` = 0, sẽ chạy `bazel test //...`. Trên Linux, lệnh này build cả `*_image` (trên macOS thì SKIPPED), nên cần tải base distroless. Kiểm thời gian job và việc pull `gcr.io` từ runner.
- **Toolchain CI** kiểm ở bước "Phiên bản toolchain": Go theo `go.mod` (1.27.1, setup-go v7 phải có bản này), Bazel 8.7.0 qua Bazelisk 1.29.0 (setup-bazel), golangci-lint 2.14.0, pnpm 11.18.0 (`packageManager`), Node theo `engines.node` `>=22`: setup-node lấy bản mới nhất thỏa điều kiện, **có thể lớn hơn 22**. Nếu muốn cố định thì thêm `.node-version`.
- **Required checks**: sau khi CI xanh trên `main`, cân nhắc bật branch protection cho `main` với required check `changes`, `server (golangci-lint + bazel test)`, `app (pnpm lint/test/build)`. Job skipped do `if` được tính là đạt. Đây là cài đặt repo trên GitHub, cần người dùng quyết định.
- **P0-AT05**: cần package TS có test thật (P0-T11/P0-T17), hoặc fixture tạm trên nhánh `ci-check/*`.
- `pull_request` từ fork không có quyền ghi. Workflow chỉ cần `contents: read`, nên không ảnh hưởng.

Ngoài phạm vi (ghi lại, không làm):
- Dòng CI `com/tm/server/api/**`, `com/tm/app/api/**` → sinh lại code + `git diff --exit-code`: thêm khi có codegen (oapi-codegen / openapi-typescript, phase sau). Khi thêm thì cập nhật `scripts/ci-changes.sh` (cả hai job) và bảng CI trong `project-structure.md` (đang ghi "chưa có").
- Dòng CI `com/tm/docs/**` → kiểm tra link markdown: chưa có job. Có thể thêm job `docs` với `lychee` hoặc `markdown-link-check`, pin SHA.
- Gọi `scripts/check-structure*.sh`, `scripts/check-compose*.sh`, `scripts/ci-*_test.sh` trong CI (job `repo` luôn chạy, hoặc chạy khi `scripts/**` đổi): chưa làm, vì không có trong dòng task. `check-compose.sh` cần `docker compose` (runner ubuntu có sẵn).
- Cache Bazel (disk/repository cache của setup-bazel) và cache pnpm store: chưa bật. Bật khi thời gian CI thành vấn đề.
- `CLAUDE.md` bước 4 (App) vẫn ghi `pnpm --filter "...[origin/main]" lint test build`. Đề xuất sửa đã có ở P0-T01b, người dùng tự sửa.
- Quy trình: agent triển khai có một lần sửa `com/tm/server/.golangci.yml` bằng python qua Bash (bỏ dòng `run.go` lặp) thay vì Edit — vi phạm quy tắc "không lách hook bằng Bash" (hook vẫn cho phép vì bước 1 đã `[x]`). Orchestrator đã đọc lại file và chạy lại golangci-lint/bazel test: nội dung đúng.

## P0-T05

- `make test` (`scripts/test-all.sh`) không chạy golangci-lint và `bazel run //:gazelle` (A4 không bắt buộc; gazelle sửa `BUILD.bazel`). Lint vẫn ở CI (`scripts/ci-server.sh`). Nếu muốn một lệnh "như CI" trên máy dev, cân nhắc target phụ (vd `make lint`) ở task sau.
- CI chưa gọi `scripts/*_test.sh` (kể cả `migrate_test.sh`, `test-all_test.sh` mới) — cùng ghi chú "Ngoài phạm vi" của P0-T04. `scripts/ci-changes.sh` chỉ coi `scripts/ci-*` là file CI, nên đổi `scripts/migrate.sh`/`test-all.sh` không kích hoạt job nào.
- Chưa có `make down`: tắt stack vẫn dùng `docker compose -f deploy/docker-compose.yml down [-v]` (đã ghi trong `local-setup.md` và `make help`). Target chạy service (core/bff/web) để cho P0-T07+, P0-T11+.
- TC14 chạy đúng thứ tự bảng thì lần `make -C $REPO migrate` sau TC12 in `OK 00001_init.sql` cho analytics (TC12 dừng analytics trước khi migrate, TC14 không `down -v`), không phải `no migrations to run`. Đã chạy lại TC14 khi cả hai DB đã migrate: in `no migrations to run` cho cả hai. Không phải lỗi của Makefile.
- `CLAUDE.md` bước 4 (App) vẫn ghi `pnpm --filter "...[origin/main]" lint test build` (ghi chú P0-T01b). Bước 4 của task này chạy lệnh đúng `pnpm --filter "...[origin/main]" --filter '!snaptix-app' --if-present run '/^(lint|test|build)$/'`.
- `actionlint` chưa cài trên máy dev, bước 4 bỏ qua.

## P0-T06

- TC16/TC17 (UI) từng bị chặn vì Claude in Chrome chưa kết nối (`auto block`). Người dùng kết nối lại, TC01–TC21 đã chạy lại và pass trên stack mới (2026-09-26).
- Grafana 13.2.2 tải plugin datasource `prometheus`/`tempo` ngầm (background installer, cần Internet) vài giây **sau** khi `/api/health` OK, nên ngay sau `up --wait` query trả 404 `plugin.notRegistered`. Dockerfile grafana đã cài sẵn `prometheus 13.2.1`, `tempo 13.2.2` lúc build. Khi nâng Grafana phải nâng hai bản pin này. Các plugin khác (zipkin, jaeger, mssql...) vẫn được tải ngầm lúc chạy — nếu muốn chạy offline hoàn toàn, cân nhắc `GF_PLUGINS_PREINSTALL_DISABLED=true` (đổi compose, ngoài phạm vi).
- `deploy/observability/grafana/check-dashboards.sh` + `_test.sh` nằm trong thư mục grafana (TC20 chỉ cho đổi `deploy/observability/grafana/**`), nên `make test` (`scripts/test-all.sh`, chỉ quét `scripts/*_test.sh`) chưa gọi. Task sau nên thêm vào `suite_scripts` và CI.
- Script kiểm tra không chứa chuỗi `__inputs`/`DS_PROMETHEUS` vì TC01 grep cả `deploy/observability/grafana`: check dạng export bằng "khoá cấp đầu bắt đầu bằng `__`" và placeholder `${DS_`.

## P0-T07

- `local-setup.md` bước 3 vẫn có `go run ./services/core/cmd/worker` và `./services/stats-worker/cmd/worker`, hai binary này chưa có (task sau). Không xoá vì là mô tả đích.
- Có hai nguồn metric cho core: scrape `/metrics` (job `core`, từ P0-T07) và OTLP push (từ P0-T10, `job` dạng `snaptix/core`, handbook P0-T06). Biến `service` của dashboard RED sẽ thấy hai giá trị. P0-T08/P0-T10 nên chọn một nguồn cho metric RED.
- Core chưa có graceful shutdown: `defer pool.Close()` không chạy khi process bị SIGTERM. Đúng phạm vi P0-T09.
- `scripts/migrate.sh` mặc định `CORE_DATABASE_URL` không có `sslmode=disable` (pgx/goose thử TLS rồi fallback). `.env.example` của core dùng `?sslmode=disable`. Không lỗi, chỉ lệch nhỏ; có thể thống nhất khi đóng phase.
- TC18 chạy ngay sau TC17 (pause/unpause PG): `dc up -d --wait postgres-core` báo `container ... is unhealthy` một lần vì healthcheck của PG còn fail từ lúc pause. Core vẫn trả 200. Đã chạy lại TC18 khi PG healthy: pass. Không phải lỗi của core.

## P0-T08

- **Quyết định nguồn metric RED** (trả lời ghi chú P0-T07): metric RED (`http.server.request.duration`) của core chỉ đi qua OTel/OTLP (`job="snaptix/core"` từ P0-T10), khớp dashboard P0-T06. Không đăng ký histogram `prometheus/client_golang` cho request; `/metrics` (job `core`) chỉ còn metric runtime Go/process. Đã ghi vào `architecture.md` (Observability) và handbook P0-T08. Trước P0-T10, MeterProvider global là no-op nên RED của core **chưa** có trên Prometheus — P0-T10 kiểm.
- P0-T10: `main` đã truyền `otel.GetTracerProvider()`/`GetMeterProvider()`/`GetTextMapPropagator()` vào `httpx.NewRouter`; chỉ cần gọi `otel.SetTracerProvider/SetMeterProvider/SetTextMapPropagator(...)` trong `pkg/otelx` (bản global trả về là delegate, tự chuyển sang bản thật kể cả khi `Set` sau lúc dựng router; nên gọi trước cho rõ ràng). Khi đó P0-AT06 (`trace_id` trong log chạy thật) và phần core của P0-AT07 mới kiểm được.
- Middleware OTel tự viết bằng OTel API (không dùng `otelhttp`): tên span/`http.route` đặt sau khi chi định tuyến. Nếu P0-T12 cần span client HTTP cho core (gọi ra ngoài) thì cân nhắc otelhttp transport khi đó.
- Không có access log cho request bị huỷ bằng `panic(http.ErrAbortHandler)` (access log ghi sau `next`). Chấp nhận; ghi trong handbook.
- 404/405 của chi vẫn trả body text mặc định (`404 page not found`), chưa theo format lỗi JSON của api.md — thuộc task có route nghiệp vụ (map lỗi → HTTP).
- Quy trình: agent triển khai có một lần sửa `services/core/internal/httpx/otel.go` bằng python qua Bash (đổi chữ ký `telemetry` bỏ trả `error`, thêm import `log/slog`) thay vì Edit — vi phạm quy tắc. Nội dung đã được kiểm lại bằng gofmt/vet/test/lint/bazel ở bước 4.
