# Cấu trúc dự án

## Tổng quan monorepo

```
snaptix/
├── README.md
├── .github/workflows/        # CI chạy theo đường dẫn thay đổi
├── deploy/                   # docker-compose, observability — dùng cho cả server lẫn app
├── loadtest/                 # kịch bản k6, kết quả theo phase
├── scripts/                  # script kiểm tra repo (cấu trúc, compose), script CI (ci-*.sh) + test *_test.sh
└── com/tm/
    ├── server/               # Go — Bazel workspace
    ├── app/                  # Node.js + React — pnpm workspace
    └── docs/                 # tài liệu
```

| Thư mục | Nội dung | Build | Lệnh dev |
|---|---|---|---|
| `com/tm/server` | Core service, stats-worker, thư viện Go, migration, hợp đồng core | **Bazel + Gazelle** | `go test ./...`, `bazel test //...` |
| `com/tm/app` | BFF (Fastify), web-client, web-admin, package TS dùng chung | **pnpm workspace** | `pnpm --filter <app> dev` |
| `com/tm/docs` | Hướng dẫn sử dụng, tài liệu kỹ thuật, planning, ADR | — | — |

Hai hệ build tách biệt, nối với nhau qua **hợp đồng OpenAPI**: `server/api/core.openapi.yaml` do core sở hữu, BFF sinh type TS từ đó.

---

## `com/tm/server` — Go + Bazel

```
com/tm/server/
├── MODULE.bazel              # rules_go, gazelle, rules_oci, tar.bzl, distroless base (bzlmod)
├── MODULE.bazel.lock         # commit để build tái lập
├── BUILD.bazel               # target gazelle + directive, platform linux_amd64 / linux_arm64
├── .bazelrc                  # pure Go, CGO off, --config=linux-arm64|amd64, profile release
├── .bazelversion             # 8.7.0 (rules_oci chưa hỗ trợ Bazel 9)
├── .golangci.yml             # golangci-lint v2 (pin v2.14.0), preset standard + gofmt
├── tools/
│   ├── rules/
│   │   ├── BUILD.bazel           # toolchain bsdtar cho `bazel run <name>_docker` khi cross-build
│   │   ├── com_tm_container.bzl  # macro com_tm_go_image: binary + OCI image
│   │   └── com_tm_container_test.bzl  # unit test (skylib) cho macro
│   └── smoke/                # binary mẫu tối thiểu kiểm chứng workspace + image
├── go.mod, go.sum            # MỘT module cho toàn bộ Go
├── api/
│   └── core.openapi.yaml     # hợp đồng API nội bộ của core
├── db/
│   ├── core/migrations/      # goose
│   └── analytics/migrations/
├── pkg/                      # dùng chung GIỮA các service (giữ nhỏ)
│   ├── postgres/             # pool, WithTx, retry 40001/40P01
│   ├── otelx/                # tracing, metrics, slog
│   └── events/               # schema sự kiện outbox — hợp đồng core ↔ stats-worker
└── services/
    ├── core/
    │   ├── cmd/
    │   │   ├── server/main.go    # wiring thủ công
    │   │   └── worker/main.go    # hết hạn hold, relay outbox, sinh slot, đối soát
    │   └── internal/
    │       ├── catalog/          # module: trạm, tuyến, phương tiện, sơ đồ ghế, chuyến, giá
    │       ├── booking/          # module: hold, booking, ticket
    │       ├── wallet/           # module: account, ledger, topup, refund
    │       ├── money/            # value type VND
    │       ├── idempotency/      # dùng bởi wallet, booking
    │       └── httpx/            # router, middleware, map lỗi → HTTP
    └── stats-worker/
        ├── cmd/worker/main.go
        └── internal/
```

### Thiết lập Bazel

| Hạng mục | Quy ước |
|---|---|
| Module Go | `github.com/toanmai8195/snaptix/com/tm/server` |
| Gazelle | `# gazelle:prefix github.com/toanmai8195/snaptix/com/tm/server`, `# gazelle:go_naming_convention import` |
| Dependency | `go_deps.from_file(go_mod = "//:go.mod")`; thêm thư viện → `go get` rồi `bazel mod tidy` |
| Target | 1 thư mục = 1 package Go = 1 `go_library`, do Gazelle sinh. Không viết BUILD tay chia nhỏ hơn |
| Sau khi thêm file | `bazel run //:gazelle` |
| Code sinh ra (sqlc, oapi-codegen) | Commit vào repo |
| Test cần Docker | `tags = ["requires-docker", "requires-network"]`, `size = "large"` |
| Image | Macro `com_tm_go_image` (`tools/rules/com_tm_container.bzl`), gazelle `map_kind` cho mọi `go_binary`. Sinh `<name>`, `<name>_image`, `<name>_docker` (tag `com.tm.go.<name>:v1.0.0`), `<name>_push` khi có `repository`. Base distroless pin digest (`gcr.io/distroless/static-debian12:nonroot`, binary tại `/app/<name>`). Target image `target_compatible_with` Linux → `bazel build //...` trên macOS bỏ qua (SKIPPED) image, vẫn build binary/test |
| Build image | `bazel run --config=linux-arm64 //path:<name>_docker` (Apple Silicon) · `--config=linux-amd64` (server x86). Ví dụ: `//tools/smoke:smoke_docker` → `com.tm.go.smoke:v1.0.0` |
| IDE / gopls | Dùng `go.mod` trực tiếp, không cần `GOPACKAGESDRIVER`. Code phải build được bằng cả `go` lẫn Bazel |

### Module trong core (modular monolith)

Core là **một** service nhưng chia **module theo nghiệp vụ** với ranh giới như thể sắp tách thành service riêng. Xem [ADR-0001](adr/0001-modular-monolith-core.md).

- Mỗi module chỉ lộ ra `Service` và kiểu dữ liệu công khai; phần còn lại không export.
- Module gọi nhau qua **interface khai báo ở phía dùng** (nhỏ, 1–3 method).
- Mỗi module **sở hữu bảng của mình**; không truy vấn bảng của module khác.
- Ngoại lệ có chủ đích: **dùng chung transaction** (`pgx.Tx` truyền qua interface) để luồng đặt vé là một transaction ACID.

Bên trong một module, code nằm chung package theo vertical slice:

```
internal/booking/
├── booking.go        # type Booking, Hold, Ticket, lỗi nghiệp vụ
├── pricing.go        # hàm thuần: tính giá — test không cần DB
├── service.go        # type Service + interface nó cần
├── store.go          # truy vấn PG (gọi sqlc)
├── http.go           # handler + Routes()
└── *_test.go
```

```go
// internal/booking/service.go
type debiter interface { // khai báo ở phía dùng
	Debit(ctx context.Context, tx pgx.Tx, userID uuid.UUID, amount money.VND, ref string) error
}

type Service struct {
	pool   *pgxpool.Pool
	wallet debiter
}

func NewService(pool *pgxpool.Pool, wallet debiter) *Service { // nhận interface, trả struct
	return &Service{pool: pool, wallet: wallet}
}
```

```go
// cmd/server/main.go — wiring thủ công, không DI framework
pool := postgres.MustConnect(ctx, cfg.DatabaseURL)
walletSvc := wallet.NewService(pool)
bookingSvc := booking.NewService(pool, walletSvc)
catalogSvc := catalog.NewService(pool)
router := httpx.NewRouter(catalog.Routes(catalogSvc), booking.Routes(bookingSvc), wallet.Routes(walletSvc))
```

### Go cho người từ Java

| Thói quen Java | Trong snaptix |
|---|---|
| `controller/`, `service/`, `repository/`, `model/` | Package theo nghiệp vụ: `booking/`, `wallet/` |
| `IFooRepository` + `FooRepositoryImpl` | Không tạo interface khi chưa có người dùng cần; không tiền tố `I`, hậu tố `Impl` |
| Interface cạnh implementation | Interface ở phía dùng |
| Dagger / Guice / wire / fx | Wiring tay trong `main.go` |
| `utils/`, `common/`, `helpers/` | Đặt tên theo chức năng: `money`, `httpx`, `postgres` |
| Package lồng sâu | Cấu trúc phẳng, dùng `internal/` để giới hạn |
| Getter/setter | Export field khi không có bất biến cần bảo vệ |
| Mock mọi repository | Test với PG thật (testcontainers); logic thuần test không cần DB |
| Exception | Error value, `errors.Is/As`, wrap có ngữ cảnh |
| `TxManager`, domain "sạch" không biết DB | Service dùng `postgres.WithTx` và `pgx.Tx` trực tiếp; chỉ tách **logic tính toán** thành hàm thuần |

---

## `com/tm/app` — Node.js + React + pnpm

```
com/tm/app/
├── package.json              # tên snaptix-app; script gốc build/test/lint/typecheck (pnpm --recursive --if-present), dev thêm --parallel
├── pnpm-workspace.yaml
├── pnpm-lock.yaml
├── api/
│   └── bff.openapi.yaml      # hợp đồng API công khai của BFF
├── apps/
│   ├── bff/                  # Node.js + Fastify
│   ├── web-client/           # React + Vite
│   └── web-admin/            # React + Vite
└── packages/
    ├── types/                # Zod schema dùng chung BFF ↔ web; type sinh từ OpenAPI
    ├── api-client/           # client gọi BFF
    ├── ui/                   # shadcn/ui components
    └── config/               # eslint, prettier, tsconfig, tailwind preset
```

| Việc | Công cụ |
|---|---|
| Dependency | pnpm workspace (`apps/*`, `packages/*`), một lockfile; `packageManager: pnpm@11.18.0`, Node ≥ 22 |
| Chạy script | Gốc: `pnpm build`/`test`/`lint`/`typecheck` (mọi package, thứ tự topo, bỏ qua package thiếu script), `pnpm dev` (song song); một app: `pnpm --filter <app> <script>`; app và mọi package nó phụ thuộc: `pnpm --filter "<app>..." <script>` |
| Dev BFF | `tsx watch` |
| Build BFF | `tsc --noEmit` (kiểm type) + `tsup` (esbuild) → `dist/` |
| Build web | Vite |
| Test | Vitest, Testing Library, Playwright |
| Image BFF | Dockerfile multi-stage + `pnpm deploy --filter bff --prod` |
| Sinh type từ OpenAPI | `openapi-typescript` từ `../server/api/core.openapi.yaml` và `api/bff.openapi.yaml`; commit kết quả |

### BFF (Fastify)

```
apps/bff/src/
├── server.ts                 # khởi tạo Fastify, graceful shutdown
├── plugins/                  # hạ tầng: mongo, redis, session, csrf, rate-limit, otel, error-handler
├── core-client/              # undici: timeout, retry, circuit breaker
└── routes/                   # vertical slice theo tài nguyên
    ├── auth/
    ├── trips/
    ├── holds/
    ├── bookings/             # route + schema + gọi core cho bookings
    ├── wallet/
    └── admin/
```

- **Plugin là đơn vị module**; dependency gắn bằng `fastify.decorate`, không dùng DI container.
- **Không dùng NestJS**: decorator và DI kiểu Spring che mất event loop, vòng đời request, stream — những thứ cần học.
- TypeScript strict, ESM, Node 22.
- Không tách `controllers/`, `services/`, `repositories/`.

### Web apps (React + Vite)

```
apps/web-client/src/
├── main.tsx
├── routes/                   # React Router: public (tìm chuyến), protected (checkout, ví, vé)
├── features/                 # search, seat-map, checkout, wallet, tickets
├── components/
└── lib/                      # query client, auth guard
```

`web-admin` có cấu trúc tương tự với các feature: catalog, schedules, fares, bookings, users, stats.

---

## Code dùng chung — quy tắc 3 tầng

| Tầng | Vị trí (Go) | Vị trí (TS) | Khi nào đưa vào |
|---|---|---|---|
| 1. Trong một feature | Trong package, không export | Trong `features/<x>` hoặc `routes/<x>` | Mặc định |
| 2. Trong một service/app | `services/core/internal/<tên>` | `apps/<app>/src/lib` | Feature thứ hai cần |
| 3. Giữa các service/app | `server/pkg/<tên>` | `app/packages/<tên>` | Service thứ hai cần **và** là hạ tầng hoặc hợp đồng |

**Không đưa vào tầng 3**: domain model (`Booking`, `Trip`...), logic nghiệp vụ, package tên `common`/`utils`/`helpers`/`shared`. `pkg/` không được import `services/`.

**Copy khi**: đoạn code ngắn, mỗi nơi cần hơi khác nhau (health handler, parse config). **Dùng chung khi**: logic đủ khó để hai bản copy lệch nhau thành bug (`WithTx` + retry) và API đã ổn định.

Kiểm tra cái giá của việc dùng chung: `bazel query 'rdeps(//..., //pkg/postgres)'`.

---

## CI

Một workflow `.github/workflows/ci.yml` (PR vào `main` và push lên `main`): job `changes` luôn chạy và tính vùng đổi, job `server` / `app` chỉ chạy khi vùng tương ứng đổi (`if`, không lọc bằng `paths:` — job skipped vẫn đạt khi làm required check). Logic nằm trong `scripts/ci-*.sh`, chạy lại cục bộ được (xem [local-setup](local-setup.md#build--kiểm-thử)).

| Thay đổi | Chạy |
|---|---|
| `com/tm/server/**` | `golangci-lint run ./...` (v2.14.0, `.golangci.yml`) + `bazel test` các target bị ảnh hưởng (`scripts/ci-server.sh`) |
| `com/tm/app/**` | `pnpm install --frozen-lockfile` → `pnpm --filter "...[origin/main]" --filter '!snaptix-app' --if-present run '/^(lint\|test\|build)$/'` → `git diff --exit-code pnpm-lock.yaml` (`scripts/ci-app.sh`, xem ghi chú dưới bảng) |
| `.github/workflows/**`, `scripts/ci-*` | Cả hai job (để chạy thử thay đổi CI) |
| `com/tm/server/api/**`, `com/tm/app/api/**` | Cả hai + sinh lại code và `git diff --exit-code` — *chưa có, thêm khi có codegen* |
| `com/tm/docs/**` | Kiểm tra link markdown — *chưa có* |

Base của diff: PR → merge-base với `origin/<nhánh đích>`; push lên `main` → `github.event.before` (base rỗng / 40 số 0 → chạy tất cả). Checkout `fetch-depth: 0`.

Target Bazel bị ảnh hưởng (`scripts/ci-affected.sh`, G14): file đổi → label (`//<pkg>:<file>`; file `BUILD` đổi hoặc file bị xoá → `//<pkg>:all`), lọc label không phải target (`bazel query --keep_going`), rồi `tests(rdeps(//..., set(...)))` bỏ tag `manual`. File toàn cục (`MODULE.bazel`, `MODULE.bazel.lock`, `go.mod`, `go.sum`, `.bazelrc`, `.bazelversion`, `BUILD.bazel` gốc, mọi `*.bzl`) → `//...`. Không có target nào → bỏ qua `bazel test` (tránh `No test targets were found`, exit 4).

App chạy **mọi** package (`--recursive --filter '!snaptix-app'`) khi `package.json` gốc, `pnpm-workspace.yaml`, `pnpm-lock.yaml` hoặc file CI đổi — filter `...[base]` + `!snaptix-app` khi đó không chọn package nào.

Ghi chú lệnh pnpm (đã kiểm ở P0-T01b):
- **Không** viết `pnpm --filter ... lint test build`: pnpm chỉ chạy script đầu tiên (`lint`) và coi `test build` là **đối số** của nó — lỗi bị che vì exit 0. Chạy nhiều script bằng regex: `run '/^(lint|test|build)$/'` (các script khớp chạy **đồng thời** trong cùng package, package vẫn theo thứ tự topo), hoặc gọi lần lượt từng lệnh nếu cần tuần tự.
- `...[origin/main]`: package có file thay đổi so với `origin/main` **cộng mọi package phụ thuộc vào nó** (dependents). File chưa được git track không tính là thay đổi.
- `--filter '!snaptix-app'`: loại package gốc (`com/tm/app/package.json`, tên `snaptix-app`) — khi `package.json` gốc đổi, root cũng bị chọn và script gốc sẽ tự chạy đệ quy lại cả workspace.
- `--if-present`: không lỗi `ERR_PNPM_RECURSIVE_RUN_NO_SCRIPT` khi package được chọn không có script khớp. Filter không khớp package nào → in `No projects matched the filters`, exit 0; thêm `--fail-if-no-match` nếu muốn exit 1.

---

## Quy ước chung

| Hạng mục | Quy ước |
|---|---|
| Tiền | `int64` (Go) / `bigint` (PG) / `string` trong JSON — đơn vị VND |
| Thời gian | Lưu `timestamptz` UTC; hiển thị theo `Asia/Ho_Chi_Minh` |
| ID | UUIDv7 (sắp xếp được theo thời gian) |
| Commit | Conventional Commits, ghi ID task/challenge: `feat(core): ... [P4-T05][P1]` |
| Migration | Chỉ thêm mới, không sửa migration đã merge |
| Test Go | Table-driven; integration test với testcontainers |
| Test TS | Vitest; E2E với Playwright |
