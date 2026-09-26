#!/usr/bin/env bash
# ci-affected_test.sh — test cho scripts/ci-affected.sh.
# Mỗi ca dựng một repo git trong thư mục tạm, không đụng repo thật. Bazel được
# thay bằng một script giả (biến BAZEL) ghi lại truy vấn và trả kết quả dựng
# sẵn — test kiểm cách ánh xạ file đổi → label và xử lý kết quả/exit code;
# hành vi với Bazel thật được kiểm ở test case P0-T04-TC14..TC18.
#
# Dùng:  bash scripts/ci-affected_test.sh
set -uo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SUT="$SCRIPT_DIR/ci-affected.sh"

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/ci-affected-test.XXXXXX")
trap 'rm -rf "$TMP_ROOT"' EXIT

pass=0
fail=0

# Bazel giả: mỗi lần gọi ghi một dòng "<args>" vào $FAKE_LOG.
# `query --keep_going ...` (bước lọc) → in $FAKE_SET_OUT, exit $FAKE_SET_RC.
# `query ...` còn lại (rdeps)       → in $FAKE_RDEPS_OUT, exit $FAKE_RDEPS_RC.
FAKE_BAZEL="$TMP_ROOT/fake-bazel"
cat >"$FAKE_BAZEL" <<'EOF'
#!/usr/bin/env bash
echo "$*" >>"$FAKE_LOG"
if [[ "$*" == *--keep_going* ]]; then
  [[ -n "${FAKE_SET_OUT:-}" ]] && printf '%s\n' "$FAKE_SET_OUT"
  exit "${FAKE_SET_RC:-0}"
fi
[[ -n "${FAKE_RDEPS_OUT:-}" ]] && printf '%s\n' "$FAKE_RDEPS_OUT"
exit "${FAKE_RDEPS_RC:-0}"
EOF
chmod +x "$FAKE_BAZEL"

g() { git -C "$REPO" "$@"; }

# new_repo NAME — workspace giả: package gốc, //pkg/a, //pkg/b, tools/rules.
new_repo() {
  REPO="$TMP_ROOT/$1"
  local s="$REPO/com/tm/server" f
  mkdir -p "$s/pkg/a/sub" "$s/pkg/b" "$s/tools/rules" "$s/db/core/migrations" "$REPO/com/tm/app"
  g init -q -b main
  g config user.email t@example.com
  g config user.name t
  g config commit.gpgsign false
  for f in BUILD.bazel MODULE.bazel MODULE.bazel.lock go.mod .bazelrc .bazelversion \
    pkg/a/BUILD.bazel pkg/a/a.go pkg/a/sub/data.txt pkg/b/BUILD.bazel pkg/b/b.go \
    tools/rules/BUILD.bazel tools/rules/x.bzl db/core/migrations/00001_init.sql; do
    echo "# $f" >"$s/$f"
  done
  echo '{}' >"$REPO/com/tm/app/package.json"
  g add -A
  g commit -qm base
}

# change PATH... — sửa/tạo file (tương đối với com/tm/server; bắt đầu bằng
# "/" = tương đối với gốc repo) rồi commit.
change() {
  local f p
  for f in "$@"; do
    if [[ "$f" == /* ]]; then p="$REPO$f"; else p="$REPO/com/tm/server/$f"; fi
    mkdir -p "$(dirname "$p")"
    echo "$RANDOM" >>"$p"
  done
  g add -A
  g commit -qm change
}

remove() {
  g rm -rq "$@"
  g commit -qm rm
}

# run_case NAME WANT_EXIT WANT_OUT WANT_LOG BASE
#   WANT_OUT  stdout mong đợi (các dòng nối bằng ",").
#   WANT_LOG  regex ERE phải khớp log Bazel giả; "-" = Bazel không được gọi;
#             rỗng = không kiểm.
#   FAKE_* đặt trước khi gọi (env của lệnh).
run_case() {
  local name=$1 want_exit=$2 want_out=$3 want_log=$4 base=$5
  export FAKE_LOG="$TMP_ROOT/$name.log"
  : >"$FAKE_LOG"
  local out got_exit
  out=$(cd "$REPO" && BAZEL="$FAKE_BAZEL" bash "$SUT" "$base" HEAD 2>/dev/null)
  got_exit=$?
  local got log
  got=$(tr '\n' ',' <<<"$out")
  got=${got%,}
  log=$(cat "$FAKE_LOG")

  local ok=1 why=""
  [[ $got_exit -eq $want_exit ]] || { ok=0; why="exit=$got_exit, muốn $want_exit"; }
  [[ $ok -eq 0 || "$got" == "$want_out" ]] || { ok=0; why="stdout '$got', muốn '$want_out'"; }
  if [[ $ok -eq 1 && "$want_log" == "-" ]]; then
    [[ -z "$log" ]] || { ok=0; why="bazel không được gọi, nhưng log: $log"; }
  elif [[ $ok -eq 1 && -n "$want_log" ]]; then
    grep -qE -- "$want_log" <<<"$log" || { ok=0; why="log không khớp /$want_log/: $log"; }
  fi

  if [[ $ok -eq 1 ]]; then
    echo "PASS $name"
    pass=$((pass + 1))
  else
    echo "FAIL $name — $why"
    fail=$((fail + 1))
  fi
  unset FAKE_SET_OUT FAKE_SET_RC FAKE_RDEPS_OUT FAKE_RDEPS_RC
}

# --- Chạy tất cả, không gọi Bazel ---
new_repo empty_base
run_case empty_base 0 "//..." - ""
new_repo zero_base
run_case zero_base 0 "//..." - 0000000000000000000000000000000000000000
for f in MODULE.bazel MODULE.bazel.lock go.mod go.sum .bazelrc .bazelversion BUILD.bazel tools/rules/x.bzl; do
  n="global_${f//[^A-Za-z]/_}"
  new_repo "$n"
  change "$f"
  run_case "$n" 0 "//..." - main~1
done

# --- Không có file server nào đổi ---
new_repo app_only
change /com/tm/app/package.json /README.md
run_case app_only 0 "" - main~1

# --- Ánh xạ file → label ứng viên ---
new_repo source_file
change pkg/a/a.go
FAKE_SET_OUT="//pkg/a:a.go" FAKE_RDEPS_OUT=$'//pkg/b:b_test\n//pkg/a:a_test' \
  run_case source_file 0 "//pkg/a:a_test,//pkg/b:b_test" \
  'query --keep_going --output=label set\("//pkg/a:a\.go" \)' main~1
# Truy vấn rdeps nhận đúng tập đã lọc, dùng tests() và loại tag manual.
new_repo rdeps_expr
change pkg/a/a.go
FAKE_SET_OUT="//pkg/a:a.go" \
  run_case rdeps_expr 0 "" \
  'query --output=label tests\(rdeps\(//\.\.\., set\("//pkg/a:a\.go" \)\)\) except attr\("tags", ".*manual.*", //\.\.\.\)' main~1
# File trong thư mục con không có BUILD → thuộc package cha.
new_repo nested_file
change pkg/a/sub/data.txt
run_case nested_file 0 "" 'set\("//pkg/a:sub/data\.txt" \)' main~1
# File BUILD đổi → cả package.
new_repo build_file
change pkg/b/BUILD.bazel
run_case build_file 0 "" 'set\("//pkg/b:all" \)' main~1
# File trong package gốc (không phải BUILD.bazel gốc) → //:<đường dẫn>, không //...
new_repo root_package_file
change db/core/migrations/99999_tc.sql
run_case root_package_file 0 "" 'set\("//:db/core/migrations/99999_tc\.sql" \)' main~1
# File bị xoá → package gần nhất còn tồn tại.
new_repo deleted_file
remove com/tm/server/pkg/a/a.go
run_case deleted_file 0 "" 'set\("//pkg/a:all" \)' main~1
# Cả package bị xoá → package cha (ở đây là gốc), không nhắc tới //pkg/b.
new_repo deleted_package
remove com/tm/server/pkg/b
run_case deleted_package 0 "" 'set\("//:all" \)$' main~1
# Nhiều file → một truy vấn chứa mọi ứng viên.
new_repo many_files
change pkg/a/a.go pkg/b/b.go
run_case many_files 0 "" 'set\("//pkg/a:a\.go" "//pkg/b:b\.go" \)' main~1

# --- Kết quả / exit code của Bazel ---
# Không ứng viên nào là target → không in gì, không chạy rdeps.
new_repo no_target
change db/core/migrations/99999_tc.sql
FAKE_SET_RC=3 run_case no_target 0 "" "" main~1
if [[ $(grep -c . "$TMP_ROOT/no_target.log") -eq 1 ]]; then
  echo "PASS no_target_single_query"
  pass=$((pass + 1))
else
  echo "FAIL no_target_single_query — $(cat "$TMP_ROOT/no_target.log")"
  fail=$((fail + 1))
fi
# --keep_going exit 3 (kết quả một phần) được chấp nhận.
new_repo partial
change pkg/a/a.go db/x.sql
FAKE_SET_OUT="//pkg/a:a.go" FAKE_SET_RC=3 FAKE_RDEPS_OUT="//pkg/a:a_test" \
  run_case partial 0 "//pkg/a:a_test" "" main~1
# Lỗi khác ở bước lọc → fail.
new_repo set_error
change pkg/a/a.go
FAKE_SET_RC=2 run_case set_error 2 "" "" main~1
# Lỗi ở bước rdeps → fail.
new_repo rdeps_error
change pkg/a/a.go
FAKE_SET_OUT="//pkg/a:a.go" FAKE_RDEPS_RC=7 run_case rdeps_error 7 "" "" main~1

echo "ci-affected_test: $pass pass, $fail fail"
[[ $fail -eq 0 ]]
