#!/usr/bin/env bash
# check-compose_test.sh — test cho scripts/check-compose.sh.
# Mỗi ca ghi một file compose (và Dockerfile nếu cần) vào thư mục tạm; chỉ gọi
# `docker compose config` (không tạo container, không pull image).
#
# Dùng:  bash scripts/check-compose_test.sh
set -uo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
CHECK="$SCRIPT_DIR/check-compose.sh"
REPO_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/check-compose-test.XXXXXX")
trap 'rm -rf "$TMP_ROOT"' EXIT

pass=0
fail=0

HC_OK='    healthcheck:
      test: ["CMD", "pg_isready", "-h", "127.0.0.1"]'

# run_case NAME WANT_EXIT STDERR_PATTERN COMPOSE_BODY [DOCKERFILE_BODY]
#   COMPOSE_BODY: nội dung sau dòng `services:`.
#   DOCKERFILE_BODY: nếu có, ghi vào <case>/app/Dockerfile.
#   STDERR_PATTERN rỗng = không kiểm stderr; "-" = stderr phải rỗng.
run_case() {
  local name=$1 want_exit=$2 pattern=$3 body=$4 dockerfile=${5:-}
  local dir="$TMP_ROOT/$name"
  mkdir -p "$dir/app"
  printf 'name: tc\nservices:\n%s\n' "$body" >"$dir/compose.yml"
  [[ -n "$dockerfile" ]] && printf '%s\n' "$dockerfile" >"$dir/app/Dockerfile"

  local out err got_exit
  out=$(bash "$CHECK" "$dir/compose.yml" 2>"$dir.err")
  got_exit=$?
  err=$(cat "$dir.err")

  local ok=1 why=""
  [[ $got_exit -eq $want_exit ]] || { ok=0; why="exit=$got_exit, muốn $want_exit"; }
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

# --- Hợp lệ ------------------------------------------------------------------
run_case valid_pinned_tag 0 - "  db:
    image: postgres:18.6-alpine3.23
$HC_OK"

run_case valid_digest 0 - "  db:
    image: postgres@sha256:0000000000000000000000000000000000000000000000000000000000000000
$HC_OK"

run_case valid_registry_port 0 - "  db:
    image: localhost:5000/team/db:1.2.3
$HC_OK"

run_case valid_build_multistage 0 - "  app:
    build:
      context: ./app
    image: tc/app:1.0.0
$HC_OK" 'FROM otel/opentelemetry-collector-contrib:0.161.0 AS otelcol
FROM alpine:3.24.2
COPY --from=otelcol /otelcol-contrib /otelcol-contrib'

run_case valid_from_internal_stage 0 - "  app:
    build:
      context: ./app
    image: tc/app:1.0.0
$HC_OK" 'FROM --platform=linux/amd64 golang:1.27.1 AS build
FROM build AS test
FROM scratch'

run_case valid_shell_health 0 - '  cache:
    image: redis:8.8.3-alpine
    healthcheck:
      test: ["CMD-SHELL", "redis-cli -h 127.0.0.1 ping | grep -qx PONG"]'

# --- Image không pin ----------------------------------------------------------
run_case image_latest 1 'UNPINNED image: db' "  db:
    image: postgres:latest
$HC_OK"

run_case image_no_tag 1 'UNPINNED image: db' "  db:
    image: postgres
$HC_OK"

run_case image_registry_port_no_tag 1 'UNPINNED image: db' "  db:
    image: localhost:5000/db
$HC_OK"

run_case build_without_image 1 'UNPINNED build image: app' "  app:
    build:
      context: ./app
$HC_OK" 'FROM alpine:3.24.2'

run_case build_from_latest 1 'UNPINNED FROM: app: FROM alpine:latest' "  app:
    build:
      context: ./app
    image: tc/app:1.0.0
$HC_OK" 'FROM alpine:latest'

run_case build_from_no_tag 1 'UNPINNED FROM: app: FROM otel/opentelemetry-collector-contrib AS otelcol' "  app:
    build:
      context: ./app
    image: tc/app:1.0.0
$HC_OK" 'FROM otel/opentelemetry-collector-contrib AS otelcol
FROM alpine:3.24.2'

run_case build_missing_dockerfile 1 'MISSING Dockerfile: app' "  app:
    build:
      context: ./app
      dockerfile: Nope.Dockerfile
    image: tc/app:1.0.0
$HC_OK"

# --- Healthcheck thiếu / giả ------------------------------------------------------
run_case health_missing 1 'MISSING healthcheck: db' '  db:
    image: postgres:18.6'

run_case health_disabled 1 'MISSING healthcheck: db' '  db:
    image: postgres:18.6
    healthcheck:
      disable: true'

fake_case() {
  local name=$1 test=$2
  run_case "$name" 1 'FAKE healthcheck: svc' "  svc:
    image: alpine:3.24.2
    healthcheck:
      test: $test"
}
fake_case fake_cmd_true '["CMD", "true"]'
fake_case fake_shell_exit0 '["CMD-SHELL", "exit 0"]'
fake_case fake_colon '["CMD-SHELL", ":"]'
fake_case fake_or_exit0 '["CMD-SHELL", "wget -q http://127.0.0.1:1/ || exit 0"]'
fake_case fake_or_true '["CMD-SHELL", "pg_isready || true"]'
fake_case fake_validate '["CMD", "/otelcol-contrib", "validate", "--config", "/c.yaml"]'
fake_case fake_print_config '["CMD", "/otelcol-contrib", "print-config"]'
fake_case fake_pgrep '["CMD", "pgrep", "mongod"]'
fake_case fake_kill0 '["CMD-SHELL", "kill -0 1"]'
fake_case fake_ps '["CMD-SHELL", "ps aux | grep -q tempo"]'

# --- Lỗi đầu vào ------------------------------------------------------------------
missing_file() {
  local out got_exit
  out=$(bash "$CHECK" "$TMP_ROOT/khong-ton-tai.yml" 2>&1)
  got_exit=$?
  if [[ $got_exit -eq 2 ]]; then
    echo "PASS missing_compose_file"
    pass=$((pass + 1))
  else
    echo "FAIL missing_compose_file — exit=$got_exit, muốn 2 ($out)"
    fail=$((fail + 1))
  fi
}
missing_file

# --- Nhiều vi phạm được liệt kê đủ ------------------------------------------------
run_case many_violations 1 'check-compose: 4 vi phạm' '  a:
    image: redis
  b:
    image: mongo:latest
    healthcheck:
      test: ["CMD", "true"]'

# --- File thật của repo -------------------------------------------------------------
repo_file() {
  local out got_exit
  out=$(bash "$CHECK" "$REPO_ROOT/deploy/docker-compose.yml" 2>&1)
  got_exit=$?
  if [[ $got_exit -eq 0 ]]; then
    echo "PASS repo_compose"
    pass=$((pass + 1))
  else
    echo "FAIL repo_compose — exit=$got_exit ($out)"
    fail=$((fail + 1))
  fi
}
repo_file

echo "---"
echo "PASS: $pass  FAIL: $fail"
[[ $fail -eq 0 ]]
