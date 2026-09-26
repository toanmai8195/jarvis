# Phase 2 — Đăng nhập, BFF, web client

## Mục tiêu

Hoàn thiện BFF (Fastify) với đăng nhập Google, session, gọi core tin cậy; web client hiển thị tìm chuyến và sơ đồ ghế (chỉ xem).

**Mốc demo**: mở web, đăng nhập Google, tìm chuyến, xem chi tiết và sơ đồ ghế; đăng xuất.

## Phạm vi

- **Trong**: OAuth Google, session MongoDB, CSRF, đồng bộ user sang core, core client, API public tìm chuyến, web client (trang chủ, kết quả, chi tiết chuyến, trang cá nhân).
- **Ngoài**: đặt vé, ví, admin.

## Workstream

`bff` · `core` · `web` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P2-FR1 | FR | Đăng nhập bằng Google; lần đầu tạo user ở MongoDB và user + ví ở core |
| P2-FR2 | FR | Đăng xuất xoá session |
| P2-FR3 | FR | `GET /api/me` trả thông tin người dùng hiện tại |
| P2-FR4 | FR | API public tìm chuyến, chi tiết, sơ đồ ghế (gọi core) |
| P2-FR5 | FR | Web: tìm chuyến, xem kết quả, xem chi tiết + sơ đồ ghế |
| P2-FR6 | FR | Web: trang cần đăng nhập tự chuyển sang đăng nhập rồi quay lại |
| P2-NFR1 | NFR | Cookie session HttpOnly, Secure, SameSite=Lax; CSRF cho mọi request ghi |
| P2-NFR2 | NFR | Core lỗi/chậm → BFF trả lỗi trong ≤ 2s, không treo |
| P2-NFR3 | NFR | JS tải ban đầu web-client < 200KB gzip |

## Task

### bff
- [ ] **P2-T01** Plugin MongoDB, collection `users`, `sessions` với index + TTL `[M1][M2]`
- [ ] **P2-T02** Luồng OAuth Google (`@fastify/oauth2`), kiểm tra `state`, xác thực ID token `[N3]`
- [ ] **P2-T03** Session tự quản lý: tạo, xoay session id sau đăng nhập, hết hạn, đăng xuất `[N3]`
- [ ] **P2-T04** CSRF token (double submit hoặc synchronizer) cho POST/PUT/PATCH/DELETE `[N3]`
- [ ] **P2-T05** Core client (undici): keep-alive, timeout, retry có giới hạn cho request idempotent, circuit breaker `[N2]`
- [ ] **P2-T06** Đồng bộ user sang core idempotent + job đối soát user thiếu ví `[M3]`
- [ ] **P2-T07** Zod schema cho request/response; sinh type TS từ `core.openapi.yaml` `[N8]`
- [ ] **P2-T08** Route `/api/auth/*`, `/api/me`, `/api/stations`, `/api/trips*`
- [ ] **P2-T09** Error handler thống nhất định dạng lỗi

### core
- [ ] **P2-T10** `POST /internal/v1/users` idempotent: tạo `users` + `accounts` (ví) trong một transaction
- [ ] **P2-T11** Middleware xác thực service token, đọc `X-User-Id`, `X-User-Role`

### web
- [ ] **P2-T12** React Router, layout, TanStack Query, API client dùng chung (`packages/api-client`)
- [ ] **P2-T13** Trang chủ + form tìm chuyến (autocomplete trạm)
- [ ] **P2-T14** Trang kết quả: lọc, sắp xếp; trang chi tiết + sơ đồ ghế (chỉ xem)
- [ ] **P2-T15** Auth context, protected route, redirect về trang cũ sau đăng nhập `[R10]`
- [ ] **P2-T16** Code splitting theo route, phân tích bundle `[R11]`

### qa
- [ ] **P2-T17** Test BFF với core giả lập (mock server) cho timeout/lỗi
- [ ] **P2-T18** Test component tìm chuyến bằng Testing Library

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| N2 | Node.js | Gọi core tin cậy | Core chậm hoặc lỗi tạm thời | Timeout, retry có idempotency key, circuit breaker, keep-alive agent (undici) | Core down → BFF trả lỗi nhanh, không treo | ⬜ |
| N3 | Node.js | Xác thực an toàn | Login Google, session | OAuth/OIDC, cookie HttpOnly/SameSite, CSRF, xoay session | Qua checklist OWASP cho auth | ⬜ |
| N8 | Node.js | Validate và type an toàn đầu cuối | Web ↔ BFF ↔ core | Zod schema, sinh type từ OpenAPI | Đổi contract core → build TS fail | ⬜ |
| M1 | MongoDB | Thiết kế document | Profile, preference, hành khách lưu sẵn | Embed vs reference, schema validation | Mỗi màn hình đọc bằng 1 query | ⬜ |
| M2 | MongoDB | Index & TTL | Session, audit log | Unique index, compound index, TTL index | `explain()` không có COLLSCAN trên truy vấn nóng | ⬜ |
| M3 | MongoDB | Đồng bộ user với core | Tạo user Mongo + user/ví core | Upsert idempotent, retry, job đối soát | User nào ở Mongo cũng có ví ở core | ⬜ |
| R10 | React | Route guard & phân quyền UI | Trang cần đăng nhập, admin theo vai trò | Protected route, loader kiểm tra session, ẩn chức năng theo vai trò (server vẫn là nơi quyết định) | Truy cập trái phép bị chuyển hướng, không nháy nội dung | ⬜ |
| R11 | React | Bundle nhỏ, tải nhanh trên mobile | Người dùng đặt vé bằng điện thoại, mạng yếu | Code splitting theo route (`lazy`), phân tích bundle, prefetch dữ liệu khi hover | JS ban đầu < 200KB gzip, LCP < 2.5s trên 4G chậm | ⬜ |

## Definition of Done

- [ ] Luồng đăng nhập → tìm chuyến → xem ghế → đăng xuất chạy trên web
- [ ] Mỗi user ở MongoDB đều có user + ví ở core (job đối soát báo 0 lệch)
- [ ] Tắt core → web hiển thị lỗi thân thiện trong ≤ 2s
- [ ] Đổi một field trong `core.openapi.yaml` → build BFF fail
- [ ] Đạt P2-NFR3
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [api](../../api.md), [services](../../services.md), [database](../../database.md) (phần MongoDB)
- [ ] ADR: cách quản lý session (tự viết vs thư viện), chiến lược CSRF
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
