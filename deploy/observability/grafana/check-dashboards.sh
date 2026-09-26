#!/usr/bin/env bash
# check-dashboards.sh — kiểm tra tĩnh dashboard Grafana provisioning (P0-T06).
#
# Với mỗi file <GRAFANA_DIR>/dashboards/*.json:
#   1. JSON hợp lệ;
#   2. `uid` cố định khớp ^[A-Za-z0-9_-]{1,40}$, không trùng giữa các file;
#      `id` là null hoặc không có; `title` không rỗng;
#   3. không phải dạng export để import tay ("Export for sharing externally"):
#      không có khoá cấp đầu bắt đầu bằng `__` (khoá khai báo input/requires của
#      bản export), không có placeholder `${DS_...}`;
#   4. mọi `datasource` là object {type, uid} trỏ tới datasource provision sẵn
#      (prometheus/prometheus, tempo/tempo) hoặc annotation mặc định
#      grafana/"-- Grafana --"; không dùng tên datasource dạng chuỗi;
#   5. mọi target có `expr` thì `expr` không rỗng.
# Với mỗi provider <GRAFANA_DIR>/provisioning/dashboards/*.y*ml:
#   6. có `apiVersion: 1`, `type: file`, `disableDeletion: true`, `allowUiUpdates: false`.
#
# Dùng:  bash deploy/observability/grafana/check-dashboards.sh [GRAFANA_DIR]
#   GRAFANA_DIR  mặc định: thư mục chứa script này.
#
# Cần: jq. Exit 0 khi hợp lệ; exit 1 khi có vi phạm (mỗi vi phạm một dòng ra
# stderr); exit 2 khi GRAFANA_DIR không có dashboard hoặc provider.
set -euo pipefail

UID_RE='^[A-Za-z0-9_-]{1,40}$'

# check_dashboard FILE — in vi phạm của một file dashboard (mỗi dòng một lỗi).
check_dashboard() {
  local f=$1 name
  name=${f##*/}
  if ! jq -e . "$f" >/dev/null 2>&1; then
    echo "INVALID JSON: $name"
    return 0
  fi
  jq -r --arg name "$name" --arg re "$UID_RE" '
    def ds_ok: type == "object" and
      ([.type, .uid] | IN(["prometheus", "prometheus"], ["tempo", "tempo"], ["grafana", "-- Grafana --"]));
    (if (.uid | type) != "string" or (.uid | test($re) | not)
       then "BAD uid: \($name) (\(.uid | tojson))" else empty end),
    (if .id != null then "BAD id: \($name) (id phải null hoặc không có, đang là \(.id | tojson))" else empty end),
    (if (.title | type) != "string" or (.title | length) == 0
       then "MISSING title: \($name)" else empty end),
    ([keys[] | select(startswith("__"))][] | "EXPORT FORMAT: \($name) có khoá \(.) (dạng export để import tay)"),
    ([.. | strings | select(test("\\$\\{DS_"))] | unique[] | "EXPORT FORMAT: \($name) dùng \(.)"),
    ([.. | objects | select(has("datasource")) | .datasource | select(ds_ok | not)]
       | unique[] | "BAD datasource: \($name) \(tojson)"),
    ([.. | objects | select(has("expr")) | select((.expr | type) != "string" or (.expr | test("^\\s*$")))]
       | length | select(. > 0) | "EMPTY expr: \($name) (\(.) target)")
  ' "$f"
}

# check_provider FILE — in vi phạm của một file provider.
check_provider() {
  local f=$1 name key
  name=${f##*/}
  for key in 'apiVersion: 1' 'type: file' 'disableDeletion: true' 'allowUiUpdates: false'; do
    grep -qE "^[[:space:]]*(- )?${key}[[:space:]]*(#.*)?$" "$f" || echo "PROVIDER: $name thiếu \`$key\`"
  done
}

main() {
  local dir=${1:-}
  [[ -n "$dir" ]] || dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

  local dashboards=() providers=() f
  for f in "$dir"/dashboards/*.json; do
    [[ -e "$f" ]] && dashboards+=("$f")
  done
  for f in "$dir"/provisioning/dashboards/*.yaml "$dir"/provisioning/dashboards/*.yml; do
    [[ -e "$f" ]] && providers+=("$f")
  done
  if [[ ${#dashboards[@]} -eq 0 || ${#providers[@]} -eq 0 ]]; then
    echo "check-dashboards: không có dashboard (${dir}/dashboards/*.json) hoặc provider (${dir}/provisioning/dashboards/*.y*ml)" >&2
    return 2
  fi

  local errors=() line
  for f in "${dashboards[@]}"; do
    while IFS= read -r line; do
      [[ -n "$line" ]] && errors+=("$line")
    done < <(check_dashboard "$f")
  done

  # uid không trùng giữa các file (Grafana chỉ nạp một trong các bản trùng).
  while IFS= read -r line; do
    [[ -n "$line" ]] && errors+=("DUPLICATE uid: $line")
  done < <(for f in "${dashboards[@]}"; do jq -r '.uid // empty' "$f" 2>/dev/null || true; done | sort | uniq -d)

  for f in "${providers[@]}"; do
    while IFS= read -r line; do
      [[ -n "$line" ]] && errors+=("$line")
    done < <(check_provider "$f")
  done

  if [[ ${#errors[@]} -gt 0 ]]; then
    printf '%s\n' "${errors[@]}" >&2
    echo "check-dashboards: ${#errors[@]} vi phạm" >&2
    return 1
  fi
  echo "check-dashboards: OK (${#dashboards[@]} dashboard, ${#providers[@]} provider trong $dir)"
}

main "$@"
