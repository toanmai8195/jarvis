#!/usr/bin/env bash
# check-structure.sh — kiểm tra khung thư mục monorepo snaptix theo
# com/tm/docs/technical/project-structure.md.
#
# Dùng:  bash scripts/check-structure.sh [ROOT]
#   ROOT  thư mục gốc repo cần kiểm tra (mặc định: gốc git của cwd, hoặc cwd).
#
# Exit 0 khi hợp lệ. Exit 1 khi thiếu thư mục bắt buộc hoặc có thư mục cấm
# (tên kiểu Java / chung chung); mỗi lỗi in một dòng ra stderr.
# Exit 2 khi ROOT không phải thư mục.
set -euo pipefail

# Thư mục bắt buộc (tương đối với ROOT).
REQUIRED_DIRS=(
  deploy
  loadtest
  loadtest/results
  scripts
  com/tm/server
  com/tm/server/api
  com/tm/server/pkg
  com/tm/server/services
  com/tm/app
  com/tm/app/api
  com/tm/app/apps
  com/tm/app/packages
  com/tm/docs
)

# Nơi quét thư mục cấm.
SCAN_DIRS=(com/tm/server com/tm/app deploy loadtest scripts)

# Tên thư mục cấm: thói quen Java (project-structure "Go cho người từ Java")
# và tên chung chung không được đưa vào tầng 3 ("quy tắc 3 tầng").
FORBIDDEN_NAMES=(service repository controller model utils util common helpers shared)

resolve_root() {
  if [[ $# -ge 1 && -n "$1" ]]; then
    printf '%s\n' "$1"
  elif top=$(git rev-parse --show-toplevel 2>/dev/null); then
    printf '%s\n' "$top"
  else
    pwd
  fi
}

main() {
  local root
  root=$(resolve_root "$@")
  if [[ ! -d "$root" ]]; then
    echo "check-structure: ROOT không phải thư mục: $root" >&2
    return 2
  fi

  local errors=0 d

  for d in "${REQUIRED_DIRS[@]}"; do
    if [[ ! -d "$root/$d" ]]; then
      echo "MISSING dir: $d" >&2
      errors=$((errors + 1))
    fi
  done

  # Biểu thức find: -name a -o -name b ...
  local name_expr=() n
  for n in "${FORBIDDEN_NAMES[@]}"; do
    [[ ${#name_expr[@]} -gt 0 ]] && name_expr+=(-o)
    name_expr+=(-name "$n")
  done

  local scan hit
  for scan in "${SCAN_DIRS[@]}"; do
    [[ -d "$root/$scan" ]] || continue
    # Bỏ qua dependency / build output: tên thư mục bên trong không do ta đặt.
    while IFS= read -r hit; do
      [[ -z "$hit" ]] && continue
      echo "FORBIDDEN dir: ${hit#"$root"/}" >&2
      errors=$((errors + 1))
    done < <(find "$root/$scan" \
      \( -name node_modules -o -name 'bazel-*' -o -name dist -o -name .git \) -prune -o \
      -type d \( "${name_expr[@]}" \) -print | LC_ALL=C sort)
  done

  if [[ $errors -gt 0 ]]; then
    echo "check-structure: $errors lỗi (xem com/tm/docs/technical/project-structure.md)" >&2
    return 1
  fi
  echo "check-structure: OK ($root)"
}

main "$@"
