#!/usr/bin/env bash
# ci-changes_test.sh — test cho scripts/ci-changes.sh.
# Mỗi ca dựng một repo git trong thư mục tạm, không đụng repo thật.
#
# Dùng:  bash scripts/ci-changes_test.sh
set -uo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SUT="$SCRIPT_DIR/ci-changes.sh"

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/ci-changes-test.XXXXXX")
trap 'rm -rf "$TMP_ROOT"' EXIT

pass=0
fail=0

g() { git -C "$REPO" "$@"; }

# new_repo NAME — repo có sẵn file ở mọi vùng, commit `base`, nhánh `main`.
new_repo() {
  REPO="$TMP_ROOT/$1"
  mkdir -p "$REPO"
  g init -q -b main
  g config user.email t@example.com
  g config user.name t
  g config commit.gpgsign false
  mkdir -p "$REPO"/com/tm/server/pkg/a "$REPO"/com/tm/app/apps/x "$REPO"/com/tm/docs \
    "$REPO"/.github/workflows "$REPO"/scripts "$REPO"/deploy
  echo a >"$REPO/com/tm/server/pkg/a/a.go"
  echo x >"$REPO/com/tm/app/apps/x/index.js"
  echo d >"$REPO/com/tm/docs/d.md"
  echo w >"$REPO/.github/workflows/ci.yml"
  echo s >"$REPO/scripts/ci-app.sh"
  echo c >"$REPO/scripts/check-structure.sh"
  echo r >"$REPO/README.md"
  echo y >"$REPO/deploy/docker-compose.yml"
  g add -A
  g commit -qm base
}

# change PATH... — sửa (hoặc tạo) từng file rồi commit.
change() {
  local f
  for f in "$@"; do
    mkdir -p "$(dirname "$REPO/$f")"
    echo "$RANDOM" >>"$REPO/$f"
  done
  g add -A
  g commit -qm change
}

# run_case NAME WANT BASE [HEAD] — WANT dạng "server=..,app=..".
run_case() {
  local name=$1 want=$2 base=$3 head=${4:-HEAD}
  local out got_exit
  out=$(cd "$REPO" && bash "$SUT" "$base" "$head" 2>/dev/null)
  got_exit=$?
  local got
  got=$(tr '\n' ',' <<<"$out")
  got=${got%,}
  if [[ $got_exit -eq 0 && "$got" == "$want" ]]; then
    echo "PASS $name"
    pass=$((pass + 1))
  else
    echo "FAIL $name — exit=$got_exit, got '$got', muốn '$want'"
    fail=$((fail + 1))
  fi
}

new_repo server_only
change com/tm/server/pkg/a/a.go
run_case server_only "server=true,app=false" main~1

new_repo app_only
change com/tm/app/apps/x/index.js
run_case app_only "server=false,app=true" main~1

new_repo outside_only
change com/tm/docs/d.md README.md deploy/docker-compose.yml CLAUDE.md scripts/check-structure.sh
run_case outside_only "server=false,app=false" main~1

new_repo both
change com/tm/server/pkg/a/a.go com/tm/app/apps/x/index.js
run_case both "server=true,app=true" main~1

new_repo workflow_file
change .github/workflows/ci.yml
run_case workflow_file "server=true,app=true" main~1

new_repo ci_script
change scripts/ci-app.sh
run_case ci_script "server=true,app=true" main~1

new_repo prefix_boundary
change com/tm/serverx/a.txt com/tm/app-old/b.txt
run_case prefix_boundary "server=false,app=false" main~1

new_repo delete_server_file
g rm -q com/tm/server/pkg/a/a.go
g commit -qm rm
run_case delete_server_file "server=true,app=false" main~1

new_repo rename_app_to_docs
g mv com/tm/app/apps/x/index.js com/tm/docs/index.js
g commit -qm mv
run_case rename_app_to_docs "server=false,app=true" main~1

new_repo empty_base
run_case empty_base "server=true,app=true" ""

new_repo zero_base
run_case zero_base "server=true,app=true" 0000000000000000000000000000000000000000

new_repo unknown_base
run_case unknown_base "server=true,app=true" deadbeefdeadbeefdeadbeefdeadbeefdeadbeef

new_repo no_change
run_case no_change "server=false,app=false" HEAD

# Push lên main: base = before (HEAD~1) → chỉ commit mới.
new_repo push_before
change com/tm/server/pkg/a/a.go
change com/tm/app/apps/x/index.js
run_case push_before "server=false,app=true" HEAD~1

# PR: main đi tiếp sau điểm tách nhánh → thay đổi phía main không tính.
new_repo merge_base
g checkout -q -b feature
change com/tm/app/apps/x/index.js
g checkout -q main
change com/tm/server/pkg/a/a.go
g checkout -q feature
run_case merge_base "server=false,app=true" main

echo "ci-changes_test: $pass pass, $fail fail"
[[ $fail -eq 0 ]]
