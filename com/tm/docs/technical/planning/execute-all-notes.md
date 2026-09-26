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
