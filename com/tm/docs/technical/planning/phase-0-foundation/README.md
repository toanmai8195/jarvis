# Phase 0 — Nền móng

## Mục tiêu

Dựng khung monorepo, hạ tầng local, CI và observability để mọi phase sau chỉ tập trung vào nghiệp vụ.

**Mốc demo**: `docker compose up` chạy đủ PostgreSQL, MongoDB, Redis, Grafana; service Go và Fastify "hello world" có health check, log JSON, trace hiển thị trên Grafana; CI xanh.

## Phạm vi

- **Trong**: cấu trúc repo, Docker Compose, migration tool, CI, lint/format, skeleton service, logging/tracing/metrics.
- **Ngoài**: mọi logic nghiệp vụ.

## Workstream

`infra` · `core` · `bff` · `web` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P0-FR1 | FR | Một lệnh khởi động toàn bộ hạ tầng local |
| P0-FR2 | FR | Migration chạy được cho PG core và PG analytics |
| P0-FR3 | FR | `core` và `bff` có `/healthz` (sống) và `/readyz` (kết nối được DB) |
| P0-FR4 | FR | CI chạy lint, build, test cho Go và TS trên mỗi PR |
| P0-NFR1 | NFR | Log dạng JSON, có `trace_id`, `request_id` |
| P0-NFR2 | NFR | Trace truyền từ `bff` sang `core` qua `traceparent` |
| P0-NFR3 | NFR | Service dừng an toàn khi nhận SIGTERM, không cắt ngang request |

## Task

### infra
- [x] **P0-T01** Khởi tạo cấu trúc `com/tm/{server,app,docs}`, `deploy/`, `loadtest/` theo [project-structure](../../project-structure.md)
  - [x] 1. Test case: P0-T01-TC01..TC14 — đã được duyệt (review agent, execute-all)
  - [x] 2. Code
  - [x] 3. Unit test
  - [x] 4. Build + unit test pass
  - [x] 5. Test case pass + handbook
  - [x] 6. Commit: `chore(repo): khởi tạo khung monorepo và script kiểm tra cấu trúc [P0-T01]` · Push: không (execute-all push khi đóng phase)
- [x] **P0-T01a** `com/tm/server`: `MODULE.bazel` (rules_go, gazelle, rules_oci), `.bazelversion`, `.bazelrc`, một `go.mod`, target `//:gazelle` với `gazelle:prefix` và `go_naming_convention import`; macro `com_tm_go_image` (`tools/rules`) build binary + OCI image distroless, gazelle `map_kind` cho `go_binary` (theo repo thor) `[G14]`
  - [x] 1. Test case: P0-T01a-TC01..TC22 — đã được duyệt (review agent, execute-all)
  - [x] 2. Code
  - [x] 3. Unit test
  - [x] 4. Build + unit test pass
  - [x] 5. Test case pass + handbook
  - [x] 6. Commit: `build(server): Bazel workspace bzlmod, Gazelle, go.mod và macro com_tm_go_image distroless [P0-T01a][G14]` · Push: không (execute-all push khi đóng phase)
- [x] **P0-T01b** `com/tm/app`: `pnpm-workspace.yaml`, `package.json` gốc, script `dev`/`build`/`test` chạy theo filter
  - [x] 1. Test case: P0-T01b-TC01..TC19 — đã được duyệt (review agent, execute-all)
  - [x] 2. Code
  - [x] 3. Unit test
  - [x] 4. Build + unit test pass
  - [x] 5. Test case pass + handbook
  - [x] 6. Commit: `build(app): pnpm workspace, package.json gốc pin pnpm@11.18.0 và script chạy theo filter [P0-T01b]` · Push: không (execute-all push khi đóng phase)
- [x] **P0-T02** `deploy/docker-compose.yml`: PG core (5432), PG analytics (5433), MongoDB, Redis, otel-collector, Prometheus, Grafana, Tempo/Jaeger
  - [x] 1. Test case: P0-T02-TC01..TC25 — đã được duyệt (review agent, execute-all)
  - [x] 2. Code
  - [x] 3. Unit test
  - [x] 4. Build + unit test pass
  - [x] 5. Test case pass + handbook
  - [x] 6. Commit: `build(deploy): docker compose hạ tầng local PG core/analytics, MongoDB, Redis, otel-collector, Prometheus, Tempo, Grafana với healthcheck thật [P0-T02]` · Push: không (execute-all push khi đóng phase)
- [x] **P0-T03** Cấu hình goose, thư mục `com/tm/server/db/core/migrations`, `com/tm/server/db/analytics/migrations`, migration rỗng đầu tiên
  - [x] 1. Test case: P0-T03-TC01..TC18 — đã được duyệt (review agent, execute-all)
  - [x] 2. Code
  - [x] 3. Unit test
  - [x] 4. Build + unit test pass
  - [x] 5. Test case pass + handbook
  - [x] 6. Commit: `build(server): goose v3.28.0, thư mục migration db/core và db/analytics với migration rỗng 00001_init [P0-T03]` · Push: không (execute-all push khi đóng phase)
- [x] **P0-T04** GitHub Actions chạy theo đường dẫn thay đổi: `com/tm/server/**` → `golangci-lint` + `bazel test` target bị ảnh hưởng; `com/tm/app/**` → `pnpm --filter "...[origin/main]" lint test build` `[G14]`
  - [x] 1. Test case: P0-T04-TC01..TC27 — đã được duyệt (review agent, execute-all)
  - [x] 2. Code
  - [x] 3. Unit test
  - [x] 4. Build + unit test pass
  - [x] 5. Test case pass + handbook
  - [x] 6. Commit: `ci: GitHub Actions theo đường dẫn thay đổi — golangci-lint v2.14.0 + bazel test target bị ảnh hưởng, pnpm lint/test/build package bị ảnh hưởng [P0-T04][G14]` · Push: không (execute-all push khi đóng phase)
- [x] **P0-T05** Makefile / script: `make up`, `make migrate`, `make test`
  - [x] 1. Test case: P0-T05-TC01..TC22 — đã được duyệt (review agent, execute-all)
  - [x] 2. Code
  - [x] 3. Unit test
  - [x] 4. Build + unit test pass
  - [x] 5. Test case pass + handbook
  - [x] 6. Commit: `build: Makefile make up/migrate/test — scripts/migrate.sh tìm goose ngoài PATH và gom lỗi core/analytics, scripts/test-all.sh chạy scripts/server/app không che lỗi [P0-T05]` · Push: không (execute-all push khi đóng phase)
- [x] **P0-T06** Dashboard Grafana cơ bản: RED metrics cho mỗi service
  - [x] 1. Test case: P0-T06-TC01..TC21 — đã được duyệt (review agent, execute-all)
  - [x] 2. Code
  - [x] 3. Unit test
  - [x] 4. Build + unit test pass
  - [x] 5. Test case pass + handbook
  - [x] 6. Commit: `feat(deploy): dashboard Grafana RED theo service provisioning từ file, cài sẵn plugin prometheus/tempo khi build [P0-T06]` · Push: không (execute-all push khi đóng phase)

### core
- [x] **P0-T07** Skeleton `com/tm/server/services/core`: config (env), slog JSON, chi router, pgxpool, `/healthz`, `/readyz`, `/metrics`
  - [x] 1. Test case: P0-T07-TC01..TC27 — đã được duyệt (review agent, execute-all)
  - [x] 2. Code
  - [x] 3. Unit test
  - [x] 4. Build + unit test pass
  - [x] 5. Test case pass + handbook
  - [x] 6. Commit: `feat(core): skeleton services/core — config env CORE_*, slog JSON, chi, pgxpool lười, /healthz, /readyz, /metrics, Prometheus scrape job core [P0-T07]` · Push: không (execute-all push khi đóng phase)
- [x] **P0-T08** Middleware: request ID, recover, access log, OTel HTTP
  - [x] 1. Test case: P0-T08-TC01..TC25 — đã được duyệt (review agent, execute-all)
  - [x] 2. Code
  - [x] 3. Unit test
  - [x] 4. Build + unit test pass
  - [x] 5. Test case pass + handbook
  - [x] 6. Commit: `feat(core): middleware request ID (X-Request-ID), recover 500 INTERNAL, access log slog JSON, OTel span + http.server.request.duration theo route pattern chi [P0-T08]` · Push: không (execute-all push khi đóng phase)
- [x] **P0-T09** Graceful shutdown: bắt SIGTERM, `http.Server.Shutdown` có timeout, đóng pool sau cùng `[G3]`
  - [x] 1. Test case: P0-T09-TC01..TC22 — đã được duyệt (review agent, execute-all)
  - [x] 2. Code
  - [x] 3. Unit test
  - [x] 4. Build + unit test pass
  - [x] 5. Test case pass + handbook
  - [x] 6. Commit: `feat(core): graceful shutdown — SIGTERM/SIGINT, http.Server.Shutdown có hạn CORE_SHUTDOWN_TIMEOUT, đóng pool PG sau cùng có hạn, log shutdown_step, exit 0/1 [P0-T09]` · Push: không (execute-all push khi đóng phase)
- [x] **P0-T10** Tích hợp OpenTelemetry SDK trong `pkg/otelx`, export OTLP `[G10]`
  - [x] 1. Test case: P0-T10-TC01..TC22 — đã được duyệt (review agent, execute-all)
  - [x] 2. Code
  - [x] 3. Unit test
  - [x] 4. Build + unit test pass
  - [x] 5. Test case pass + handbook
  - [x] 6. Commit: `feat(otelx): pkg/otelx — OTel SDK trace+metric export OTLP/HTTP, propagator TraceContext+Baggage, OTEL_SDK_DISABLED, validate endpoint; core: span PG qua otelpgx, flush telemetry song song đóng pool [P0-T10]` · Push: không (execute-all push khi đóng phase)
- [x] **P0-T10a** Image OCI cho core: dùng macro `com_tm_go_image` (có từ P0-T01a) cho `cmd/server`, `cmd/worker` `[G14]`
  - [x] 1. Test case: P0-T10a-TC01..TC17 — đã được duyệt (review agent, execute-all)
  - [x] 2. Code
  - [x] 3. Unit test
  - [x] 4. Build + unit test pass
  - [x] 5. Test case pass + handbook
  - [x] 6. Commit: `feat(core): image OCI core-server/core-worker bằng com_tm_go_image — attr image cho tag, tắt runfiles trong layer, skeleton cmd/worker [P0-T10a]` · Push: không (execute-all push khi đóng phase)

### bff
- [x] **P0-T11** Skeleton `com/tm/app/apps/bff` Fastify + TS (ESM, strict, `tsx` khi dev, `tsup` khi build): plugin config, logger pino JSON, `/healthz`, `/readyz`
  - [x] 1. Test case: P0-T11-TC01..TC22 — đã được duyệt (review agent, execute-all)
  - [x] 2. Code
  - [x] 3. Unit test
  - [x] 4. Build + unit test pass
  - [x] 5. Test case pass + handbook
  - [x] 6. Commit: `feat(bff): skeleton apps/bff Fastify + TS ESM strict — plugin config, logger pino JSON request_id, X-Request-ID, mongo client lười, /healthz, /readyz; tsx dev, tsup build, Vitest, ESLint [P0-T11]` · Push: không (execute-all push khi đóng phase)
- [x] **P0-T12** OTel cho Node, gọi thử `core /healthz` để kiểm tra trace xuyên service `[G10]`
  - [x] 1. Test case: P0-T12-TC01..TC23 — đã được duyệt (review agent, execute-all)
  - [x] 2. Code
  - [x] 3. Unit test
  - [x] 4. Build + unit test pass
  - [x] 5. Test case pass + handbook
  - [x] 6. Commit: `feat(bff): OTel Node — instrumentation.ts nạp bằng --import (http + @fastify/otel + undici), OTLP/HTTP, log trace_id/span_id, /healthz?deep=1 gọi core /readyz cho trace bff → core → PG, CORE_BASE_URL [P0-T12][G10]` · Push: không (execute-all push khi đóng phase)
- [ ] **P0-T13** Graceful shutdown Fastify (`close` hooks)

### web
- [ ] **P0-T14** Skeleton `web-client`, `web-admin` bằng Vite + React + TS + Tailwind + shadcn/ui
- [ ] **P0-T15** `com/tm/app/packages/config`: eslint, prettier, tsconfig dùng chung

### qa
- [ ] **P0-T16** Khung testcontainers-go cho integration test với PG
- [ ] **P0-T17** Khung Vitest cho TS

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| G3 | Golang | Graceful shutdown | Deploy khi đang có giao dịch | Bắt SIGTERM, ngừng nhận request, chờ in-flight, đóng pool theo thứ tự | Rolling deploy dưới tải không mất/không lỗi request | 🟨 |
| G10 | Golang | Observability | Debug trên nhiều service | OpenTelemetry trace, slog JSON, Prometheus metrics | Một trace hiển thị đủ BFF → core → PG | ✅ |
| G14 | Golang | Monorepo Go với Bazel | `com/tm/server` nhiều service + thư viện | rules_go + Gazelle + bzlmod, một `go.mod`, visibility, test theo target bị ảnh hưởng | Code build được bằng cả `go` và Bazel; CI chỉ test target bị ảnh hưởng | 🟨 |

## Definition of Done

- [ ] Clone repo mới → `make up && make migrate` chạy thành công trong < 5 phút
- [ ] CI xanh trên nhánh `main`; sửa một file trong `com/tm/app` không kích hoạt `bazel test` và ngược lại
- [ ] `go test ./...` và `bazel test //...` trong `com/tm/server` đều pass
- [ ] Gọi `bff /healthz?deep=1` → thấy **một trace** gồm span của bff và core trên Grafana
- [ ] Gửi SIGTERM khi đang có request chậm → request hoàn thành, service thoát sạch
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [local-setup](../../local-setup.md) theo thực tế
- [ ] ADR: Bazel cho Go + pnpm cho TS; lựa chọn công cụ migration, tracing backend
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
