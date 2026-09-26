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
- [ ] **P0-T01** Khởi tạo cấu trúc `com/tm/{server,app,docs}`, `deploy/`, `loadtest/` theo [project-structure](../../project-structure.md)
- [ ] **P0-T01a** `com/tm/server`: `MODULE.bazel` (rules_go, gazelle, rules_oci), `.bazelversion`, `.bazelrc`, một `go.mod`, target `//:gazelle` với `gazelle:prefix` và `go_naming_convention import`; macro `com_tm_go_image` (`tools/rules`) build binary + OCI image distroless, gazelle `map_kind` cho `go_binary` (theo repo thor) `[G14]`
- [ ] **P0-T01b** `com/tm/app`: `pnpm-workspace.yaml`, `package.json` gốc, script `dev`/`build`/`test` chạy theo filter
- [ ] **P0-T02** `deploy/docker-compose.yml`: PG core (5432), PG analytics (5433), MongoDB, Redis, otel-collector, Prometheus, Grafana, Tempo/Jaeger
- [ ] **P0-T03** Cấu hình goose, thư mục `com/tm/server/db/core/migrations`, `com/tm/server/db/analytics/migrations`, migration rỗng đầu tiên
- [ ] **P0-T04** GitHub Actions chạy theo đường dẫn thay đổi: `com/tm/server/**` → `golangci-lint` + `bazel test` target bị ảnh hưởng; `com/tm/app/**` → `pnpm --filter "...[origin/main]" lint test build` `[G14]`
- [ ] **P0-T05** Makefile / script: `make up`, `make migrate`, `make test`
- [ ] **P0-T06** Dashboard Grafana cơ bản: RED metrics cho mỗi service

### core
- [ ] **P0-T07** Skeleton `com/tm/server/services/core`: config (env), slog JSON, chi router, pgxpool, `/healthz`, `/readyz`, `/metrics`
- [ ] **P0-T08** Middleware: request ID, recover, access log, OTel HTTP
- [ ] **P0-T09** Graceful shutdown: bắt SIGTERM, `http.Server.Shutdown` có timeout, đóng pool sau cùng `[G3]`
- [ ] **P0-T10** Tích hợp OpenTelemetry SDK trong `pkg/otelx`, export OTLP `[G10]`
- [ ] **P0-T10a** Image OCI cho core: dùng macro `com_tm_go_image` (có từ P0-T01a) cho `cmd/server`, `cmd/worker` `[G14]`

### bff
- [ ] **P0-T11** Skeleton `com/tm/app/apps/bff` Fastify + TS (ESM, strict, `tsx` khi dev, `tsup` khi build): plugin config, logger pino JSON, `/healthz`, `/readyz`
- [ ] **P0-T12** OTel cho Node, gọi thử `core /healthz` để kiểm tra trace xuyên service `[G10]`
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
| G3 | Golang | Graceful shutdown | Deploy khi đang có giao dịch | Bắt SIGTERM, ngừng nhận request, chờ in-flight, đóng pool theo thứ tự | Rolling deploy dưới tải không mất/không lỗi request | ⬜ |
| G10 | Golang | Observability | Debug trên nhiều service | OpenTelemetry trace, slog JSON, Prometheus metrics | Một trace hiển thị đủ BFF → core → PG | ⬜ |
| G14 | Golang | Monorepo Go với Bazel | `com/tm/server` nhiều service + thư viện | rules_go + Gazelle + bzlmod, một `go.mod`, visibility, test theo target bị ảnh hưởng | Code build được bằng cả `go` và Bazel; CI chỉ test target bị ảnh hưởng | ⬜ |

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
