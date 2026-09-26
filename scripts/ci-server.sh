#!/usr/bin/env bash
# ci-server.sh — toàn bộ job `server` của CI (.github/workflows/ci.yml):
#   1. golangci-lint trên toàn module com/tm/server (cấu hình .golangci.yml)
#   2. bazel test các target bị ảnh hưởng (scripts/ci-affected.sh)
#
# Dùng:  bash scripts/ci-server.sh BASE [HEAD]      (xem ci-lib.sh về BASE)
#   Chạy lại job CI trên máy dev: bash scripts/ci-server.sh origin/main
# Biến môi trường: GOLANGCI_LINT (mặc định golangci-lint), BAZEL (mặc định bazel).
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

GOLANGCI_LINT=${GOLANGCI_LINT:-golangci-lint}
BAZEL=${BAZEL:-bazel}

main() {
  local base=${1-} head=${2:-HEAD}
  local root
  root=$(git rev-parse --show-toplevel)
  cd "$root/com/tm/server"

  echo "==> golangci-lint ($("$GOLANGCI_LINT" --version))"
  "$GOLANGCI_LINT" run ./...

  echo "==> target Bazel bị ảnh hưởng (base: ${base:-<rỗng>})"
  local out targets=() t
  out=$(BAZEL="$BAZEL" bash "$SCRIPT_DIR/ci-affected.sh" "$base" "$head")
  while IFS= read -r t; do
    [[ -n "$t" ]] && targets+=("$t")
  done <<<"$out"

  if [[ ${#targets[@]} -eq 0 ]]; then
    echo "==> không có target Bazel bị ảnh hưởng — bỏ qua bazel test"
    return 0
  fi
  printf '    %s\n' "${targets[@]}"

  echo "==> bazel test ($("$BAZEL" --version))"
  "$BAZEL" test -- "${targets[@]}"
}

main "$@"
