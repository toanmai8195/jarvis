# Handbook

Ghi chú kỹ thuật **theo task**: kỹ thuật đã áp dụng, lý do, bẫy gặp phải, kèm link tới đoạn code thật. Agent viết sau khi task qua bước 5 (xem [`CLAUDE.md`](../../../../../CLAUDE.md#handbook)).

| | Handbook | Lessons learned |
|---|---|---|
| Cấp độ | Task | Phase |
| Người viết | Agent | Người dùng |
| Nội dung | Kỹ thuật cụ thể + link code | Nhìn lại quá trình, số liệu, điều làm khác |

## Cấu trúc

Mỗi task một file, nhóm theo phase:

```
handbook/
├── README.md              # file này: chỉ mục theo task và theo chủ đề
├── phase-0/
│   └── P0-T01.md
├── phase-1/
│   └── ...
└── phase-9/
```

## Theo task

| Task | Tên | File |
|---|---|---|
| P0-T01 | Khởi tạo cấu trúc `com/tm/{server,app,docs}`, `deploy/`, `loadtest/` | [phase-0/P0-T01.md](phase-0/P0-T01.md) |
| P0-T01a | `com/tm/server` Bazel workspace (bzlmod, Gazelle, một `go.mod`, macro `com_tm_go_image`) | [phase-0/P0-T01a.md](phase-0/P0-T01a.md) |
| P0-T01b | `com/tm/app` pnpm workspace (`pnpm-workspace.yaml`, `package.json` gốc, script chạy theo filter) | [phase-0/P0-T01b.md](phase-0/P0-T01b.md) |
| P0-T02 | `deploy/docker-compose.yml`: PG core/analytics, MongoDB, Redis, otel-collector, Prometheus, Tempo, Grafana | [phase-0/P0-T02.md](phase-0/P0-T02.md) |
| P0-T03 | goose, `db/core/migrations`, `db/analytics/migrations`, migration rỗng đầu tiên | [phase-0/P0-T03.md](phase-0/P0-T03.md) |
| P0-T04 | GitHub Actions theo đường dẫn thay đổi: golangci-lint + `bazel test` target bị ảnh hưởng, pnpm lint/test/build package bị ảnh hưởng | [phase-0/P0-T04.md](phase-0/P0-T04.md) |
| P0-T05 | Makefile / script: `make up`, `make migrate`, `make test` | [phase-0/P0-T05.md](phase-0/P0-T05.md) |
| P0-T06 | Dashboard Grafana cơ bản: RED metrics cho mỗi service | [phase-0/P0-T06.md](phase-0/P0-T06.md) |
| P0-T07 | Skeleton `services/core`: config env, slog JSON, chi, pgxpool, `/healthz`, `/readyz`, `/metrics` | [phase-0/P0-T07.md](phase-0/P0-T07.md) |

## Chỉ mục theo chủ đề

Tag gợi ý: `go/channel` · `go/errgroup` · `go/context` · `go/generics` · `go/pprof` · `go/testing` · `pg/lock` · `pg/isolation` · `pg/index` · `pg/mvcc` · `pg/partition` · `node/event-loop` · `node/stream` · `node/fastify` · `mongo/index` · `react/state` · `react/memo` · `react/query` · `bazel` · ...

| Chủ đề | Task | Bài học | Link |
|---|---|---|---|
| `git` | P0-T01 | Git không track thư mục rỗng: `.gitkeep` hoặc README | [P0-T01.md](phase-0/P0-T01.md#git-không-track-thư-mục-rỗng-gitkeep-hoặc-readme) |
| `bash`, `testing` | P0-T01 | Script kiểm tra cấu trúc nhận ROOT làm tham số | [P0-T01.md](phase-0/P0-T01.md#script-kiểm-tra-cấu-trúc-nhận-root-làm-tham-số) |
| `bazel`, `bazel/bzlmod`, `go/modules` | P0-T01a | bzlmod: một `go.mod` làm nguồn sự thật cho cả `go` và Bazel | [P0-T01a.md](phase-0/P0-T01a.md#bzlmod-một-gomod-làm-nguồn-sự-thật-cho-cả-go-và-bazel) |
| `bazel/gazelle` | P0-T01a | Gazelle: `go_naming_convention import` và `map_kind go_binary` | [P0-T01a.md](phase-0/P0-T01a.md#gazelle-go_naming_convention-import-và-map_kind-go_binary) |
| `bazel/rules_oci`, `bazel/macro`, `docker` | P0-T01a | Macro `com_tm_go_image`: binary + layer + OCI image distroless | [P0-T01a.md](phase-0/P0-T01a.md#macro-com_tm_go_image-binary--layer--oci-image-distroless) |
| `bazel/toolchain`, `bazel/platforms` | P0-T01a | Cross-build Linux trên macOS: `oci_load` dùng tar của platform đích | [P0-T01a.md](phase-0/P0-T01a.md#cross-build-linux-trên-macos-oci_load-dùng-tar-của-platform-đích) |
| `go/testing` | P0-T01a | Binary mẫu tách `run(io.Writer)` khỏi `main` | [P0-T01a.md](phase-0/P0-T01a.md#binary-mẫu-tách-runiowriter-khỏi-main) |
| `node/pnpm`, `node/tooling` | P0-T01b | Pin phiên bản pnpm bằng `packageManager`, không cần corepack | [P0-T01b.md](phase-0/P0-T01b.md#pin-phiên-bản-pnpm-bằng-packagemanager-không-cần-corepack) |
| `node/pnpm` | P0-T01b | Script gốc `pnpm --recursive --if-present`, `dev` thêm `--parallel` | [P0-T01b.md](phase-0/P0-T01b.md#script-gốc-pnpm---recursive---if-present-dev-thêm---parallel) |
| `node/pnpm`, `ci` | P0-T01b | `pnpm ... lint test build` chỉ chạy `lint`: dùng `run '/regex/'` | [P0-T01b.md](phase-0/P0-T01b.md#pnpm--lint-test-build-chỉ-chạy-lint-dùng-run-regex) |
| `node/pnpm`, `ci` | P0-T01b | pnpm 11 tự cài lại deps trước khi chạy script | [P0-T01b.md](phase-0/P0-T01b.md#pnpm-11-tự-cài-lại-deps-trước-khi-chạy-script) |
| `docker/compose`, `docker/healthcheck`, `otel/collector` | P0-T02 | Healthcheck thật cho từng image, kể cả image distroless | [P0-T02.md](phase-0/P0-T02.md#healthcheck-thật-cho-từng-image-kể-cả-image-distroless) |
| `observability/tempo` | P0-T02 | Chọn Tempo 3.0.3 thay vì 2.10.x | [P0-T02.md](phase-0/P0-T02.md#chọn-tempo-303-thay-vì-210x) |
| `observability/prometheus`, `otel/collector` | P0-T02 | Metric vào Prometheus bằng OTLP push, không scrape exporter | [P0-T02.md](phase-0/P0-T02.md#metric-vào-prometheus-bằng-otlp-push-không-scrape-exporter) |
| `docker/desktop`, `docker/compose`, `macos` | P0-T02 | Không bind mount từ `~/Documents`: bake cấu hình vào image local | [P0-T02.md](phase-0/P0-T02.md#không-bind-mount-từ-documents-bake-cấu-hình-vào-image-local) |
| `mongo/ops`, `docker` | P0-T02 | MongoDB 8.x không khởi động trên kernel ≥ 6.19: `GLIBC_TUNABLES=glibc.pthread.rseq=1` | [P0-T02.md](phase-0/P0-T02.md#mongodb-8x-không-khởi-động-trên-kernel--619-glibc_tunablesglibcpthreadrseq1) |
| `pg/ops`, `docker` | P0-T02 | PostgreSQL 18: volume mount ở `/var/lib/postgresql`, không phải `.../data` | [P0-T02.md](phase-0/P0-T02.md#postgresql-18-volume-mount-ở-varlibpostgresql-không-phải-data) |
| `bash`, `testing`, `docker/compose` | P0-T02 | Script kiểm tĩnh compose: pin image + healthcheck không giả | [P0-T02.md](phase-0/P0-T02.md#script-kiểm-tĩnh-compose-pin-image--healthcheck-không-giả) |
| `pg/migration`, `go/tooling` | P0-T03 | Pin goose bằng `go install ...@v3.28.0`, chạy trên host | [P0-T03.md](phase-0/P0-T03.md#pin-goose-bằng-go-install-v3280-chạy-trên-host) |
| `pg/migration` | P0-T03 | Migration rỗng đầu tiên: SQL, đánh số tuần tự, annotation Up/Down | [P0-T03.md](phase-0/P0-T03.md#migration-rỗng-đầu-tiên-sql-đánh-số-tuần-tự-annotation-updown) |
| `ci/github-actions` | P0-T04 | Một workflow: job `changes` + `if`, không lọc bằng `paths:` | [P0-T04.md](phase-0/P0-T04.md#một-workflow-job-changes--if-không-lọc-bằng-paths) |
| `bazel`, `bazel/query`, `ci` | P0-T04 | Target Bazel bị ảnh hưởng bằng `bazel query rdeps` (G14) | [P0-T04.md](phase-0/P0-T04.md#target-bazel-bị-ảnh-hưởng-bằng-bazel-query-rdeps-g14) |
| `node/pnpm`, `ci` | P0-T04 | Job app: install frozen, filter theo merge-base, chạy tất cả khi file gốc workspace đổi | [P0-T04.md](phase-0/P0-T04.md#job-app-install-frozen-filter-theo-merge-base-chạy-tất-cả-khi-file-gốc-workspace-đổi) |
| `go/lint`, `ci` | P0-T04 | golangci-lint v2: pin bản, preset `standard`, không bật `std-error-handling` | [P0-T04.md](phase-0/P0-T04.md#golangci-lint-v2-pin-bản-preset-standard-không-bật-std-error-handling) |
| `bash`, `testing`, `ci` | P0-T04 | Test script CI bằng binary giả qua biến môi trường | [P0-T04.md](phase-0/P0-T04.md#test-script-ci-bằng-binary-giả-qua-biến-môi-trường) |
| `make`, `bash` | P0-T05 | Makefile mỏng cho GNU Make 3.81, logic nằm trong `scripts/` | [P0-T05.md](phase-0/P0-T05.md#makefile-mỏng-cho-gnu-make-381-logic-nằm-trong-scripts) |
| `pg/migration`, `go/tooling`, `bash` | P0-T05 | `make migrate`: tìm goose ngoài PATH, không treo, không che lỗi DB sau | [P0-T05.md](phase-0/P0-T05.md#make-migrate-tìm-goose-ngoài-path-không-treo-không-che-lỗi-db-sau) |
| `bash`, `testing`, `node/pnpm` | P0-T05 | `make test`: chạy mọi bộ, gom lỗi, không để bước cuối che bước đầu | [P0-T05.md](phase-0/P0-T05.md#make-test-chạy-mọi-bộ-gom-lỗi-không-để-bước-cuối-che-bước-đầu) |
| `observability/grafana`, `docker` | P0-T06 | Provisioning dashboard từ file, bake vào image | [P0-T06.md](phase-0/P0-T06.md#provisioning-dashboard-từ-file-bake-vào-image) |
| `observability/grafana` | P0-T06 | `disableDeletion` + `allowUiUpdates: false`: dashboard là code | [P0-T06.md](phase-0/P0-T06.md#disabledeletion--allowuiupdates-false-dashboard-là-code) |
| `observability/prometheus`, `otel/semconv` | P0-T06 | Label service là `job` khi metric vào bằng OTLP push | [P0-T06.md](phase-0/P0-T06.md#label-service-là-job-khi-metric-vào-bằng-otlp-push) |
| `observability/promql` | P0-T06 | Error rate: `(5xx or total * 0) / total` để service không lỗi ra 0 | [P0-T06.md](phase-0/P0-T06.md#error-rate-5xx-or-total--0--total-để-service-không-lỗi-ra-0) |
| `observability/grafana`, `docker/healthcheck` | P0-T06 | Grafana 13 tải plugin datasource ngầm sau health: cài sẵn khi build | [P0-T06.md](phase-0/P0-T06.md#grafana-13-tải-plugin-datasource-ngầm-sau-health-cài-sẵn-khi-build) |
| `go/config`, `go/errors`, `security` | P0-T07 | Config từ env: validate hết trước khi listen, lỗi nêu tên biến, không lộ mật khẩu | [P0-T07.md](phase-0/P0-T07.md#config-từ-env-validate-hết-trước-khi-listen-lỗi-nêu-tên-biến-không-lộ-mật-khẩu) |
| `pg/pool`, `go/pgx`, `ops/health` | P0-T07 | Pool lười: core khởi động khi PG chưa lên, `/readyz` tự hồi phục | [P0-T07.md](phase-0/P0-T07.md#pool-lười-core-khởi-động-khi-pg-chưa-lên-readyz-tự-hồi-phục) |
| `go/context`, `go/interface`, `go/testing` | P0-T07 | `/readyz` có deadline riêng, phụ thuộc interface `pinger` ở phía dùng | [P0-T07.md](phase-0/P0-T07.md#readyz-có-deadline-riêng-phụ-thuộc-interface-pinger-ở-phía-dùng) |
| `go/http`, `go/chi` | P0-T07 | chi: `r.Method(GET, ...)` cho `/metrics` để có 405 | [P0-T07.md](phase-0/P0-T07.md#chi-rmethodget--cho-metrics-để-có-405) |
| `observability/prometheus`, `docker/compose` | P0-T07 | `/metrics` bằng `promhttp` + Prometheus scrape service trên host | [P0-T07.md](phase-0/P0-T07.md#metrics-bằng-promhttp--prometheus-scrape-service-trên-host) |
| `bazel`, `bazel/bzlmod`, `go/modules` | P0-T07 | Thêm dependency Go vào workspace Bazel | [P0-T07.md](phase-0/P0-T07.md#thêm-dependency-go-vào-workspace-bazel) |

## Mẫu một file

```markdown
# Handbook — P4-T03: Use case hold ghế

## Update có điều kiện thay cho SELECT ... FOR UPDATE
- **Chủ đề**: `pg/lock`, `pg/isolation`
- **Bối cảnh**: 500 request cùng giữ ghế A05.
- **Cách làm & lý do**: `UPDATE ... WHERE status = 'AVAILABLE' RETURNING`, so số dòng trả về; không cần khoá tường minh vì ...
- **Bẫy / lưu ý**: phải sắp `seat_id` trước khi update nhiều ghế, nếu không sẽ deadlock (tái hiện ở P4-AT14).
- **Code**: [booking/store.go#L30-L55 — holdSeats](../../../../server/services/core/internal/booking/store.go#L30-L55)
- **Tham khảo**: PostgreSQL docs — Explicit Locking
```
