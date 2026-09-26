#!/usr/bin/env bash
# migrate.sh — `make migrate`: goose up cho PG core và PG analytics, chạy trên
# host (bước 2 của com/tm/docs/technical/local-setup.md). Tương đương:
#   goose -dir com/tm/server/db/core/migrations      postgres "$CORE_DATABASE_URL" up
#   goose -dir com/tm/server/db/analytics/migrations postgres "$ANALYTICS_DATABASE_URL" up
#
# Dùng:  bash scripts/migrate.sh        (chạy được từ thư mục bất kỳ)
#
# Biến môi trường (chưa đặt → mặc định theo bảng "Biến môi trường" của local-setup):
#   CORE_DATABASE_URL        postgres://snaptix:snaptix@localhost:5432/core
#   ANALYTICS_DATABASE_URL   postgres://snaptix:snaptix@localhost:5433/analytics
#   MIGRATE_CONNECT_TIMEOUT  giây chờ kết nối DB (mặc định 10); thêm `connect_timeout`
#                            vào URL nếu URL chưa có, để DB không phản hồi không làm treo.
#
# goose (pin v3.28.0): tìm trong PATH, không có thì trong `go env GOBIN`, rồi
# `<mỗi phần tử của go env GOPATH>/bin` — nơi `go install` đặt binary, thường
# chưa có trong PATH. Không tìm thấy → exit 1 kèm lệnh cài đúng bản.
#
# Migrate cả hai DB kể cả khi DB đầu lỗi; exit 1 nếu có DB lỗi (in tên DB).
set -uo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
ROOT=$(cd "$SCRIPT_DIR/.." && pwd)

GOOSE_VERSION=v3.28.0
# Viết nguyên văn (không ghép từ GOOSE_VERSION) để grep được lệnh cài giống docs.
GOOSE_INSTALL="go install github.com/pressly/goose/v3/cmd/goose@v3.28.0"

# find_goose — in đường dẫn goose; exit 1 nếu không có.
find_goose() {
  local p dir gobin gopath parts=()
  if p=$(command -v goose 2>/dev/null); then
    printf '%s\n' "$p"
    return 0
  fi
  command -v go >/dev/null 2>&1 || return 1
  gobin=$(go env GOBIN 2>/dev/null) || gobin=""
  if [[ -n "$gobin" && -x "$gobin/goose" ]]; then
    printf '%s\n' "$gobin/goose"
    return 0
  fi
  gopath=$(go env GOPATH 2>/dev/null) || return 1
  IFS=: read -r -a parts <<<"$gopath"
  for dir in ${parts[@]+"${parts[@]}"}; do
    if [[ -n "$dir" && -x "$dir/bin/goose" ]]; then
      printf '%s\n' "$dir/bin/goose"
      return 0
    fi
  done
  return 1
}

# mask_url URL — ẩn mật khẩu để in ra log: postgres://u:***@host/db.
mask_url() {
  sed -E 's#(://[^:/@]+:)[^@]*@#\1***@#' <<<"$1"
}

# with_timeout URL SECONDS — thêm connect_timeout nếu URL chưa có.
with_timeout() {
  local url=$1 secs=$2
  case "$url" in
    *connect_timeout=*) printf '%s\n' "$url" ;;
    *\?*) printf '%s&connect_timeout=%s\n' "$url" "$secs" ;;
    *) printf '%s?connect_timeout=%s\n' "$url" "$secs" ;;
  esac
}

main() {
  local goose
  if ! goose=$(find_goose); then
    {
      echo "migrate: không tìm thấy goose (PATH, go env GOBIN, go env GOPATH/bin)."
      echo "  Cài đúng bản pin: $GOOSE_INSTALL"
      echo "  rồi thêm vào PATH: export PATH=\"\$(go env GOPATH)/bin:\$PATH\""
    } >&2
    return 1
  fi

  local version
  version=$("$goose" -version 2>&1 | head -1)
  echo "==> goose: $goose ($version)"
  if [[ "$version" != *"$GOOSE_VERSION"* ]]; then
    echo "migrate: CẢNH BÁO — goose không phải bản pin $GOOSE_VERSION. Cài lại: $GOOSE_INSTALL" >&2
  fi

  local timeout=${MIGRATE_CONNECT_TIMEOUT:-10}
  local failed=() db var def url dir
  for db in core analytics; do
    case "$db" in
      core)
        var=CORE_DATABASE_URL
        def=postgres://snaptix:snaptix@localhost:5432/core
        ;;
      analytics)
        var=ANALYTICS_DATABASE_URL
        def=postgres://snaptix:snaptix@localhost:5433/analytics
        ;;
    esac
    url=${!var:-$def}
    dir="$ROOT/com/tm/server/db/$db/migrations"
    echo "==> migrate $db: goose -dir com/tm/server/db/$db/migrations postgres $(mask_url "$url") up"
    if "$goose" -dir "$dir" postgres "$(with_timeout "$url" "$timeout")" up; then
      echo "    $db: OK"
    else
      echo "migrate $db: LỖI — DB $db không migrate được. Kiểm tra stack đã chạy (make up) và $var." >&2
      failed+=("$db")
    fi
  done

  if [[ ${#failed[@]} -gt 0 ]]; then
    echo "migrate: thất bại ở: ${failed[*]}" >&2
    return 1
  fi
  echo "migrate: OK (core, analytics)"
}

main "$@"
