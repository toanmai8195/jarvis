#!/usr/bin/env bash
# ci-app.sh — toàn bộ job `app` của CI (.github/workflows/ci.yml), chạy trong
# com/tm/app:
#   1. pnpm install --frozen-lockfile   (lockfile lệch → fail ngay)
#   2. lint/test/build các package bị ảnh hưởng (lệnh của project-structure.md
#      mục CI; KHÔNG liệt kê nhiều tên script liền nhau sau lệnh pnpm — pnpm
#      chỉ chạy script đầu tiên, các tên sau thành đối số của nó)
#   3. git diff --exit-code pnpm-lock.yaml   (script không được viết lại lockfile)
#
# Dùng:  bash scripts/ci-app.sh BASE [HEAD]      (xem ci-lib.sh về BASE)
#   Chạy lại job CI trên máy dev: bash scripts/ci-app.sh origin/main
# Chạy MỌI package (trừ root `snaptix-app`) khi: không có base, file gốc của
# workspace đổi (package.json, pnpm-workspace.yaml, pnpm-lock.yaml) hoặc file
# của chính CI đổi — vì filter `...[base]` + `!snaptix-app` khi đó không chọn
# package nào.
# Biến môi trường: PNPM (mặc định pnpm).
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=scripts/ci-lib.sh
source "$SCRIPT_DIR/ci-lib.sh"

PNPM=${PNPM:-pnpm}
APP_PREFIX=com/tm/app/

# run_all MERGE_BASE HEAD — exit 0 khi cần chạy mọi package.
run_all() {
  local files f
  files=$(ci_changed_files "$1" "$2")
  while IFS= read -r f; do
    case "$f" in
      "${APP_PREFIX}package.json" | "${APP_PREFIX}pnpm-workspace.yaml" | "${APP_PREFIX}pnpm-lock.yaml")
        echo "==> file gốc workspace đổi: $f → chạy mọi package" >&2
        return 0
        ;;
    esac
    if ci_is_ci_file "$f"; then
      echo "==> file CI đổi: $f → chạy mọi package" >&2
      return 0
    fi
  done <<<"$files"
  return 1
}

main() {
  local base=${1-} head=${2:-HEAD}
  local root mb
  root=$(git rev-parse --show-toplevel)
  cd "$root/com/tm/app"

  echo "==> pnpm install --frozen-lockfile (pnpm $("$PNPM" --version))"
  "$PNPM" install --frozen-lockfile

  if ! mb=$(ci_merge_base "$base" "$head"); then
    echo "==> không có base dùng được ('$base') → chạy mọi package" >&2
    "$PNPM" --recursive --filter '!snaptix-app' --if-present run '/^(lint|test|build)$/'
  elif run_all "$mb" "$head"; then
    "$PNPM" --recursive --filter '!snaptix-app' --if-present run '/^(lint|test|build)$/'
  else
    echo "==> package thay đổi so với $mb + package phụ thuộc"
    "$PNPM" --filter "...[$mb]" --filter '!snaptix-app' --if-present run '/^(lint|test|build)$/'
  fi

  echo "==> git diff --exit-code pnpm-lock.yaml"
  git diff --exit-code -- pnpm-lock.yaml
}

main "$@"
