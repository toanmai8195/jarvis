#!/usr/bin/env bash
# test-all_test.sh — test cho scripts/test-all.sh (`make test`).
# Mỗi ca dựng một "repo" giả trong thư mục tạm: bản sao test-all.sh, các
# scripts/*_test.sh + check-*.sh giả, thư mục com/tm/server, com/tm/app.
# go, bazel, pnpm là script giả đặt đầu PATH. Mọi lệnh giả ghi
# "<thư mục tương đối> <lệnh>" vào $FAKE_LOG và exit 1 khi lệnh khớp ERE
# $FAIL_ON. Chạy với công cụ thật được kiểm ở P0-T05-TC16..TC18.
#
# Dùng:  bash scripts/test-all_test.sh
set -uo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SUT="$SCRIPT_DIR/test-all.sh"

# pwd -P: đường dẫn thật (TMPDIR của macOS có `//`, /var → /private/var), để so
# được với `pwd -P` trong lệnh giả.
TMP_ROOT=$(cd "$(mktemp -d "${TMPDIR:-/tmp}/test-all-test.XXXXXX")" && pwd -P)
trap 'rm -rf "$TMP_ROOT"' EXIT

pass=0
fail=0

# Lệnh giả dùng chung: ghi log rồi exit theo $FAIL_ON (biến mở rộng lúc lệnh giả chạy).
# shellcheck disable=SC2016
FAKE_BODY='
rel=$(pwd -P); rel=${rel#"$FAKE_ROOT"}; rel=${rel#/}; rel=${rel:-.}
line="$(basename "$0")${*:+ $*}"
echo "$rel $line" >>"$FAKE_LOG"
if [[ -n "${FAIL_ON:-}" ]] && grep -qE -- "$FAIL_ON" <<<"$line"; then echo "fake-fail: $line"; exit 1; fi
exit 0'

BIN="$TMP_ROOT/bin"
mkdir -p "$BIN"
for t in go bazel pnpm; do
  printf '#!/usr/bin/env bash%s\n' "$FAKE_BODY" >"$BIN/$t"
  chmod +x "$BIN/$t"
done

# new_root NAME — repo giả; script trong scripts/ ghi log như lệnh giả (qua `bash <file>`).
new_root() {
  FAKE_ROOT="$TMP_ROOT/$1"
  mkdir -p "$FAKE_ROOT/scripts" "$FAKE_ROOT/com/tm/server" "$FAKE_ROOT/com/tm/app"
  cp "$SUT" "$FAKE_ROOT/scripts/test-all.sh"
  local f
  for f in a_test.sh b_test.sh check-structure.sh check-compose.sh; do
    printf '#!/usr/bin/env bash%s\n' "${FAKE_BODY//basename \"\$0\"/echo scripts/\$(basename \"\$0\")}" \
      >"$FAKE_ROOT/scripts/$f"
  done
}

# run_case NAME WANT_EXIT OUT_RE LOG_RE NOT_LOG_RE [ARGS...]
#   WANT_EXIT: số hoặc "nonzero". Pattern là ERE; rỗng = không kiểm.
#   LOG = các lời gọi nối thành một dòng (kiểm được thứ tự).
run_case() {
  local name=$1 want_exit=$2 out_re=$3 log_re=$4 not_log_re=$5
  shift 5
  export FAKE_LOG="$TMP_ROOT/$name.log" FAKE_ROOT
  : >"$FAKE_LOG"
  local out got_exit log
  out=$(cd "$TMP_ROOT" && PATH="$BIN:$PATH" bash "$FAKE_ROOT/scripts/test-all.sh" "$@" 2>&1)
  got_exit=$?
  log=$(tr "\n" "|" <"$FAKE_LOG")

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
  unset FAIL_ON
}

# Thứ tự đầy đủ khi mọi bước pass, mỗi lệnh chạy đúng thư mục.
ALL='^\. scripts/a_test\.sh\|\. scripts/b_test\.sh\|\. scripts/check-structure\.sh\|\. scripts/check-compose\.sh\|'
ALL+='com/tm/server go vet \./\.\.\.\|com/tm/server go test -race \./\.\.\.\|com/tm/server bazel test //\.\.\.\|'
ALL+='com/tm/app pnpm install --frozen-lockfile\|com/tm/app pnpm lint\|com/tm/app pnpm typecheck\|'
ALL+='com/tm/app pnpm test\|com/tm/app pnpm build\|$'

new_root all_pass
run_case all_pass 0 'test-all: OK \(scripts server app\)' "$ALL" ''

# Bộ đầu lỗi → exit ≠0, các bộ sau vẫn chạy, tóm tắt nêu bước lỗi.
new_root script_test_fail
FAIL_ON='a_test' run_case script_test_fail nonzero 'FAIL — 1 bước lỗi' \
  'b_test\.sh.*go vet.*pnpm build' ''
new_root check_compose_fail
FAIL_ON='check-compose' run_case check_compose_fail nonzero '\[scripts\] \.: bash scripts/check-compose\.sh' \
  'pnpm build' ''

# Từng bước server lỗi → exit ≠0, các bước còn lại vẫn chạy.
new_root go_vet_fail
FAIL_ON='^go vet' run_case go_vet_fail nonzero '\[server\] com/tm/server: go vet' 'go test -race.*bazel test.*pnpm build' ''
new_root go_test_fail
FAIL_ON='^go test' run_case go_test_fail nonzero 'fake-fail: go test -race' 'bazel test.*pnpm build' ''
new_root bazel_fail
FAIL_ON='^bazel test' run_case bazel_fail nonzero '\[server\] com/tm/server: bazel test //\.\.\.' 'pnpm build' ''

# Bước chạy cuối cùng lỗi → không bị che.
new_root last_step_fail
FAIL_ON='^pnpm build' run_case last_step_fail nonzero '\[app\] com/tm/app: pnpm build' '' ''
new_root pnpm_test_fail
FAIL_ON='^pnpm test' run_case pnpm_test_fail nonzero '\[app\] com/tm/app: pnpm test' 'pnpm build' ''

# install lỗi → bỏ các bước app còn lại (pnpm sẽ tự install không frozen, sửa lockfile).
new_root install_fail
FAIL_ON='^pnpm install' run_case install_fail nonzero 'bỏ qua lint/typecheck/test/build' '' 'pnpm (lint|typecheck|test|build)'

# Nhiều bước lỗi → đếm đủ.
new_root multi_fail
FAIL_ON='a_test|^go vet|^pnpm lint' run_case multi_fail nonzero 'FAIL — 3 bước lỗi' '' ''

# Chọn bộ.
new_root only_server
run_case only_server 0 'test-all: OK \(server\)' '^com/tm/server go vet.*bazel test //\.\.\.\|$' 'scripts/|pnpm' server
new_root two_suites
run_case two_suites 0 'OK \(app scripts\)' '^com/tm/app pnpm install.*pnpm build\|\. scripts/a_test' '' app scripts
new_root bad_suite
run_case bad_suite 2 'bộ không hợp lệ: nope' '^$' '' server nope

# Không có scripts/*_test.sh → vẫn chạy check-*.sh.
new_root no_tests
rm -f "$FAKE_ROOT"/scripts/*_test.sh
run_case no_tests 0 'không có scripts/\*_test\.sh' '^\. scripts/check-structure\.sh\|\. scripts/check-compose\.sh\|$' '' scripts

echo "test-all_test: $pass pass, $fail fail"
[[ $fail -eq 0 ]]
