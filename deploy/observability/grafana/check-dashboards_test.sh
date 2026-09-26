#!/usr/bin/env bash
# Chuỗi `${DS_...}`, backtick trong filter jq/pattern là dữ liệu test, cố ý không expand.
# shellcheck disable=SC2016
# check-dashboards_test.sh — test cho check-dashboards.sh (P0-T06).
# Mỗi ca dựng một thư mục grafana giả trong thư mục tạm (dashboards/ +
# provisioning/dashboards/) từ dashboard/provider thật của repo, biến đổi bằng
# jq/sed, rồi chạy check-dashboards.sh trên thư mục đó. Không cần Docker.
#
# Dùng:  bash deploy/observability/grafana/check-dashboards_test.sh
set -uo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
CHECK="$SCRIPT_DIR/check-dashboards.sh"
REAL_DASH="$SCRIPT_DIR/dashboards/red.json"
REAL_PROV="$SCRIPT_DIR/provisioning/dashboards/dashboards.yaml"

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/check-dashboards-test.XXXXXX")
trap 'rm -rf "$TMP_ROOT"' EXIT

pass=0
fail=0

# setup NAME — tạo thư mục ca NAME với bản sao dashboard/provider thật, in đường dẫn.
setup() {
  local dir="$TMP_ROOT/$1"
  mkdir -p "$dir/dashboards" "$dir/provisioning/dashboards"
  cp "$REAL_DASH" "$dir/dashboards/red.json"
  cp "$REAL_PROV" "$dir/provisioning/dashboards/dashboards.yaml"
  printf '%s\n' "$dir"
}

# jqi DIR FILTER — áp FILTER jq lên dashboards/red.json của DIR.
jqi() {
  local f="$1/dashboards/red.json"
  jq "$2" "$f" >"$f.tmp" && mv "$f.tmp" "$f"
}

# expect NAME DIR WANT_EXIT STDERR_PATTERN — chạy check trên DIR và so kết quả.
#   STDERR_PATTERN "-" = stderr phải rỗng.
expect() {
  local name=$1 dir=$2 want_exit=$3 pattern=$4 out err got_exit ok=1 why=""
  # stderr ghi vào TMP_ROOT (không ghi cạnh DIR: ca repo_default dùng thư mục thật của repo).
  out=$(bash "$CHECK" "$dir" 2>"$TMP_ROOT/$name.err")
  got_exit=$?
  err=$(cat "$TMP_ROOT/$name.err")
  [[ $got_exit -eq $want_exit ]] || { ok=0; why="exit=$got_exit, muốn $want_exit"; }
  if [[ $ok -eq 1 && "$pattern" == "-" ]]; then
    [[ -z "$err" ]] || { ok=0; why="stderr không rỗng: $err"; }
  elif [[ $ok -eq 1 ]]; then
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
expect repo_default "$SCRIPT_DIR" 0 -

d=$(setup valid_copy)
expect valid_copy "$d" 0 -

d=$(setup valid_id_absent)
jqi "$d" 'del(.id)'
expect valid_id_absent "$d" 0 -

d=$(setup valid_tempo_datasource)
jqi "$d" '.panels += [{"type":"traces","title":"t","datasource":{"type":"tempo","uid":"tempo"},"targets":[]}]'
expect valid_tempo_datasource "$d" 0 -

d=$(setup valid_yml_provider)
mv "$d/provisioning/dashboards/dashboards.yaml" "$d/provisioning/dashboards/dashboards.yml"
expect valid_yml_provider "$d" 0 -

d=$(setup valid_two_dashboards)
jq '.uid = "other-dash"' "$REAL_DASH" >"$d/dashboards/other.json"
expect valid_two_dashboards "$d" 0 -

# --- JSON / uid / id / title --------------------------------------------------
d=$(setup invalid_json)
printf '{"uid": "x",' >"$d/dashboards/red.json"
expect invalid_json "$d" 1 '^INVALID JSON: red.json'

d=$(setup uid_missing)
jqi "$d" 'del(.uid)'
expect uid_missing "$d" 1 '^BAD uid: red.json'

d=$(setup uid_empty)
jqi "$d" '.uid = ""'
expect uid_empty "$d" 1 '^BAD uid'

d=$(setup uid_bad_chars)
jqi "$d" '.uid = "snaptix red/x"'
expect uid_bad_chars "$d" 1 '^BAD uid'

d=$(setup uid_too_long)
jqi "$d" '.uid = "a234567890b234567890c234567890d234567890e"'
expect uid_too_long "$d" 1 '^BAD uid'

d=$(setup uid_40_chars)
jqi "$d" '.uid = "a234567890b234567890c234567890d234567890"'
expect uid_40_chars "$d" 0 -

d=$(setup uid_duplicate)
cp "$d/dashboards/red.json" "$d/dashboards/red-copy.json"
expect uid_duplicate "$d" 1 '^DUPLICATE uid: snaptix-red'

d=$(setup id_set)
jqi "$d" '.id = 12'
expect id_set "$d" 1 '^BAD id: red.json'

d=$(setup title_empty)
jqi "$d" '.title = ""'
expect title_empty "$d" 1 '^MISSING title'

# --- Dạng export / datasource -------------------------------------------------
d=$(setup export_requires)
jqi "$d" '.__requires = [{"type":"datasource","id":"prometheus","name":"Prometheus"}]'
expect export_requires "$d" 1 '^EXPORT FORMAT: red.json có khoá __requires'

d=$(setup export_elements)
jqi "$d" '.__elements = {}'
expect export_elements "$d" 1 '^EXPORT FORMAT: red.json có khoá __elements'

d=$(setup export_ds_placeholder)
jqi "$d" '.panels[1].datasource = {"type":"prometheus","uid":"${DS_PROM}"}'
expect export_ds_placeholder "$d" 1 'EXPORT FORMAT: red.json dùng \$\{DS_PROM\}'

d=$(setup ds_string_name)
jqi "$d" '.panels[1].datasource = "Prometheus"'
expect ds_string_name "$d" 1 '^BAD datasource: red.json "Prometheus"'

d=$(setup ds_wrong_uid)
jqi "$d" '.panels[1].targets[0].datasource.uid = "P1809F7CD0C75ACF3"'
expect ds_wrong_uid "$d" 1 'BAD datasource: .*P1809F7CD0C75ACF3'

d=$(setup ds_type_uid_mismatch)
jqi "$d" '.panels[1].datasource = {"type":"tempo","uid":"prometheus"}'
expect ds_type_uid_mismatch "$d" 1 '^BAD datasource'

d=$(setup ds_variable_datasource)
jqi "$d" '.templating.list[0].datasource = {"type":"prometheus","uid":"${datasource}"}'
expect ds_variable_datasource "$d" 1 '^BAD datasource'

d=$(setup expr_empty)
jqi "$d" '.panels[1].targets[0].expr = "  "'
expect expr_empty "$d" 1 '^EMPTY expr: red.json \(1 target\)'

d=$(setup many_errors)
jqi "$d" '.id = 1 | .uid = "" | .title = ""'
expect many_errors "$d" 1 'check-dashboards: 3 vi phạm'

# --- Provider -----------------------------------------------------------------
d=$(setup provider_allow_ui_updates)
sed -i.bak 's/allowUiUpdates: false/allowUiUpdates: true/' "$d/provisioning/dashboards/dashboards.yaml"
expect provider_allow_ui_updates "$d" 1 '^PROVIDER: dashboards.yaml thiếu `allowUiUpdates: false`'
rm -f "$d/provisioning/dashboards/dashboards.yaml.bak"

d=$(setup provider_no_disable_deletion)
sed -i.bak '/disableDeletion/d' "$d/provisioning/dashboards/dashboards.yaml"
rm -f "$d/provisioning/dashboards/dashboards.yaml.bak"
expect provider_no_disable_deletion "$d" 1 'thiếu `disableDeletion: true`'

d=$(setup provider_commented_out)
sed -i.bak 's/^\([[:space:]]*\)allowUiUpdates: false/\1# allowUiUpdates: false/' "$d/provisioning/dashboards/dashboards.yaml"
rm -f "$d/provisioning/dashboards/dashboards.yaml.bak"
expect provider_commented_out "$d" 1 'thiếu `allowUiUpdates: false`'

# --- Thiếu file ---------------------------------------------------------------
d=$(setup no_dashboard)
rm "$d/dashboards/red.json"
expect no_dashboard "$d" 2 'không có dashboard'

d=$(setup no_provider)
rm "$d/provisioning/dashboards/dashboards.yaml"
expect no_provider "$d" 2 'không có dashboard .* hoặc provider'

echo
echo "check-dashboards_test: $pass pass, $fail fail"
[[ $fail -eq 0 ]]
