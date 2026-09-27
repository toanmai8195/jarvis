# Phase 7 — BFF nền tảng & đăng nhập

> Chặng B — Node.js + MongoDB (BFF) · Công nghệ: **Node.js + MongoDB**

## Mục tiêu

Học Node qua BFF: OTel, dừng êm, OAuth Google, session MongoDB, CSRF, gọi core tin cậy.

**Mốc demo**: Đăng nhập Google, `curl` tìm chuyến qua BFF; một trace đi bff → core → PG

## Kiến thức trọng tâm

Node: event loop, async, Fastify plugin/hook, undici, stream · Mongo: document, index, TTL

## Phạm vi

- **Trong**: OTel Node, graceful shutdown, auth, session, core client, API tìm chuyến public.
- **Ngoài**: Giao diện web.

## Workstream

`bff` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P7-FR1 | FR | Đăng nhập bằng Google; lần đầu tạo user ở MongoDB và user + ví ở core |
| P7-FR2 | FR | Đăng xuất xoá session |
| P7-FR3 | FR | `GET /api/me` trả thông tin người dùng hiện tại |
| P7-FR4 | FR | API public tìm chuyến, chi tiết, sơ đồ ghế (gọi core) |
| P7-NFR1 | NFR | Cookie session HttpOnly, Secure, SameSite=Lax; CSRF cho mọi request ghi |
| P7-NFR2 | NFR | Core lỗi/chậm → BFF trả lỗi trong ≤ 2s, không treo |
| P7-FR5 | FR | BFF có `/healthz` và `/readyz` (kết nối được MongoDB) |

## Task

### infra (chuyển từ Phase 0)
- [ ] **P7-T01** Thêm MongoDB vào compose; pnpm workspace `com/tm/app` (`pnpm-workspace.yaml`, script gốc)
- [ ] **P7-T02** CI cho app: `pnpm install --frozen-lockfile`, lint/test/build theo package thay đổi
- [ ] **P7-T03** Skeleton BFF Fastify + TypeScript (ESM, strict, tsx/tsup): config zod, log pino JSON, request ID, `/healthz`, `/readyz` (Mongo)

### bff
- [ ] **P7-T04** OpenTelemetry cho Node (`node --import`); `bff /healthz?deep=1` gọi endpoint có trace của core (thêm `core GET /internal/v1/ping`) để kiểm tra trace bff → core → PG `[G10]` _(trước đây P0-T12)_
- [ ] **P7-T05** Graceful shutdown Fastify: SIGTERM → `app.close()`, chờ request đang chạy, đóng MongoClient sau cùng _(trước đây P0-T13)_
- [ ] **P7-T06** Plugin MongoDB, collection `users`, `sessions` với index + TTL `[M1][M2]` _(trước đây P2-T01)_
- [ ] **P7-T07** Luồng OAuth Google (`@fastify/oauth2`), kiểm tra `state`, xác thực ID token `[N3]` _(trước đây P2-T02)_
- [ ] **P7-T08** Session tự quản lý: tạo, xoay session id sau đăng nhập, hết hạn, đăng xuất `[N3]` _(trước đây P2-T03)_
- [ ] **P7-T09** CSRF token (double submit hoặc synchronizer) cho POST/PUT/PATCH/DELETE `[N3]` _(trước đây P2-T04)_
- [ ] **P7-T10** Core client (undici): keep-alive, timeout, retry có giới hạn cho request idempotent, circuit breaker `[N2]` _(trước đây P2-T05)_
- [ ] **P7-T11** Đồng bộ user sang core idempotent + job đối soát user thiếu ví `[M3]` _(trước đây P2-T06)_
- [ ] **P7-T12** Zod schema cho request/response; sinh type TS từ `core.openapi.yaml` `[N8]` _(trước đây P2-T07)_
- [ ] **P7-T13** Route `/api/auth/*`, `/api/me`, `/api/stations`, `/api/trips*` _(trước đây P2-T08)_
- [ ] **P7-T14** Error handler thống nhất định dạng lỗi _(trước đây P2-T09)_

### qa
- [ ] **P7-T15** Test BFF với core giả lập (mock server) cho timeout/lỗi _(trước đây P2-T17)_

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| G10 | Golang | Observability | Debug trên nhiều service | OpenTelemetry trace, slog JSON, Prometheus metrics | Một trace hiển thị đủ BFF → core → PG | 🟨 |
| M1 | MongoDB | Thiết kế document | Profile, preference, hành khách lưu sẵn | Embed vs reference, schema validation | Mỗi màn hình đọc bằng 1 query | ⬜ |
| M2 | MongoDB | Index & TTL | Session, audit log | Unique index, compound index, TTL index | `explain()` không có COLLSCAN trên truy vấn nóng | ⬜ |
| N3 | Node.js | Xác thực an toàn | Login Google, session | OAuth/OIDC, cookie HttpOnly/SameSite, CSRF, xoay session | Qua checklist OWASP cho auth | ⬜ |
| N2 | Node.js | Gọi core tin cậy | Core chậm hoặc lỗi tạm thời | Timeout, retry có idempotency key, circuit breaker, keep-alive agent (undici) | Core down → BFF trả lỗi nhanh, không treo | ⬜ |
| M3 | MongoDB | Đồng bộ user với core | Tạo user Mongo + user/ví core | Upsert idempotent, retry, job đối soát | User nào ở Mongo cũng có ví ở core | ⬜ |
| N8 | Node.js | Validate và type an toàn đầu cuối | Web ↔ BFF ↔ core | Zod schema, sinh type từ OpenAPI | Đổi contract core → build TS fail | ⬜ |

## Definition of Done

- [ ] Gọi `bff /healthz?deep=1` → một trace gồm span bff, core và PG trên Grafana
- [ ] Đăng nhập → tìm chuyến → đăng xuất bằng `curl` / trình duyệt
- [ ] Tắt core → BFF trả lỗi trong ≤ 2s
- [ ] Đổi field trong `core.openapi.yaml` → build BFF fail
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [api](../../api.md), [database](../../database.md) phần MongoDB
- [ ] ADR: quản lý session, chiến lược CSRF
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
