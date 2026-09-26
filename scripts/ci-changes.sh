#!/usr/bin/env bash
# ci-changes.sh — tính vùng thay đổi cho job `changes` của CI
# (.github/workflows/ci.yml). In đúng định dạng của $GITHUB_OUTPUT:
#   server=true|false   có thay đổi trong com/tm/server/**
#   app=true|false      có thay đổi trong com/tm/app/**
#
# Dùng:  bash scripts/ci-changes.sh BASE [HEAD]      (chạy trong repo git)
#   BASE rỗng / 40 số 0 / không tìm được merge-base → cả hai `true`.
#   File của chính CI (.github/workflows/**, scripts/ci-*) đổi → cả hai `true`.
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=scripts/ci-lib.sh
source "$SCRIPT_DIR/ci-lib.sh"

main() {
  local base=${1-} head=${2:-HEAD}
  local server=false app=false mb files f

  if ! mb=$(ci_merge_base "$base" "$head"); then
    echo "ci-changes: không có base dùng được ('$base') → chạy tất cả" >&2
    printf 'server=true\napp=true\n'
    return 0
  fi

  files=$(ci_changed_files "$mb" "$head")
  while IFS= read -r f; do
    [[ -z "$f" ]] && continue
    case "$f" in
      com/tm/server/*) server=true ;;
      com/tm/app/*) app=true ;;
    esac
    if ci_is_ci_file "$f"; then
      server=true
      app=true
    fi
  done <<<"$files"

  printf 'server=%s\napp=%s\n' "$server" "$app"
}

main "$@"
