#!/usr/bin/env bash
# ci-server_test.sh — test cho scripts/ci-server.sh (luồng của job `server`).
# Mỗi ca dựng một repo git trong thư mục tạm, không đụng repo thật.
# golangci-lint và Bazel được thay bằng script giả (biến GOLANGCI_LINT, BAZEL)
# ghi lại lời gọi; hành vi với công cụ thật được kiểm ở P0-T04-TC08, TC17,
# TC19, TC20.
#
# Dùng:  bash scripts/ci-server_test.sh
set -uo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SUT="$SCRIPT_DIR/ci-server.sh"

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/ci-server-test.XXXXXX")
trap 'rm -rf "$TMP_ROOT"' EXIT

pass=0
fail=0

# golangci-lint giả: exit $FAKE_LINT_RC.
FAKE_LINT="$TMP_ROOT/fake-golangci-lint"
cat >"$FAKE_LINT" <<'EOF'
#!/usr/bin/env bash
echo "golangci-lint $*" >>"$FAKE_LOG"
[[ "$1" == "--version" ]] && { echo "golangci-lint has version fake"; exit 0; }
exit "${FAKE_LINT_RC:-0}"
EOF
# Bazel giả: `query` in $FAKE_TARGETS (bước lọc lẫn rdeps), `test` exit $FAKE_TEST_RC.
FAKE_BAZEL="$TMP_ROOT/fake-bazel"
cat >"$FAKE_BAZEL" <<'EOF'
#!/usr/bin/env bash
echo "bazel $*" >>"$FAKE_LOG"
case "$1" in
  --version) echo "bazel 8.7.0" ;;
  query) [[ -n "${FAKE_TARGETS:-}" ]] && printf '%s\n' "$FAKE_TARGETS"; exit 0 ;;
  test) exit "${FAKE_TEST_RC:-0}" ;;
esac
EOF
chmod +x "$FAKE_LINT" "$FAKE_BAZEL"

g() { git -C "$REPO" "$@"; }

# new_repo NAME FILE — repo có com/tm/server, commit base, rồi commit sửa FILE
# (tương đối với com/tm/server).
new_repo() {
  REPO="$TMP_ROOT/$1"
  mkdir -p "$REPO/com/tm/server/pkg/a"
  g init -q -b main
  g config user.email t@example.com
  g config user.name t
  g config commit.gpgsign false
  echo "# root" >"$REPO/com/tm/server/BUILD.bazel"
  echo "# a" >"$REPO/com/tm/server/pkg/a/BUILD.bazel"
  echo a >"$REPO/com/tm/server/pkg/a/a.go"
  g add -A
  g commit -qm base
  echo "$RANDOM" >>"$REPO/com/tm/server/$2"
  g add -A
  g commit -qm change
}

# run_case NAME WANT_EXIT OUT_PATTERN LOG_PATTERN NOT_LOG_PATTERN BASE
#   Pattern là ERE; rỗng = không kiểm.
run_case() {
  local name=$1 want_exit=$2 out_re=$3 log_re=$4 not_log_re=$5 base=$6
  export FAKE_LOG="$TMP_ROOT/$name.log"
  : >"$FAKE_LOG"
  local out got_exit log
  out=$(cd "$REPO" && GOLANGCI_LINT="$FAKE_LINT" BAZEL="$FAKE_BAZEL" bash "$SUT" "$base" HEAD 2>&1)
  got_exit=$?
  log=$(tr "\n" " " <"$FAKE_LOG")  # một dòng: pattern kiểm được thứ tự lời gọi

  local ok=1 why=""
  if [[ "$want_exit" == "nonzero" ]]; then
    [[ $got_exit -ne 0 ]] || { ok=0; why="exit=0, muốn ≠0"; }
  else
    [[ $got_exit -eq $want_exit ]] || { ok=0; why="exit=$got_exit, muốn $want_exit"; }
  fi
  [[ $ok -eq 0 || -z "$out_re" ]] || grep -qE -- "$out_re" <<<"$out" ||
    { ok=0; why="output không khớp /$out_re/: $out"; }
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
    echo "FAIL $name — $why"
    fail=$((fail + 1))
  fi
  unset FAKE_LINT_RC FAKE_TARGETS FAKE_TEST_RC
}

# Lint chạy trong com/tm/server trên toàn module, trước bazel.
new_repo lint_then_test pkg/a/a.go
FAKE_TARGETS="//pkg/a:a_test" run_case lint_then_test 0 "" \
  'golangci-lint run \./\.\..*bazel test -- //pkg/a:a_test' "" main~1
new_repo lint_cwd pkg/a/a.go
(cd "$REPO/com/tm/server" && pwd -P) >"$TMP_ROOT/lint_cwd.want"
cat >"$TMP_ROOT/pwd-lint" <<EOF
#!/usr/bin/env bash
[[ "\$1" == "run" ]] && pwd -P >"$TMP_ROOT/lint_cwd.got"
exit 0
EOF
chmod +x "$TMP_ROOT/pwd-lint"
(cd "$REPO" && GOLANGCI_LINT="$TMP_ROOT/pwd-lint" BAZEL="$FAKE_BAZEL" FAKE_LOG=/dev/null \
  bash "$SUT" main~1 HEAD >/dev/null 2>&1)
if diff -q "$TMP_ROOT/lint_cwd.want" "$TMP_ROOT/lint_cwd.got" >/dev/null 2>&1; then
  echo "PASS lint_cwd"
  pass=$((pass + 1))
else
  echo "FAIL lint_cwd — golangci-lint không chạy trong com/tm/server"
  fail=$((fail + 1))
fi

# Lint fail → job fail, không chạy bazel.
new_repo lint_fail pkg/a/a.go
FAKE_LINT_RC=1 FAKE_TARGETS="//pkg/a:a_test" run_case lint_fail nonzero "" "" 'bazel' main~1

# Không có target → bỏ qua bazel test, exit 0, thông báo rõ.
new_repo no_targets pkg/a/a.go
run_case no_targets 0 "không có target Bazel bị ảnh hưởng" "" 'bazel test' main~1

# Nhiều target → một lệnh bazel test.
new_repo many_targets pkg/a/a.go
FAKE_TARGETS=$'//pkg/a:a_test\n//pkg/b:b_test' run_case many_targets 0 "" \
  'bazel test -- //pkg/a:a_test //pkg/b:b_test' "" main~1

# //... → bazel test //... (không truy vấn Bazel khi file toàn cục đổi).
new_repo all_targets MODULE.bazel
run_case all_targets 0 "" 'bazel test -- //\.\.\.' 'bazel query' main~1

# Test fail → job fail.
new_repo test_fail pkg/a/a.go
FAKE_TARGETS="//pkg/a:a_test" FAKE_TEST_RC=3 run_case test_fail nonzero "" "bazel test" "" main~1

echo "ci-server_test: $pass pass, $fail fail"
[[ $fail -eq 0 ]]
