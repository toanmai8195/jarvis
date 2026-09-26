#!/usr/bin/env bash
# check-compose.sh — kiểm tra tĩnh deploy/docker-compose.yml (P0-T02):
#   1. image pin phiên bản: không `:latest`, không thiếu tag (digest @sha256 được chấp nhận);
#      service có `build:` phải khai báo `image:` có tag, và mọi `FROM` trong
#      Dockerfile phải pin tag/digest (`FROM <stage>` nội bộ được chấp nhận);
#   2. mọi service có healthcheck, và healthcheck không phải dạng giả
#      (`true`, `exit 0`, `:`, `|| exit 0`, `|| true`, `validate`, `print-config`,
#      `pgrep`, `kill -0`, `ps`).
#
# Dùng:  bash scripts/check-compose.sh [COMPOSE_FILE]
#   COMPOSE_FILE  mặc định: <gốc repo>/deploy/docker-compose.yml
#
# Cần: docker compose, jq. Exit 0 khi hợp lệ; exit 1 khi có vi phạm (mỗi vi
# phạm một dòng ra stderr); exit 2 khi không đọc được file compose.
set -euo pipefail

# image_pinned REF — 0 nếu REF có tag cụ thể (≠ latest) hoặc digest.
image_pinned() {
  local ref=$1
  [[ "$ref" == *@sha256:* ]] && return 0
  local last=${ref##*/}          # bỏ registry/namespace (registry có thể có :port)
  [[ "$last" == *:* ]] || return 1
  local tag=${last##*:}
  [[ -n "$tag" && "$tag" != "latest" ]]
}

# healthcheck_fake TEST — 0 nếu lệnh healthcheck (đã nối bằng dấu cách) là dạng giả.
healthcheck_fake() {
  local t=$1
  # Bỏ tiền tố CMD / CMD-SHELL.
  t=$(sed -E 's/^(CMD-SHELL|CMD)[[:space:]]+//' <<<"$t")
  t=$(sed -E 's/^[[:space:]]+|[[:space:]]+$//g' <<<"$t")
  [[ -z "$t" || "$t" == "NONE" ]] && return 0
  grep -qE '^(true|exit 0|:|/bin/true)$' <<<"$t" && return 0
  grep -qE '\|\|[[:space:]]*(true|exit 0|:)[[:space:]]*$' <<<"$t" && return 0
  grep -qE '(^|[[:space:]/])(validate|print-config|pgrep|pidof)([[:space:]]|$)|kill -0|(^|[[:space:]])ps([[:space:]]|$)' <<<"$t" && return 0
  return 1
}

# dockerfile_from_violations FILE — in các dòng FROM không pin (bỏ qua stage nội bộ).
dockerfile_from_violations() {
  local file=$1 stages=() line ref alias s internal
  while IFS= read -r line; do
    # FROM [--platform=...] <ref> [AS <name>]
    read -r -a parts <<<"$line"
    local i=1
    [[ "${parts[1]:-}" == --* ]] && i=2
    ref=${parts[$i]:-}
    alias=""
    if [[ "${parts[$((i + 1))]:-}" =~ ^[Aa][Ss]$ ]]; then
      alias=${parts[$((i + 2))]:-}
    fi
    internal=0
    for s in "${stages[@]+"${stages[@]}"}"; do
      [[ "$s" == "$ref" ]] && internal=1
    done
    if [[ $internal -eq 0 && "$ref" != "scratch" ]] && ! image_pinned "$ref"; then
      echo "$line"
    fi
    [[ -n "$alias" ]] && stages+=("$alias")
  done < <(grep -iE '^[[:space:]]*FROM[[:space:]]' "$file" | sed -E 's/^[[:space:]]+//')
}

main() {
  local file=${1:-}
  if [[ -z "$file" ]]; then
    local root
    root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
    file="$root/deploy/docker-compose.yml"
  fi
  local json
  if ! json=$(docker compose -f "$file" config --format json 2>/dev/null); then
    echo "check-compose: không đọc được compose: $file" >&2
    return 2
  fi

  local errors=0 svc image ctx dockerfile test v
  while IFS=$'\t' read -r svc image ctx dockerfile test; do
    [[ "$image" == "-" ]] && image=""
    if [[ "$ctx" == "-" ]]; then
      if [[ -z "$image" ]] || ! image_pinned "$image"; then
        echo "UNPINNED image: $svc ($image)" >&2
        errors=$((errors + 1))
      fi
    else
      if [[ -z "$image" ]] || ! image_pinned "$image"; then
        echo "UNPINNED build image: $svc phải khai báo image: có tag ($image)" >&2
        errors=$((errors + 1))
      fi
      local df="$dockerfile"
      [[ "$df" == /* ]] || df="$ctx/$df"
      if [[ ! -f "$df" ]]; then
        echo "MISSING Dockerfile: $svc ($df)" >&2
        errors=$((errors + 1))
      else
        while IFS= read -r v; do
          [[ -z "$v" ]] && continue
          echo "UNPINNED FROM: $svc: $v" >&2
          errors=$((errors + 1))
        done < <(dockerfile_from_violations "$df")
      fi
    fi
    if [[ "$test" == "-" ]]; then
      echo "MISSING healthcheck: $svc" >&2
      errors=$((errors + 1))
    elif healthcheck_fake "$test"; then
      echo "FAKE healthcheck: $svc ($test)" >&2
      errors=$((errors + 1))
    fi
  done < <(jq -r '.services | to_entries[] | [
      .key,
      (.value.image // "-"),
      (.value.build.context // "-"),
      (.value.build.dockerfile // "Dockerfile"),
      (if (.value.healthcheck.disable // false) then "-"
       else ((.value.healthcheck.test // null) | if . == null then "-" else join(" ") end) end)
    ] | @tsv' <<<"$json")

  if [[ $errors -gt 0 ]]; then
    echo "check-compose: $errors vi phạm" >&2
    return 1
  fi
  echo "check-compose: OK ($file)"
}

main "$@"
