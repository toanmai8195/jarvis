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
