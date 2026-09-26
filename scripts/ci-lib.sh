# shellcheck shell=bash
# ci-lib.sh — hàm dùng chung cho các script CI (ci-changes.sh, ci-affected.sh,
# ci-app.sh). Chỉ để `source`, không chạy trực tiếp.
#
# Quy ước tham số của mọi script CI:
#   BASE  commit/ref để so sánh. PR: `origin/<nhánh đích>`; push lên main:
#         `github.event.before`. Rỗng hoặc 40 số 0 (push tạo nhánh / lần push
#         đầu) → không có base, script coi như MỌI THỨ thay đổi.
#   HEAD  commit đang kiểm (mặc định HEAD).
# Diff luôn tính từ merge-base(BASE, HEAD) — giống `git diff BASE...HEAD`:
# thay đổi phía BASE sau điểm tách nhánh không được tính.

# ci_merge_base BASE HEAD — in merge-base. Exit 1 khi không có base dùng được
# (rỗng, toàn số 0, commit không tồn tại — ví dụ `before` sau force push).
ci_merge_base() {
  local base=${1-} head=${2:-HEAD}
  if [[ -z "$base" || "$base" =~ ^0+$ ]]; then
    return 1
  fi
  git merge-base "$base" "$head" 2>/dev/null
}

# ci_changed_files MERGE_BASE HEAD — mọi đường dẫn (tính từ gốc repo) bị thêm,
# sửa, xoá giữa MERGE_BASE và HEAD. `--no-renames`: file đổi tên in cả đường
# dẫn cũ lẫn mới (mặc định git chỉ in đường dẫn mới).
ci_changed_files() {
  git diff --name-only --no-renames "$1" "${2:-HEAD}" --
}

# ci_is_ci_file PATH — file của chính CI: đổi thì chạy thử mọi job.
ci_is_ci_file() {
  case "$1" in
    .github/workflows/* | scripts/ci-*) return 0 ;;
  esac
  return 1
}
