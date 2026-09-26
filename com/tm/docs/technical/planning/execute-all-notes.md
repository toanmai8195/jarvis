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
