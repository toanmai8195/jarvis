# Phase 0 — Nền móng (tối giản cho chặng A)

> Chỉ dựng những gì chặng A (Go + PostgreSQL) cần. MongoDB, Redis, BFF, observability đầy đủ được thêm ở phase dùng tới chúng.

## Mục tiêu

Có repo, Bazel + Go, PostgreSQL local, migration, một core service "rỗng" chạy được (health, log, dừng êm, image), CI cho Go.

**Mốc demo**: `make up && make migrate` → `go run` core → `curl /readyz` trả 200; CI xanh.

## Kiến thức trọng tâm

Go: module, package `main`, `net/http`, chi, `context`, `slog`, `signal` · PG: chạy bằng Docker, migration với goose · Công cụ: Bazel + Gazelle, Docker Compose, Makefile, GitHub Actions

## Phạm vi

- **Trong**: cấu trúc repo, Docker Compose (chỉ PG core), goose, Bazel + Go, skeleton core, graceful shutdown, image OCI, testcontainers-go, Makefile, CI Go.
- **Ngoài**: logic nghiệp vụ; MongoDB, Redis, BFF, web (chặng B, C); OpenTelemetry/Grafana (Phase 6); PG analytics (Phase 6).

## Workstream

`infra` · `db` · `core` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P0-FR1 | FR | Một lệnh khởi động hạ tầng local (PG core) |
| P0-FR2 | FR | Migration chạy được cho PG core |
| P0-FR3 | FR | `core` có `/healthz` (sống) và `/readyz` (kết nối được DB) |
| P0-FR4 | FR | CI chạy lint, build, test Go trên mỗi PR |
| P0-NFR1 | NFR | Log dạng JSON một dòng |
| P0-NFR2 | NFR | Service dừng an toàn khi nhận SIGTERM, không cắt ngang request |

## Task

### infra
- [ ] **P0-T01** Cấu trúc thư mục `com/tm/server`, `.gitignore` cho Go/Bazel
- [ ] **P0-T02** `deploy/docker-compose.yml` chỉ có PostgreSQL core (5432), healthcheck, volume

### db
- [ ] **P0-T03** goose migration cho PG core: `scripts/migrate.sh`, migration rỗng đầu tiên

### core
- [ ] **P0-T04** Bazel + Gazelle + một `go.mod` cho `com/tm/server` `[G14]`
- [ ] **P0-T05** Skeleton core: hello world → HTTP server → chi router + `/healthz` → config từ env → log `slog` JSON → pgxpool + `/readyz`
- [ ] **P0-T06** Graceful shutdown: SIGTERM, `http.Server.Shutdown` có timeout, đóng pool sau cùng `[G3]`
- [ ] **P0-T07** Image OCI cho core bằng macro `com_tm_go_image` (rules_oci, distroless, non-root) `[G14]`

### qa
- [ ] **P0-T08** Khung testcontainers-go cho integration test với PG

### infra
- [ ] **P0-T09** Makefile: `up`, `down`, `migrate`, `test`, `gazelle`
- [ ] **P0-T10** GitHub Actions cho Go: gazelle diff, golangci-lint, `bazel test` target bị ảnh hưởng `[G14]`

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| G3 | Golang | Graceful shutdown | Deploy khi đang có giao dịch | Bắt SIGTERM, ngừng nhận request, chờ in-flight, đóng pool theo thứ tự | Rolling deploy dưới tải không mất/không lỗi request | ⬜ |
| G14 | Golang | Monorepo Go với Bazel | `com/tm/server` nhiều service + thư viện | rules_go + Gazelle + bzlmod, một `go.mod`, visibility, test theo target bị ảnh hưởng | Code build được bằng cả `go` và Bazel; CI chỉ test target bị ảnh hưởng; core build được thành image | ⬜ |

## Definition of Done

- [ ] Clone repo mới → `make up && make migrate` chạy thành công trong < 5 phút
- [ ] Core chạy được bằng `go run` và bằng image Docker; `/readyz` phản ánh đúng trạng thái PG
- [ ] Gửi SIGTERM khi đang có request chậm → request hoàn thành, service thoát sạch
- [ ] CI xanh trên `main`
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [local-setup](../../local-setup.md) theo thực tế
- [ ] ADR: Bazel cho Go + pnpm cho TS; công cụ migration
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
