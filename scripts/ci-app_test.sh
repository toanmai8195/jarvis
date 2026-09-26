#!/usr/bin/env bash
# ci-app_test.sh — test cho scripts/ci-app.sh (luồng của job `app`).
# Mỗi ca dựng một repo git trong thư mục tạm, không đụng repo thật. pnpm được
# thay bằng script giả (biến PNPM) ghi lại lời gọi; hành vi với pnpm thật được
# kiểm ở P0-T04-TC21..TC24.
#
# Dùng:  bash scripts/ci-app_test.sh
set -uo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SUT="$SCRIPT_DIR/ci-app.sh"

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/ci-app-test.XXXXXX")
trap 'rm -rf "$TMP_ROOT"' EXIT

pass=0
fail=0

# pnpm giả: ghi "<args>" (mỗi lời gọi một dòng) vào $FAKE_LOG.
#   install → exit $FAKE_INSTALL_RC
#   run     → exit $FAKE_RUN_RC; FAKE_TOUCH_LOCK=1 thì viết lại pnpm-lock.yaml.
FAKE_PNPM="$TMP_ROOT/fake-pnpm"
cat >"$FAKE_PNPM" <<'EOF'
#!/usr/bin/env bash
[[ "$1" == "--version" ]] && { echo "11.18.0"; exit 0; }
printf '%s\n' "$*" >>"$FAKE_LOG"
if [[ "$1" == "install" ]]; then exit "${FAKE_INSTALL_RC:-0}"; fi
[[ "${FAKE_TOUCH_LOCK:-}" == 1 ]] && echo "changed" >>pnpm-lock.yaml
exit "${FAKE_RUN_RC:-0}"
EOF
chmod +x "$FAKE_PNPM"

g() { git -C "$REPO" "$@"; }

# new_repo NAME FILE... — commit base, rồi commit sửa từng FILE (tương đối
# với gốc repo).
new_repo() {
  REPO="$TMP_ROOT/$1"
  shift
  local a="$REPO/com/tm/app" f
  mkdir -p "$a/apps/x" "$REPO/.github/workflows" "$REPO/scripts"
  g init -q -b main
  g config user.email t@example.com
  g config user.name t
  g config commit.gpgsign false
  for f in package.json pnpm-workspace.yaml pnpm-lock.yaml apps/x/package.json; do
    echo "# $f" >"$a/$f"
  done
  echo w >"$REPO/.github/workflows/ci.yml"
  echo s >"$REPO/scripts/ci-app.sh"
  echo r >"$REPO/README.md"
  g add -A
  g commit -qm base
  for f in "$@"; do
    echo "$RANDOM" >>"$REPO/$f"
  done
  g add -A
  g commit -qm change
}

FILTERED="--filter ...\[[0-9a-f]{40}\] --filter !snaptix-app --if-present run /\^\(lint\|test\|build\)\\$/"
ALL="--recursive --filter !snaptix-app --if-present run /\^\(lint\|test\|build\)\\$/"

# run_case NAME WANT_EXIT LOG_PATTERN NOT_LOG_PATTERN BASE
#   LOG_PATTERN: ERE khớp log pnpm giả (các lời gọi nối bằng " ; ").
run_case() {
  local name=$1 want_exit=$2 log_re=$3 not_log_re=$4 base=$5
  export FAKE_LOG="$TMP_ROOT/$name.log"
  : >"$FAKE_LOG"
  local out got_exit log
  out=$(cd "$REPO" && PNPM="$FAKE_PNPM" bash "$SUT" "$base" HEAD 2>&1)
  got_exit=$?
  log=$(awk 'NR>1{printf " ; "}{printf "%s",$0}' "$FAKE_LOG")

  local ok=1 why=""
  if [[ "$want_exit" == "nonzero" ]]; then
    [[ $got_exit -ne 0 ]] || { ok=0; why="exit=0, muốn ≠0"; }
  else
    [[ $got_exit -eq $want_exit ]] || { ok=0; why="exit=$got_exit, muốn $want_exit"; }
  fi
  [[ $ok -eq 0 || -z "$log_re" ]] || grep -qE -- "$log_re" <<<"$log" ||
    { ok=0; why="log không khớp /$log_re/: $log"; }
  if [[ $ok -eq 1 && -n "$not_log_re" ]] && grep -qE -- "$not_log_re" <<<"$log"; then
    ok=0
    why="log không được khớp /$not_log_re/: $log"
  fi

  if [[ $ok -eq 1 ]]; then
    echo "PASS $name"
    pass=$((pass + 1))
  else
    echo "FAIL $name — $why (output: $out)"
    fail=$((fail + 1))
  fi
  unset FAKE_INSTALL_RC FAKE_RUN_RC FAKE_TOUCH_LOCK
}

# Package thay đổi → install frozen trước, rồi lệnh filter theo merge-base.
new_repo package_change com/tm/app/apps/x/package.json
run_case package_change 0 "^install --frozen-lockfile ; $FILTERED$" "" main~1
# File gốc workspace / file CI đổi → chạy mọi package (trừ root).
for f in com/tm/app/package.json com/tm/app/pnpm-workspace.yaml com/tm/app/pnpm-lock.yaml \
  .github/workflows/ci.yml scripts/ci-app.sh; do
  n="all_${f//[^A-Za-z]/_}"
  new_repo "$n" "$f"
  run_case "$n" 0 "^install --frozen-lockfile ; $ALL$" "" main~1
done
# Không có base → chạy mọi package.
new_repo empty_base README.md
run_case empty_base 0 "$ALL" "" ""
new_repo zero_base README.md
run_case zero_base 0 "$ALL" "" 0000000000000000000000000000000000000000
# Không bao giờ dùng dạng sai: nhiều tên script liền nhau (pnpm chỉ chạy tên đầu).
new_repo no_wrong_form com/tm/app/apps/x/package.json
run_case no_wrong_form 0 "" "lint[ ]test[ ]build" main~1
# Install lỗi (lockfile lệch) → fail, không chạy script.
new_repo install_fail com/tm/app/apps/x/package.json
FAKE_INSTALL_RC=1 run_case install_fail nonzero "^install --frozen-lockfile$" "run" main~1
# Script lỗi → fail.
new_repo run_fail com/tm/app/apps/x/package.json
FAKE_RUN_RC=1 run_case run_fail nonzero "run" "" main~1
# Script viết lại lockfile → git diff --exit-code làm job fail.
new_repo lock_rewritten com/tm/app/apps/x/package.json
FAKE_TOUCH_LOCK=1 run_case lock_rewritten nonzero "run" "" main~1

echo "ci-app_test: $pass pass, $fail fail"
[[ $fail -eq 0 ]]
