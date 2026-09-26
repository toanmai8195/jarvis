#!/usr/bin/env bash
# ci-affected.sh — in các target Bazel TEST bị ảnh hưởng bởi thay đổi trong
# com/tm/server giữa merge-base(BASE, HEAD) và HEAD, mỗi dòng một label.
#   `//...`      chạy tất cả (file toàn cục của workspace đổi, hoặc không có base)
#   (không in)   không có target test nào bị ảnh hưởng
#
# Dùng:  bash scripts/ci-affected.sh BASE [HEAD]
#   Chạy trong repo git; HEAD phải là commit đang checkout (Bazel đọc BUILD
#   trên working tree). Biến môi trường BAZEL (mặc định `bazel`, tức Bazelisk
#   đọc com/tm/server/.bazelversion).
#
# Cách tính (G14):
#   1. File đổi → label ứng viên: file còn tồn tại → `//<pkg>:<file>` (pkg là
#      thư mục gần nhất có BUILD.bazel); file BUILD đổi hoặc file bị xoá →
#      `//<pkg>:all` của package gần nhất còn tồn tại.
#   2. `bazel query --keep_going set(...)` lọc bỏ ứng viên không phải target
#      (vd db/**/*.sql nằm trong package gốc nhưng không được khai báo).
#   3. `tests(rdeps(//..., set(...)))` bỏ test gắn tag `manual`.
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=scripts/ci-lib.sh
source "$SCRIPT_DIR/ci-lib.sh"

BAZEL=${BAZEL:-bazel}
SERVER_PREFIX=com/tm/server/

# File mà mọi target phụ thuộc vào (đường dẫn tính từ com/tm/server).
is_global_file() {
  case "$1" in
    MODULE.bazel | MODULE.bazel.lock | go.mod | go.sum | .bazelrc | .bazelversion | \
      .bazelignore | BUILD.bazel | BUILD | REPO.bazel | WORKSPACE | WORKSPACE.bazel | *.bzl)
      return 0
      ;;
  esac
  return 1
}

is_build_file() {
  case "${1##*/}" in
    BUILD.bazel | BUILD) return 0 ;;
  esac
  return 1
}

# package_of SERVER_DIR DIR — thư mục package Bazel gần nhất (tính từ
# com/tm/server) chứa DIR; "." là package gốc.
package_of() {
  local server=$1 dir=$2
  while [[ "$dir" != "." ]]; do
    if [[ -f "$server/$dir/BUILD.bazel" || -f "$server/$dir/BUILD" ]]; then
      printf '%s\n' "$dir"
      return 0
    fi
    dir=$(dirname "$dir")
  done
  printf '.\n'
}

# candidate_label SERVER_DIR REL — label ứng viên cho một file đã đổi.
candidate_label() {
  local server=$1 rel=$2 pkg label_pkg
  pkg=$(package_of "$server" "$(dirname "$rel")")
  if [[ "$pkg" == "." ]]; then label_pkg="//"; else label_pkg="//$pkg"; fi

  if is_build_file "$rel" || [[ ! -e "$server/$rel" ]]; then
    printf '%s:all\n' "$label_pkg"
  elif [[ "$pkg" == "." ]]; then
    printf '//:%s\n' "$rel"
  else
    printf '%s:%s\n' "$label_pkg" "${rel#"$pkg"/}"
  fi
}

# query_set LABEL... — biểu thức `set("a" "b" ...)` cho bazel query.
query_set() {
  local out="set(" l
  for l in "$@"; do out+="\"$l\" "; done
  printf '%s)\n' "$out"
}

main() {
  local base=${1-} head=${2:-HEAD}
  local root server mb files f rel
  root=$(git rev-parse --show-toplevel)
  server="$root/com/tm/server"

  if ! mb=$(ci_merge_base "$base" "$head"); then
    echo "ci-affected: không có base dùng được ('$base') → //..." >&2
    echo "//..."
    return 0
  fi

  files=$(ci_changed_files "$mb" "$head")
  local labels=""
  while IFS= read -r f; do
    [[ "$f" == "$SERVER_PREFIX"* ]] || continue
    rel=${f#"$SERVER_PREFIX"}
    if is_global_file "$rel"; then
      echo "ci-affected: file toàn cục $f đổi → //..." >&2
      echo "//..."
      return 0
    fi
    labels+="$(candidate_label "$server" "$rel")"$'\n'
  done <<<"$files"

  # Bỏ trùng (vd nhiều file bị xoá cùng package → cùng `//<pkg>:all`).
  local candidates=()
  while IFS= read -r f; do
    [[ -n "$f" ]] && candidates+=("$f")
  done < <(printf '%s' "$labels" | LC_ALL=C sort -u)

  if [[ ${#candidates[@]} -eq 0 ]]; then
    echo "ci-affected: không có file nào đổi trong $SERVER_PREFIX" >&2
    return 0
  fi

  cd "$server"
  # Bước 2: giữ ứng viên là target thật. --keep_going: "no such target" /
  # "no such package" không làm hỏng cả truy vấn (exit 3 = kết quả một phần).
  local existing rc=0
  existing=$("$BAZEL" query --keep_going --output=label \
    "$(query_set "${candidates[@]}")" 2>/dev/null) || rc=$?
  if [[ $rc -ne 0 && $rc -ne 3 ]]; then
    echo "ci-affected: bazel query lỗi (exit $rc)" >&2
    return "$rc"
  fi

  local targets=()
  while IFS= read -r f; do
    [[ -n "$f" ]] && targets+=("$f")
  done <<<"$existing"
  if [[ ${#targets[@]} -eq 0 ]]; then
    echo "ci-affected: file đổi không thuộc target Bazel nào" >&2
    return 0
  fi

  # Bước 3: không --keep_going — lỗi BUILD thật phải làm job fail.
  local set_expr
  set_expr=$(query_set "${targets[@]}")
  "$BAZEL" query --output=label \
    "tests(rdeps(//..., $set_expr)) except attr(\"tags\", \"\\bmanual\\b\", //...)" |
    LC_ALL=C sort -u
}

main "$@"
