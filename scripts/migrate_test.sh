#!/usr/bin/env bash
# migrate_test.sh — test cho scripts/migrate.sh (`make migrate`).
# goose và go là script giả đặt trong PATH riêng (PATH chỉ gồm thư mục giả +
# /usr/bin:/bin, nên goose/go thật ở /opt/homebrew, ~/go/bin không lọt vào).
# goose giả ghi lại lời gọi và lỗi khi URL chứa "fail". Chạy với goose và DB
# thật được kiểm ở P0-T05-TC08..TC14.
#
# Dùng:  bash scripts/migrate_test.sh
set -uo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SUT="$SCRIPT_DIR/migrate.sh"
ROOT=$(cd "$SCRIPT_DIR/.." && pwd)

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/migrate-test.XXXXXX")
trap 'rm -rf "$TMP_ROOT"' EXIT

pass=0
fail=0

# write_goose FILE — goose giả: `-version` in bản $FAKE_GOOSE_VERSION; lệnh khác
# ghi "<tên file> <args>" vào $FAKE_LOG, exit 1 nếu URL (đối số 4) chứa "fail".
write_goose() {
  mkdir -p "$(dirname "$1")"
  cat >"$1" <<'EOF'
#!/usr/bin/env bash
if [[ "$1" == "-version" ]]; then echo "goose version: ${FAKE_GOOSE_VERSION:-v3.28.0}"; exit 0; fi
echo "$0 $*" >>"$FAKE_LOG"
if [[ "$4" == *fail* ]]; then echo "failed to connect: $4" >&2; exit 1; fi
echo "OK   00001_init.sql"
EOF
  chmod +x "$1"
}

# write_go DIR — go giả: `go env GOPATH` in $FAKE_GOPATH, `go env GOBIN` in $FAKE_GOBIN.
write_go() {
  mkdir -p "$1"
  cat >"$1/go" <<'EOF'
#!/usr/bin/env bash
[[ "$1" == "env" ]] || exit 2
case "$2" in
  GOPATH) echo "${FAKE_GOPATH-}" ;;
  GOBIN) echo "${FAKE_GOBIN-}" ;;
esac
EOF
  chmod +x "$1/go"
}

# Thư mục bin dùng chung: có go giả, không có goose.
GOBIN_ONLY="$TMP_ROOT/bin-go"
write_go "$GOBIN_ONLY"
# Có cả go và goose giả trong PATH.
BOTH="$TMP_ROOT/bin-both"
write_go "$BOTH"
write_goose "$BOTH/goose"
# Không có go lẫn goose.
NONE="$TMP_ROOT/bin-none"
mkdir -p "$NONE"

CORE_DIR="$ROOT/com/tm/server/db/core/migrations"
AN_DIR="$ROOT/com/tm/server/db/analytics/migrations"

# run_case NAME BIN_DIR WANT_EXIT OUT_RE LOG_RE NOT_OUT_RE [VAR=VAL...]
#   WANT_EXIT: số hoặc "nonzero". Pattern là ERE; rỗng = không kiểm.
#   OUT = stdout + stderr; LOG = các dòng goose giả ghi, nối thành một dòng.
run_case() {
  local name=$1 bin=$2 want_exit=$3 out_re=$4 log_re=$5 not_out_re=$6
  shift 6
  export FAKE_LOG="$TMP_ROOT/$name.log"
  : >"$FAKE_LOG"
  local out got_exit log
  out=$(cd "$TMP_ROOT" && env -u CORE_DATABASE_URL -u ANALYTICS_DATABASE_URL -u MIGRATE_CONNECT_TIMEOUT \
    PATH="$bin:/usr/bin:/bin" "$@" bash "$SUT" 2>&1)
  got_exit=$?
  log=$(tr "\n" " " <"$FAKE_LOG")

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
  if [[ $ok -eq 1 && -n "$not_out_re" ]] && grep -qE -- "$not_out_re" <<<"$out"; then
    ok=0
    why="output không được khớp /$not_out_re/: $out"
  fi

  if [[ $ok -eq 1 ]]; then
    echo "PASS $name"
    pass=$((pass + 1))
  else
    echo "FAIL $name — $why"
    fail=$((fail + 1))
  fi
}

DEF_CORE='postgres://snaptix:snaptix@localhost:5432/core\?connect_timeout=10'
DEF_AN='postgres://snaptix:snaptix@localhost:5433/analytics\?connect_timeout=10'

# goose trong PATH, URL mặc định: core rồi analytics, thư mục migration tuyệt đối của repo.
run_case default_urls "$BOTH" 0 'migrate: OK' \
  "goose -dir $CORE_DIR postgres $DEF_CORE up .*goose -dir $AN_DIR postgres $DEF_AN up" ''

# Mật khẩu không in ra log.
run_case mask_password "$BOTH" 0 'postgres://snaptix:\*\*\*@localhost:5432/core' '' 'snaptix:snaptix@'

# Biến môi trường ghi đè mặc định.
run_case env_override "$BOTH" 0 '' \
  'postgres://u:p@db1:6000/c1\?connect_timeout=10 up .*postgres://u:p@db2:6001/a1\?connect_timeout=10 up' '' \
  CORE_DATABASE_URL=postgres://u:p@db1:6000/c1 ANALYTICS_DATABASE_URL=postgres://u:p@db2:6001/a1

# connect_timeout: URL có query → thêm bằng &; đã có → giữ nguyên; đổi số giây bằng biến.
run_case timeout_append "$BOTH" 0 '' 'postgres://h/c\?sslmode=disable&connect_timeout=10 up' '' \
  CORE_DATABASE_URL='postgres://h/c?sslmode=disable'
run_case timeout_keep "$BOTH" 0 '' 'postgres://h/c\?connect_timeout=3 up' 'connect_timeout=3&' \
  CORE_DATABASE_URL='postgres://h/c?connect_timeout=3'
run_case timeout_env "$BOTH" 0 '' 'localhost:5432/core\?connect_timeout=2 up' '' MIGRATE_CONNECT_TIMEOUT=2

# goose không có trong PATH → tìm ở GOPATH/bin (máy dev: ~/go/bin ngoài PATH).
write_goose "$TMP_ROOT/gopath1/bin/goose"
run_case gopath_bin "$GOBIN_ONLY" 0 "goose: $TMP_ROOT/gopath1/bin/goose" "^$TMP_ROOT/gopath1/bin/goose -dir" '' \
  FAKE_GOPATH="$TMP_ROOT/gopath1"

# GOPATH là danh sách: phần tử đầu không có goose, phần tử sau có.
mkdir -p "$TMP_ROOT/gopath-empty"
write_goose "$TMP_ROOT/gopath2/bin/goose"
run_case gopath_list "$GOBIN_ONLY" 0 '' "^$TMP_ROOT/gopath2/bin/goose -dir" '' \
  FAKE_GOPATH="$TMP_ROOT/gopath-empty:$TMP_ROOT/gopath2"

# GOBIN được ưu tiên hơn GOPATH/bin.
write_goose "$TMP_ROOT/gobin/goose"
run_case gobin_first "$GOBIN_ONLY" 0 '' "^$TMP_ROOT/gobin/goose -dir" '' \
  FAKE_GOBIN="$TMP_ROOT/gobin" FAKE_GOPATH="$TMP_ROOT/gopath1"

# Không có goose ở đâu cả → exit ≠0, in lệnh cài đúng bản pin, không @latest, không gọi goose.
run_case no_goose "$GOBIN_ONLY" nonzero \
  'go install github\.com/pressly/goose/v3/cmd/goose@v3\.28\.0' '^$' '@latest' \
  FAKE_GOPATH="$TMP_ROOT/gopath-empty"
run_case no_go "$NONE" nonzero 'goose@v3\.28\.0' '^$' ''

# goose khác bản pin → vẫn chạy, có cảnh báo.
run_case version_warn "$BOTH" 0 'CẢNH BÁO.*v3\.28\.0' 'goose -dir' '' FAKE_GOOSE_VERSION=v3.20.0

# core lỗi → exit ≠0, nêu core, analytics vẫn được migrate.
run_case core_fail "$BOTH" nonzero 'migrate core: LỖI.*CORE_DATABASE_URL' \
  "postgres://fail/core\?connect_timeout=10 up .*$AN_DIR postgres $DEF_AN up" 'migrate analytics: LỖI' \
  CORE_DATABASE_URL=postgres://fail/core

# analytics lỗi (DB chạy sau cùng) → lỗi không bị che, nêu analytics.
run_case analytics_fail "$BOTH" nonzero 'migrate analytics: LỖI.*ANALYTICS_DATABASE_URL' \
  "$CORE_DIR postgres $DEF_CORE up" 'migrate core: LỖI' \
  ANALYTICS_DATABASE_URL=postgres://fail/analytics

# Cả hai lỗi → liệt kê cả hai.
run_case both_fail "$BOTH" nonzero 'thất bại ở: core analytics' '' '' \
  CORE_DATABASE_URL=postgres://fail/core ANALYTICS_DATABASE_URL=postgres://fail/analytics

# Chạy từ thư mục khác (run_case luôn cd vào TMP_ROOT) mà đường dẫn vẫn là của repo:
# đã kiểm ở default_urls ($CORE_DIR tuyệt đối). Thêm ca gọi qua đường dẫn tương đối.
FAKE_LOG="$TMP_ROOT/relpath.log"
: >"$FAKE_LOG"
if (cd "$ROOT/com" && env -u CORE_DATABASE_URL -u ANALYTICS_DATABASE_URL FAKE_LOG="$FAKE_LOG" PATH="$BOTH:/usr/bin:/bin" bash ../scripts/migrate.sh >/dev/null 2>&1) &&
  grep -q -- "-dir $CORE_DIR " "$FAKE_LOG"; then
  echo "PASS relpath"
  pass=$((pass + 1))
else
  echo "FAIL relpath — $(cat "$FAKE_LOG")"
  fail=$((fail + 1))
fi

echo "migrate_test: $pass pass, $fail fail"
[[ $fail -eq 0 ]]
