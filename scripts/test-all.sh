#!/usr/bin/env bash
# test-all.sh — `make test`: chạy bộ test hiện có của repo, không cần stack
# Docker (không bật stack, không sửa file được track).
#
#   scripts  mọi scripts/*_test.sh, bash scripts/check-structure.sh,
#            bash scripts/check-compose.sh (chỉ đọc config)
#   server   trong com/tm/server: go vet ./..., go test -race ./..., bazel test //...
#   app      trong com/tm/app: pnpm install --frozen-lockfile, rồi
#            pnpm lint, pnpm typecheck, pnpm test, pnpm build (đệ quy, --if-present)
#
# Dùng:  bash scripts/test-all.sh [scripts|server|app]...   (mặc định: cả ba)
#
# Mọi bộ đều chạy, kể cả khi bộ trước lỗi; exit 1 nếu có bước lỗi (in danh
# sách ở cuối). Riêng app: `pnpm install --frozen-lockfile` lỗi thì bỏ các bước
# còn lại của app — pnpm 11 tự `install` (không frozen, sửa lockfile) trước khi
# chạy script nếu deps lệch.
# golangci-lint và `bazel run //:gazelle` không chạy ở đây (gazelle sửa BUILD):
# xem local-setup.md mục "Build & kiểm thử".
set -uo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
ROOT=$(cd "$SCRIPT_DIR/.." && pwd)

failed=()

# run SUITE DIR CMD... — chạy CMD trong ROOT/DIR, ghi lại nếu lỗi. Exit = exit của CMD.
run() {
  local suite=$1 dir=$2 rc=0
  shift 2
  echo "==> [$suite] $dir: $*"
  (cd "$ROOT/$dir" && "$@") || rc=$?
  if [[ $rc -ne 0 ]]; then
    echo "==> [$suite] FAIL (exit $rc): $*"
    failed+=("[$suite] $dir: $*")
  fi
  return "$rc"
}

suite_scripts() {
  local f found=0
  for f in "$ROOT"/scripts/*_test.sh; do
    [[ -e "$f" ]] || continue
    found=1
    run scripts . bash "scripts/${f##*/}"
  done
  [[ $found -eq 1 ]] || echo "==> [scripts] không có scripts/*_test.sh"
  run scripts . bash scripts/check-structure.sh
  run scripts . bash scripts/check-compose.sh
  return 0
}

suite_server() {
  run server com/tm/server go vet ./...
  run server com/tm/server go test -race ./...
  run server com/tm/server bazel test //...
  return 0
}

suite_app() {
  run app com/tm/app pnpm install --frozen-lockfile || {
    echo "==> [app] bỏ qua lint/typecheck/test/build vì install lỗi"
    return 0
  }
  run app com/tm/app pnpm lint
  run app com/tm/app pnpm typecheck
  run app com/tm/app pnpm test
  run app com/tm/app pnpm build
  return 0
}

main() {
  local suites=("$@") s
  [[ ${#suites[@]} -gt 0 ]] || suites=(scripts server app)
  for s in "${suites[@]}"; do
    case "$s" in
      scripts | server | app) ;;
      *)
        echo "test-all: bộ không hợp lệ: $s (chọn: scripts server app)" >&2
        return 2
        ;;
    esac
  done

  for s in "${suites[@]}"; do
    "suite_$s"
  done

  echo
  if [[ ${#failed[@]} -gt 0 ]]; then
    echo "test-all: FAIL — ${#failed[@]} bước lỗi:"
    printf '  %s\n' "${failed[@]}"
    return 1
  fi
  echo "test-all: OK (${suites[*]})"
}

main "$@"
