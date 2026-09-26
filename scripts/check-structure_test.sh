#!/usr/bin/env bash
# check-structure_test.sh — test cho scripts/check-structure.sh.
# Mỗi ca dựng một cây thư mục trong thư mục tạm, không đụng repo thật.
#
# Dùng:  bash scripts/check-structure_test.sh
set -uo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
CHECK="$SCRIPT_DIR/check-structure.sh"

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/check-structure-test.XXXXXX")
trap 'rm -rf "$TMP_ROOT"' EXIT

pass=0
fail=0

# make_valid_tree DIR — dựng cây hợp lệ tối thiểu.
make_valid_tree() {
  local r=$1
  mkdir -p "$r"/{deploy,loadtest/results,scripts,com/tm/docs} \
    "$r"/com/tm/server/{api,pkg,services} \
    "$r"/com/tm/app/{api,apps,packages}
}

# run_case NAME WANT_EXIT STDERR_PATTERN SETUP_FN
#   STDERR_PATTERN rỗng = không kiểm stderr; "-" = stderr phải rỗng.
run_case() {
  local name=$1 want_exit=$2 pattern=$3 setup=$4
  local root="$TMP_ROOT/$name"
  make_valid_tree "$root"
  "$setup" "$root"

  local out err got_exit
  out=$(bash "$CHECK" "$root" 2>"$TMP_ROOT/$name.err")
  got_exit=$?
  err=$(cat "$TMP_ROOT/$name.err")

  local ok=1 why=""
  if [[ "$want_exit" == "nonzero" ]]; then
    [[ $got_exit -ne 0 ]] || { ok=0; why="exit=$got_exit, muốn ≠0"; }
  else
    [[ $got_exit -eq $want_exit ]] || { ok=0; why="exit=$got_exit, muốn $want_exit"; }
  fi
  if [[ $ok -eq 1 && "$pattern" == "-" ]]; then
    [[ -z "$err" ]] || { ok=0; why="stderr không rỗng: $err"; }
  elif [[ $ok -eq 1 && -n "$pattern" ]]; then
    grep -qE -- "$pattern" <<<"$err" || { ok=0; why="stderr không khớp /$pattern/: $err"; }
  fi

  if [[ $ok -eq 1 ]]; then
    echo "PASS $name"
    pass=$((pass + 1))
  else
    echo "FAIL $name — $why (stdout: $out)"
    fail=$((fail + 1))
  fi
}

noop() { :; }
rm_loadtest() { rm -rf "$1/loadtest"; }
rm_server_pkg() { rm -rf "$1/com/tm/server/pkg"; }
rm_app_packages() { rm -rf "$1/com/tm/app/packages"; }
rm_loadtest_results() { rm -rf "$1/loadtest/results"; }
add_pkg_utils() { mkdir -p "$1/com/tm/server/pkg/utils" && touch "$1/com/tm/server/pkg/utils/.gitkeep"; }
add_repository() { mkdir -p "$1/com/tm/server/services/core/internal/repository"; }
add_app_shared() { mkdir -p "$1/com/tm/app/packages/shared"; }
add_deploy_common() { mkdir -p "$1/deploy/common"; }
# "services" (số nhiều) là thư mục hợp lệ, không được nhầm với "service".
services_is_ok() { mkdir -p "$1/com/tm/server/services/core/internal/booking"; }
# Tên cấm trong node_modules / bazel-* không do ta đặt → bỏ qua.
ignore_vendor() {
  mkdir -p "$1/com/tm/app/node_modules/foo/utils" "$1/com/tm/server/bazel-out/common"
}
missing_and_forbidden() { rm -rf "$1/deploy"; mkdir -p "$1/com/tm/app/apps/bff/src/utils"; }

run_case valid                0        "-"                                   noop
run_case services_plural_ok   0        "-"                                   services_is_ok
run_case vendor_dirs_ignored  0        "-"                                   ignore_vendor
run_case missing_loadtest     nonzero  "MISSING dir: loadtest$"              rm_loadtest
run_case missing_loadtest_results nonzero "MISSING dir: loadtest/results$"   rm_loadtest_results
run_case missing_server_pkg   nonzero  "MISSING dir: com/tm/server/pkg$"     rm_server_pkg
run_case missing_app_packages nonzero  "MISSING dir: com/tm/app/packages$"   rm_app_packages
run_case forbidden_pkg_utils  nonzero  "FORBIDDEN dir: com/tm/server/pkg/utils$" add_pkg_utils
run_case forbidden_repository nonzero  "FORBIDDEN dir: com/tm/server/services/core/internal/repository$" add_repository
run_case forbidden_app_shared nonzero  "FORBIDDEN dir: com/tm/app/packages/shared$" add_app_shared
run_case forbidden_deploy_common nonzero "FORBIDDEN dir: deploy/common$"     add_deploy_common
run_case missing_and_forbidden nonzero "MISSING dir: deploy$"                missing_and_forbidden
run_case missing_and_forbidden_2 nonzero "FORBIDDEN dir: com/tm/app/apps/bff/src/utils$" missing_and_forbidden

# ROOT không tồn tại → exit 2.
if bash "$CHECK" "$TMP_ROOT/does-not-exist" 2>/dev/null; then
  echo "FAIL root_not_dir — exit 0"; fail=$((fail + 1))
else
  rc=$?
  if [[ $rc -eq 2 ]]; then echo "PASS root_not_dir"; pass=$((pass + 1))
  else echo "FAIL root_not_dir — exit=$rc, muốn 2"; fail=$((fail + 1)); fi
fi

# Không truyền ROOT → dùng cwd (không phải git repo).
if (cd "$TMP_ROOT/valid" && GIT_CEILING_DIRECTORIES="$TMP_ROOT" bash "$CHECK" >/dev/null 2>&1); then
  echo "PASS default_root_cwd"; pass=$((pass + 1))
else
  echo "FAIL default_root_cwd"; fail=$((fail + 1))
fi

echo "---"
echo "$pass passed, $fail failed"
[[ $fail -eq 0 ]]
