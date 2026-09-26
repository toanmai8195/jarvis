# Test cases — P0-T06: Dashboard Grafana cơ bản: RED metrics cho mỗi service

> **Phạm vi**: một dashboard Grafana RED (Rate, Errors, Duration) được **provisioning** từ file trong `deploy/observability/grafana/`, cùng file provider trong `deploy/observability/grafana/provisioning/dashboards/`. Căn cứ là các docs sau:
> - Phase 0 README: dòng P0-T06 "Dashboard Grafana cơ bản: RED metrics cho mỗi service". Mốc demo: "trace hiển thị trên Grafana". Challenge **G10** (Observability): "OpenTelemetry trace, slog JSON, Prometheus metrics".
> - [architecture](../../../../architecture.md) mục Observability: "Metrics: Prometheus — RED metrics cho mỗi endpoint", "Dashboard: Grafana (datasource Prometheus + Tempo provision sẵn)", luồng "app → OTLP (4317/4318) → otel-collector → … Prometheus (metric) → Grafana", "Metric vào Prometheus bằng OTLP push (`--web.enable-otlp-receiver`)", "Prometheus … scrape … `/metrics` của service (từ P0-T07)".
> - [execute-all-notes](../../../execute-all-notes.md) mục P0-T02: "P0-T06 (dashboard): thêm `deploy/observability/grafana/provisioning/dashboards/` (provider + JSON). Image grafana COPY cả thư mục `provisioning`, nên chỉ cần `up -d --wait` là có. Datasource uid cố định: `prometheus`, `tempo`". Mục P0-T02: không bind mount repo vào container, vì Docker Desktop treo khi mount thư mục trong `~/Documents`.
> - [local-setup](../../../../local-setup.md): Grafana cổng **3100** (ẩn danh, quyền Admin), OTLP HTTP **4318**, Prometheus **9090**. `deploy/observability/grafana/provisioning/datasources/datasources.yaml`: datasource `uid: prometheus` và `uid: tempo`, `editable: false`.
>
> **Giả định** (reviewer xem kỹ):
> - A1. **Chưa có service thật**: core/bff chỉ có từ P0-T07/P0-T11, và instrumentation OTel có từ P0-T08/P0-T10/P0-T12. Vì vậy dashboard phải **tổng quát theo biến service**, không hard-code tên service. Test dùng **metric giả** đẩy qua OTLP HTTP (cách của P0-T02-TC14) cho 2 service giả `p0t06-alpha`, `p0t06-beta`.
> - A2. **Tên metric**: theo OTel HTTP semantic conventions bản stable. Metric là histogram `http.server.request.duration`, đơn vị `s`, với attribute `http.request.method`, `http.route`, `http.response.status_code`. Prometheus 3.15 dịch tên theo mặc định (UnderscoreEscapingWithSuffixes) thành `http_server_request_duration_seconds_{bucket,count,sum}`, label `http_request_method`, `http_route`, `http_response_status_code`. Người viết TC đã kiểm chứng trên stack P0-T02. Metric semconv cũ `http.server.duration` (ms) **không** thuộc phạm vi. Nếu P0-T08/P0-T12 dùng tên khác thì task đó sửa dashboard.
> - A3. **Label service là `job`**: Prometheus đặt `job` = `<service.namespace>/<service.name>` khi push OTLP (vd `snaptix/p0t06-alpha`, đã kiểm chứng), còn khi scrape `/metrics` (P0-T07) thì `job` là tên scrape job. Không có label `service_name` trên series metric (chỉ có trong `target_info`). Vì vậy biến service lấy từ label `job` của **chính metric RED**: `label_values(http_server_request_duration_seconds_count, job)` (hoặc dạng tương đương có giới hạn theo metric). Tên biến do người làm đặt (vd `service`). Biến là loại `query`, `multi: true`, `includeAll: true`. Query lọc bằng `job=~"$<biến>"`. Biến **không** được liệt kê job không có metric RED (`prometheus`, `otel-collector`).
> - A4. **Định nghĩa RED** (docs không định nghĩa chi tiết, đây là lựa chọn theo quy ước RED phổ biến):
>   - **Rate**: số request/giây theo service, `sum by (job) (rate(..._count[...]))`, đơn vị `reqps`.
>   - **Errors**: tỉ lệ request có `http_response_status_code` **5xx** trên tổng request theo service. 4xx **không** tính là lỗi. Đơn vị `percentunit` (giá trị 0–1) hoặc `percent` (0–100). Service có request nhưng không có 5xx phải ra **0**, không bỏ trống.
>   - **Duration**: p50, p95, p99 bằng `histogram_quantile(q, sum by (job, le) (rate(..._bucket[...])))`, đơn vị `s` (hoặc `ms` nếu nhân 1000).
>   - Dùng `$__rate_interval` (hoặc cửa sổ ≥ 1m) cho `rate`.
>   - Theo architecture ("RED cho mỗi endpoint"), có **ít nhất một** panel Rate tách theo `http_route` của service đang chọn.
>   - Tiêu đề (row hoặc panel) của ba nhóm chứa lần lượt `Rate`, `Error`, `Duration` (không phân biệt hoa thường) để nhận diện.
> - A5. **Provisioning**: file JSON dashboard nằm trong `deploy/observability/grafana/` và được Dockerfile COPY vào image. Không bind mount (execute-all-notes P0-T02). Provider khai báo `disableDeletion: true`, `allowUiUpdates: false`: dashboard không lưu đè và không xoá được qua API/UI. `uid` cố định trong JSON (`^[A-Za-z0-9_-]{1,40}$`), `id` là `null` hoặc không có. Datasource tham chiếu bằng `{"type":"prometheus","uid":"prometheus"}`, không dùng `__inputs`/`${DS_...}` (dạng export để import tay). Chỉ đổi thư mục `deploy/observability/grafana/**`. Không đổi service khác trong compose, `prometheus.yml` hay config collector.
> - A6. **Số liệu kỳ vọng** (generator đẩy histogram cumulative mỗi 5 s, bucket OTel mặc định `[0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10]`). Các số dưới đây là giá trị nội suy tuyến tính của `histogram_quantile`, người viết TC đã đo lại trên stack thật. Sai số cho phép ±5 %. Chỉ so sau khi generator đã chạy ≥ 90 s.
>
>   | Service (`job`) | Series (method route status: req/s, latency) | Rate | Errors | p50 | p95 | p99 |
>   |---|---|---|---|---|---|---|
>   | `snaptix/p0t06-alpha` | GET /v1/trips 200: 8, 0.02 s · GET /v1/trips **503**: 1, 0.3 s · GET /v1/seats 200: 1, 0.02 s · GET /v1/seats **404**: 1, 0.02 s | 11 | 1/11 = **0.0909** (nếu tính cả 4xx sẽ ra 0.1818 → sai) | 0.01825 | 0.3625 | 0.4725 |
>   | `snaptix/p0t06-beta` | POST /v1/bookings 201: 4, 0.2 s | 4 | **0** | 0.175 | 0.2425 | 0.2485 |
>
>   Theo route của alpha: `/v1/trips` 9 req/s, `/v1/seats` 2 req/s.
>
> **Không thuộc task này**: instrumentation metric trong core/bff (P0-T08, P0-T10, P0-T12), target scrape `/metrics` của core (P0-T07+), dashboard nghiệp vụ (số ghế hold, độ trễ outbox, pool PG, là metric của phase sau), alerting, `metrics_generator`/service graph của Tempo, trace bff → core (P0-AT07).
>
> **Môi trường khi viết**: macOS arm64, Docker 29.8, stack P0-T02 (Grafana 13.2.2, Prometheus v3.15.0, otelcol-contrib 0.161.0), python3, jq, curl. Chrome của người dùng có extension Claude in Chrome.
>
> **Chuẩn bị**: Bash tool mở shell mới mỗi lần gọi, nên lưu khối dưới thành file `$S/prep.sh` một lần (thay `<scratchpad>` bằng đường dẫn thật; tạo bằng Write tool). Mỗi lần gọi Bash bắt đầu bằng `source <scratchpad>/p0t06/prep.sh`. Hai file `gen.py` và `dsq.py` bên dưới cũng lưu vào `$S` một lần. Phần chuẩn bị và các TC **không** có lệnh ghi lịch sử git nào (hook `guard-bash` của execute-all chặn).
> ```bash
> REPO=/Users/maiductoan/Documents/code/jarvis; cd $REPO
> S=<scratchpad>/p0t06; mkdir -p $S
> DC="docker compose -f deploy/docker-compose.yml"
> G=http://localhost:3100
> GDIR=deploy/observability/grafana
> DASH=$(grep -rl 'http_server_request_duration_seconds' $GDIR --include='*.json')   # đúng 1 file (TC01)
> PROV=$(ls $GDIR/provisioning/dashboards/*.y*ml 2>/dev/null)
> UID_=$(jq -r .uid "$DASH"); VAR=$(jq -r '[.templating.list[]|select(.type=="query")][0].name' "$DASH")
> ALL='snaptix/p0t06-alpha|snaptix/p0t06-beta'
> genstart() { genstop; nohup python3 $S/gen.py 1500 > $S/gen.log 2>&1 & echo $! > $S/gen.pid; }
> genstop()  { [ -f $S/gen.pid ] && kill "$(cat $S/gen.pid)" 2>/dev/null; rm -f $S/gen.pid; true; }
> genage()   { echo $(( $(date +%s) - $(stat -f %m $S/gen.pid) )); }       # số giây generator đã chạy
> near() { python3 -c 'import sys;a,b=map(float,sys.argv[1:]);sys.exit(0 if abs(a-b)<=0.05*abs(b)+1e-9 else 1)' "$1" "$2"; }
> dsq() { python3 $S/dsq.py "$DASH" "$@"; }        # dsq <giá trị biến> [regex tiêu đề] | dsq --var
> RATE_RE='^(?!.*(error|lỗi|duration|latency|p50|p95|p99)).*(rate|request|req|throughput|traffic|route)'   # tiêu đề panel nhóm Rate, loại Error/Duration
> ```
> `$S/gen.py` (đẩy metric giả theo bảng A6, cumulative, mỗi 5 s, trong `argv[1]` giây):
> ```python
> import json, sys, time, urllib.request
> B=[0.005,0.01,0.025,0.05,0.075,0.1,0.25,0.5,0.75,1,2.5,5,7.5,10]
> SPEC={"p0t06-alpha":[("/v1/trips","GET",200,8,0.02),("/v1/trips","GET",503,1,0.3),
>                      ("/v1/seats","GET",200,1,0.02),("/v1/seats","GET",404,1,0.02)],
>       "p0t06-beta":[("/v1/bookings","POST",201,4,0.2)]}
> def bc(n,lat):
>     c=[0]*(len(B)+1); c[next((k for k,b in enumerate(B) if lat<=b),len(B))]=n; return [str(x) for x in c]
> def push(s0,now,el):
>     rms=[]
>     for svc,ser in SPEC.items():
>         dps=[{"startTimeUnixNano":str(s0),"timeUnixNano":str(now),"count":str(int(r*el)),"sum":int(r*el)*lat,
>               "bucketCounts":bc(int(r*el),lat),"explicitBounds":B,
>               "attributes":[{"key":"http.route","value":{"stringValue":rt}},
>                             {"key":"http.request.method","value":{"stringValue":m}},
>                             {"key":"http.response.status_code","value":{"intValue":str(st)}}]}
>              for rt,m,st,r,lat in ser]
>         rms.append({"resource":{"attributes":[{"key":"service.name","value":{"stringValue":svc}},
>                                               {"key":"service.namespace","value":{"stringValue":"snaptix"}}]},
>                     "scopeMetrics":[{"scope":{"name":"p0t06-gen"},"metrics":[{"name":"http.server.request.duration",
>                       "unit":"s","histogram":{"aggregationTemporality":2,"dataPoints":dps}}]}]})
>     req=urllib.request.Request("http://localhost:4318/v1/metrics",data=json.dumps({"resourceMetrics":rms}).encode(),
>                                headers={"Content-Type":"application/json"})
>     print(int(el), urllib.request.urlopen(req).status, flush=True)
> dur=int(sys.argv[1]); t0=time.time(); s0=int(t0*1e9)
> while True:
>     el=time.time()-t0; push(s0,int(time.time()*1e9),el)
>     if el>=dur: break
>     time.sleep(5)
> ```
> `$S/dsq.py` (lấy `expr` của mọi target trong mọi panel, kể cả panel lồng trong row, thay biến service bằng giá trị cho sẵn rồi gửi Grafana `/api/ds/query` dạng instant; Grafana tự thay `$__rate_interval`/`$__interval`. In `tiêu đề | refId | labels | giá trị`, hoặc `| NO DATA` / `| ERROR ...`. `--var` in giá trị biến service mà Grafana tính từ query `label_values(metric, label)` qua resource API của datasource):
> ```python
> import json, re, sys, urllib.request, urllib.parse
> G="http://localhost:3100"; dash=json.load(open(sys.argv[1]))
> v=[x for x in dash["templating"]["list"] if x.get("type")=="query"][0]; var=v["name"]
> if sys.argv[2]=="--var":
>     q=v.get("definition") or (v["query"]["query"] if isinstance(v["query"],dict) else v["query"])
>     m=re.fullmatch(r"\s*label_values\((.+),\s*(\w+)\)\s*",q)
>     if not m: sys.exit("UNSUPPORTED "+q)
>     u=f'{G}/api/datasources/uid/prometheus/resources/api/v1/label/{m[2]}/values?'+urllib.parse.urlencode({"match[]":m[1]})
>     print("\n".join(json.load(urllib.request.urlopen(u))["data"])); sys.exit()
> val=sys.argv[2]; pat=re.compile(sys.argv[3],re.I) if len(sys.argv)>3 else None
> def walk(ps):
>     for p in ps: yield p; yield from walk(p.get("panels",[]))
> def sub(e):
>     e=re.sub(r"\$\{"+var+r"(:[a-z]+)?\}",val,e); e=re.sub(r"\[\["+var+r"\]\]",val,e)
>     return re.sub(r"\$"+var+r"\b",val,e)
> for p in walk(dash.get("panels",[])):
>     if pat and not pat.search(p.get("title","")): continue
>     for t in p.get("targets",[]):
>         if "expr" not in t: continue
>         rid=t.get("refId","A"); body={"from":"now-5m","to":"now","queries":[{"refId":rid,
>             "datasource":{"type":"prometheus","uid":"prometheus"},"expr":sub(t["expr"]),"instant":True,"range":False}]}
>         req=urllib.request.Request(G+"/api/ds/query",data=json.dumps(body).encode(),headers={"Content-Type":"application/json"})
>         try: r=json.load(urllib.request.urlopen(req))["results"][rid]
>         except urllib.error.HTTPError as ex: print(f'{p["title"]} | {rid} | ERROR HTTP {ex.code} {ex.read()[:200]}'); continue
>         if r.get("error"): print(f'{p["title"]} | {rid} | ERROR {r["error"]}'); continue
>         n=0
>         for f in r.get("frames",[]):
>             vs=f["data"]["values"]
>             if len(vs)<2 or not vs[1]: continue
>             print(f'{p["title"]} | {rid} | {json.dumps(f["schema"]["fields"][1].get("labels") or {},sort_keys=True)} | {vs[1][-1]}'); n+=1
>         if n==0: print(f'{p["title"]} | {rid} | NO DATA')
> ```
> **UI (Claude in Chrome)**: dùng Chrome đang mở của người dùng qua `mcp__claude-in-chrome__*`. Mở **tab mới** (`tabs_create_mcp`), chỉ điều hướng tới `http://localhost:3100/...`, chụp màn hình làm bằng chứng, và đóng tab khi xong (`tabs_close_mcp`). Không đụng tab khác của người dùng.

| ID | Loại | Kịch bản | Kết quả mong đợi | Trạng thái |
|---|---|---|---|---|
| P0-T06-TC01 | Cấu trúc | File dashboard và provider tồn tại, JSON hợp lệ: `echo "$DASH" \| wc -l`; `jq -e . "$DASH" >/dev/null; echo $?`; `jq -r '[.uid, (.id\|tostring), .title] \| @tsv' "$DASH"`; `echo "$PROV"`; `grep -rn '__inputs\|DS_PROMETHEUS' $GDIR` | Đúng **1** file JSON trong `deploy/observability/grafana/` chứa metric RED. `jq` exit `0`. `uid` khớp `^[A-Za-z0-9_-]{1,40}$` (cố định, không rỗng). `id` là `null`. `title` không rỗng. Có đúng 1 file provider `.yaml`/`.yml` trong `provisioning/dashboards/`. `grep` không khớp (không phải dạng export `__inputs`) | ✅ |
| P0-T06-TC02 | Cấu trúc | Provider đúng cấu hình và file có trong image (không bind mount): `cat "$PROV"`; `$DC config --format json \| jq '.services.grafana.volumes'`; sau khi stack chạy (TC06): `P=$(yq -r '.providers[0].options.path' "$PROV" 2>/dev/null \|\| grep -E '^\s*path:' "$PROV" \| awk '{print $2}' \| tr -d "\"'")`; `echo "$P"`; `$DC exec -T grafana sh -c "ls -R $P"`; so `shasum` của file JSON trong container với `$DASH` | Provider có `apiVersion: 1`, `type: file`, `disableDeletion: true`, `allowUiUpdates: false`, `options.path` trỏ thư mục trong `/etc/grafana/...`; `echo "$P"` in đường dẫn **không** có dấu nháy `"`/`'` (dù YAML viết `path: "..."` hay `path: '...'`). `volumes` của grafana là `null` (không bind mount). Trong container có file JSON, `shasum` giống hệt file trong repo | ✅ |
| P0-T06-TC03 | Cấu trúc | Datasource tham chiếu đúng uid: `jq -c '[.. \| objects \| select(has("datasource")) \| .datasource] \| unique' "$DASH"`; `jq -r '[.. \| objects \| select(has("expr"))] \| length' "$DASH"`; `curl -s -o /dev/null -w '%{http_code}' $G/api/datasources/uid/prometheus` (khi stack chạy) | Mọi `datasource` là `{"type":"prometheus","uid":"prometheus"}`, trừ annotation mặc định dùng `{"type":"grafana","uid":"-- Grafana --"}` (nếu có). Không có tên datasource dạng chuỗi, `${DS_...}` hay uid khác. Có ≥ 5 target có `expr`. API trả `200` | ✅ |
| P0-T06-TC04 | Cấu trúc | Query và biến tổng quát theo service (A2, A3): `jq '.templating.list[] \| select(.type=="query") \| {name, multi, includeAll, datasource, definition, query}' "$DASH"`; `jq -r '.. \| objects \| select(has("expr")) \| .expr' "$DASH"`; `grep -cE 'p0t06\|alpha\|beta\|"core"\|"bff"' "$DASH"` | Có biến loại `query`, datasource uid `prometheus`, query `label_values(http_server_request_duration_seconds_count, job)` (hoặc tương đương giới hạn theo metric này), `multi: true`, `includeAll: true`. Mọi `expr` chỉ dùng họ metric `http_server_request_duration_seconds_(count\|bucket\|sum)` và có bộ lọc `job=~"$<biến>"` (hoặc `${<biến>}`/`${<biến>:regex}`). `grep` trả `0` (không hard-code tên service) | ✅ |
| P0-T06-TC05 | Cấu trúc | Đủ 3 nhóm R/E/D, đúng đơn vị và phân vị (A4): `jq -r '.. \| objects \| select(has("title") and has("type")) \| "\(.type)\t\(.title)\t\(.fieldConfig.defaults.unit // "-")"' "$DASH"`; `jq -r '.. \| objects \| select(has("expr")) \| .expr' "$DASH" \| grep -oE 'histogram_quantile\(0\.[0-9]+' \| sort -u`; `jq -r '.. \| objects \| select(has("expr")) \| .expr' "$DASH" \| grep -c 'http_route'` | Có tiêu đề (row hoặc panel) chứa `Rate`, `Error`, `Duration` (không phân biệt hoa thường). Panel Rate có unit `reqps`, panel Errors có `percentunit` hoặc `percent`, panel Duration có `s` hoặc `ms`. Có đủ `histogram_quantile(0.5`, `(0.95`, `(0.99`. Error dùng `http_response_status_code=~"5.."` (hoặc tương đương chỉ 5xx). Có ≥ 1 query theo `http_route` | ✅ |
| P0-T06-TC06 | Provisioning | Stack mới hiện dashboard: `$DC down -v; $DC up -d --wait` (tự build image grafana, `pull_policy: build`); `curl -s "$G/api/search?type=dash-db" \| jq -c '[.[] \| {uid, title}]'`; `curl -s $G/api/dashboards/uid/$UID_ \| jq -c '{p: .meta.provisioned, t: .dashboard.title, n: ([.dashboard.panels[]?, .dashboard.panels[]?.panels[]?] \| length)}'` | `up` exit `0`, mọi container healthy. `/api/search` có dashboard với `uid` = `$UID_`, title giống file. `meta.provisioned` = `true`. Số panel bằng số panel trong file (`jq '[.panels[]?, .panels[]?.panels[]?] \| length' "$DASH"`) | ✅ |
| P0-T06-TC07 | Provisioning | Không lưu đè, không xoá được: `curl -s $G/api/dashboards/uid/$UID_ \| jq '.dashboard.title="HACKED" \| {dashboard, overwrite: true}' > $S/save.json`; `curl -s -w ' %{http_code}' -X POST $G/api/dashboards/db -H 'Content-Type: application/json' -d @$S/save.json`; `curl -s -w ' %{http_code}' -X DELETE $G/api/dashboards/uid/$UID_`; `curl -s $G/api/dashboards/uid/$UID_ \| jq -r .dashboard.title` | POST lưu và DELETE đều trả mã **4xx**, thông báo nêu dashboard được provisioning. Title vẫn là title gốc, không phải `HACKED`. Dashboard vẫn còn | ✅ |
| P0-T06-TC08 | Provisioning | Log Grafana không báo lỗi provisioning (chỉ xét log của `logger=provisioning*`): `$DC logs grafana \| grep -E 'logger=provisioning[.a-z]*' \| grep -iE 'level=(error\|warn)'`; `$DC logs grafana \| grep -iE 'provisioning.dashboard' \| tail -3` | Lệnh đầu không có dòng nào (không có `level=error`/`level=warn` nào từ logger `provisioning`, `provisioning.dashboard`, `provisioning.datasources`...). Cảnh báo có sẵn của Grafana 13.2.2 từ `logger=migrator` (vd `Skipping migration ... drop index UQE_dashboard_public_config_uid`) **không** thuộc phạm vi, không tính là fail. Có log provisioning dashboard khởi động bình thường (vd "starting to provision dashboards", "finished to provision dashboards") | ✅ |
| P0-T06-TC09 | No data | Stack mới, **chưa** có metric RED (chạy ngay sau TC06, trước `genstart`): `dsq --var`; `dsq '.*'`; `dsq '.*' \| grep -c ERROR` | `dsq --var` không in dòng nào (biến rỗng, không có `prometheus`/`otel-collector`). Mọi target in `NO DATA`. Không có dòng `ERROR` (đếm `0`) | ✅ |
| P0-T06-TC10 | Biến | Biến service liệt kê đúng các service có metric: `genstart`; chờ đến khi `genage` ≥ 30; `tail -2 $S/gen.log`; `dsq --var \| sort` | Log generator in HTTP `200`. Biến trả đúng 2 giá trị `snaptix/p0t06-alpha`, `snaptix/p0t06-beta`, **không** có `prometheus`, `otel-collector` | ✅ |
| P0-T06-TC11 | Rate | Rate theo service với "All": chờ `genage` ≥ 90; `dsq "$ALL" "$RATE_RE" \| grep -v http_route`. `RATE_RE` chấp nhận tiêu đề nhóm Rate theo A4 (vd "Rate", "Request rate", "Requests/s", "Throughput") và loại tiêu đề chứa `error`/`duration`/`latency`/phân vị, nên panel "Error rate" không lọt vào. Nếu panel Rate nằm trong row "Rate" mà tiêu đề panel không khớp `RATE_RE`: xác định panel có `fieldConfig.defaults.unit == "reqps"` bằng `jq`, rồi dùng đúng tiêu đề đó (neo `^...$`) làm regex | Chỉ in panel nhóm Rate (unit `reqps`), không lẫn giá trị error (~0.09) hay duration. Mỗi service một series (label `job`). Panel Rate tổng: alpha ≈ **11**, beta ≈ **4** req/s (±5 %, kiểm bằng `near`). Không `ERROR` | ✅ |
| P0-T06-TC12 | Rate / endpoint | Rate theo route của một service (A4, "RED cho mỗi endpoint"), chỉ xét panel nhóm Rate: `dsq 'snaptix/p0t06-alpha' "$RATE_RE" \| grep http_route` (cùng cách xác định panel Rate như TC11; panel Error/Duration theo route, nếu có, không thuộc TC này) | Có panel Rate (unit `reqps`) trả series theo `http_route`: `/v1/trips` ≈ **9**, `/v1/seats` ≈ **2** req/s (±5 %). Không có series của beta | ✅ |
| P0-T06-TC13 | Errors | Error rate chỉ tính 5xx: `dsq "$ALL" 'error'` | alpha ≈ **0.0909** (hoặc **9.09** nếu đơn vị `percent`), ±5 %, và **không** ≈ 0.1818/18.18 (4xx không tính). beta = **0** (có series, không bỏ trống). Không `ERROR` | ✅ |
| P0-T06-TC14 | Duration | p50/p95/p99 theo service: `dsq "$ALL" 'duration\|latency'` | Có 3 phân vị cho mỗi service. Giá trị (giây, hoặc ×1000 nếu `ms`), ±5 %: alpha p50 **0.01825**, p95 **0.3625**, p99 **0.4725**; beta p50 **0.175**, p95 **0.2425**, p99 **0.2485**. Không `ERROR` | ✅ |
| P0-T06-TC15 | Biến | Lọc theo một service và service không tồn tại: `dsq 'snaptix/p0t06-beta' \| grep -c alpha`; `dsq 'snaptix/p0t06-beta' \| grep -c 'NO DATA\|ERROR'`; `dsq 'snaptix/p0t06-none'` | Chọn beta: không series nào của alpha (`0`), mọi panel R/E/D có dữ liệu beta (không `NO DATA`, không `ERROR`; riêng panel theo route nếu tách query thì cũng chỉ có `/v1/bookings`). Service không tồn tại: mọi target `NO DATA`, không `ERROR` | ✅ |
| P0-T06-TC16 | UI | Mở dashboard trên Chrome, "All" (generator đang chạy, `genage` ≥ 90): tab mới → `http://localhost:3100/d/<UID_>?var-<VAR>=$__all&from=now-15m&to=now`; chụp màn hình; mở dropdown biến service, chụp màn hình; `read_page`/`get_page_text` để đọc tiêu đề panel và legend | Dashboard tải xong, không có thông báo lỗi. Thấy 3 nhóm **Rate**, **Errors**, **Duration** đều có đường/giá trị cho **cả** `p0t06-alpha` và `p0t06-beta` (legend có cả hai). Dropdown biến chỉ có `All`, `snaptix/p0t06-alpha`, `snaptix/p0t06-beta`. Không panel nào có biểu tượng lỗi (tam giác đỏ). Ảnh chụp đính kèm làm bằng chứng | ✅ |
| P0-T06-TC17 | UI | "No data" không lỗi: cùng tab, điều hướng `http://localhost:3100/d/<UID_>?var-<VAR>=$__all&from=now-7d&to=now-6d`; chụp màn hình; rồi đóng tab (`tabs_close_mcp`) | Các panel hiện **No data**, không có biểu tượng/thông báo lỗi. Tab đã đóng, tab khác của người dùng không bị đụng | ✅ |
| P0-T06-TC18 | Bền vững | Dashboard còn sau `down`/`up` và `down -v`/`up`: `genstop`; `$DC down; $DC up -d --wait`; `curl -s $G/api/dashboards/uid/$UID_ \| jq -c '{p: .meta.provisioned, t: .dashboard.title}'`; lặp lại với `$DC down -v; $DC up -d --wait` | Cả hai lần: `up` exit `0`, dashboard còn với cùng `uid`, cùng title, `provisioned: true` (dashboard nằm trong image, không phụ thuộc volume) | ✅ |
| P0-T06-TC19 | Bền vững | Sửa file JSON rồi `up` thì Grafana nhận bản mới (config COPY vào image): `cp "$DASH" $S/dash.bak`; `jq '.title += " TC19"' $S/dash.bak > "$DASH"`; `$DC up -d --wait`; `curl -s $G/api/dashboards/uid/$UID_ \| jq -r .dashboard.title`; khôi phục `cp $S/dash.bak "$DASH"`; `$DC up -d --wait`; kiểm lại title; `cmp $S/dash.bak "$DASH"` | Sau lần `up` đầu, title có hậu tố ` TC19`, cùng `uid`. Sau khi khôi phục và `up`, title về như cũ. `cmp` không in gì (file trong repo đã khôi phục nguyên vẹn) | ✅ |
| P0-T06-TC20 | Hồi quy | Script kiểm tra cũ vẫn pass và phạm vi thay đổi đúng: `bash scripts/check-compose_test.sh; echo $?`; `bash scripts/check-compose.sh; echo $?`; `bash scripts/check-structure.sh; echo $?`; `git status --porcelain -uall` | Ba lệnh exit `0`. `git status` chỉ có thay đổi trong `deploy/observability/grafana/**` và tài liệu của task (`com/tm/docs/**`). Không đổi `deploy/docker-compose.yml`, `prometheus.yml`, config collector, datasource | ✅ |
| P0-T06-TC21 | Dọn dẹp | Dọn stack và tiến trình sau test: `genstop`; `pgrep -f "$S/gen.py" \| wc -l`; `$DC down -v`; `docker ps -q --filter label=com.docker.compose.project=snaptix \| wc -l`; `docker volume ls -q --filter label=com.docker.compose.project=snaptix \| wc -l`; `rm -rf $S` | Không còn tiến trình generator (`0`), không còn container và volume của project (`0`, `0`). Tab Chrome của TC16/TC17 đã đóng. Scratchpad đã xoá | ✅ |

Test nghiệm thu liên quan: **P0-AT07** (một trace bff → core → PG trên Grafana): task này không làm phần trace, chỉ hoàn thiện Grafana có dashboard được provisioning (cùng mốc demo "Grafana"). Trace thật cần P0-T10/P0-T12, nên để ⬜. Challenge **G10** (Observability, "Prometheus metrics"): task này đóng góp dashboard RED. Tiêu chí "Hoàn thành khi" của G10 là trace BFF → core → PG, nên G10 chưa đổi trạng thái vì task này. **P0-AT01** không liên quan trực tiếp, nhưng TC06/TC18 xác nhận `up -d --wait` vẫn healthy khi image grafana có thêm dashboard.

## Review

### Review lần 1 — CHANGES_REQUESTED

Reviewer độc lập (execute-all), 2026-09-26. Đã kiểm chứng trên máy: dựng `prometheus otel-collector tempo grafana` từ compose, chạy `gen.py`/`dsq.py` trích nguyên văn từ file, dựng thêm một image Grafana 13.2.2 tạm (build context trong scratchpad, không bind mount, cổng 3199) có dashboard RED mẫu + provider `disableDeletion: true`, `allowUiUpdates: false`. Đã `down -v`, xoá container/image tạm, không còn container/volume/tiến trình. Không sửa repo.

**Lỗi chặn (phải sửa):**

- **P0-T06-TC08 fail với mọi cách làm.** Grafana 13.2.2 gốc (image hiện tại, chưa có dashboard) đã in sẵn 2 dòng khớp `level=(error|warn).*(provision|dashboard)`:
  `logger=migrator ... level=warn msg="Skipping migration: Already executed, but not recorded in migration log" id="drop index UQE_dashboard_public_config_uid - v1"` (và `IDX_dashboard_public_config_org_id_dashboard_uid`). Đây là cảnh báo migration DB có sẵn của Grafana, không liên quan tới dashboard của task. Sửa: chỉ lọc logger provisioning, vd `$DC logs grafana | grep -E 'logger=provisioning' | grep -iE 'level=(error|warn)'` phải rỗng (hoặc loại `logger=migrator`). Vế thứ hai (`provisioning.dashboard` có "starting/finished to provision dashboards") giữ nguyên, đã kiểm chứng có.

**Đã kiểm chứng đúng (không cần sửa):**

- A6 / TC11–TC14: số kỳ vọng tính tay và đo thật đều khớp trong ±5 %: rate alpha 10.99, beta 3.99; route `/v1/trips` 9.00, `/v1/seats` 2.00; error alpha 0.0908, beta `0` (có series khi dùng dạng `(5xx or total*0)/total`); p50/p95/p99 alpha 0.01825/0.3623/0.4725, beta 0.175/0.2425/0.2485. `histogram_quantile` nội suy tuyến tính trong bucket nên số ổn định, không phụ thuộc thời điểm; `$__rate_interval` (≥ 60 s với scrape 15 s mặc định) sau 90–100 s generator cho kết quả ổn định.
- TC10/TC09/TC15: `dsq --var` qua resource API trả đúng 2 job, không có `prometheus`/`otel-collector`; service không tồn tại trả `NO DATA`, không `ERROR`.
- TC07: Grafana 13.2.2 trả `400 {"message":"Cannot save provisioned dashboard"}` và `400 {"message":"provisioned dashboard cannot be deleted"}`, title giữ nguyên, `meta.provisioned: true`.
- A2 (tên metric/label sau dịch của Prometheus 3.15), A3 (`job` = `snaptix/<service.name>`), A5 (uid `prometheus`, `allowUiUpdates: false` là mặc định an toàn của Grafana) có căn cứ; panel theo `http_route` có căn cứ ở architecture ("RED metrics cho mỗi endpoint"), không vượt phạm vi.
- Định dạng: tiêu đề, 5 cột, ID TC01–TC21 liên tục, ⬜, có dòng "Test nghiệm thu liên quan". Không có lệnh ghi lịch sử git. TC16–TC17 chỉ localhost, tab mới, chụp màn hình, đóng tab.

**Góp ý nhỏ (không chặn, nên sửa cùng lúc):**

- TC11: regex tiêu đề `'rate'` cũng khớp panel lỗi nếu đặt tên "Error rate" (tên rất phổ biến), khi đó output lẫn giá trị ~0.09. Nên ghi rõ "chỉ xét panel có unit `reqps`" hoặc dùng regex loại trừ `error`.
- TC12: nếu người làm thêm panel Error/Duration theo route, `grep http_route` in cả các giá trị đó; nên ghi "xét panel Rate theo route".
- TC02: máy không có `yq`, nhánh dự phòng `grep path: | awk '{print $2}'` giữ nguyên dấu nháy nếu YAML viết `path: "/etc/..."`; nên thêm `| tr -d '"'\''`.

### Sửa lần 1

Agent sinh test case (execute-all), 2026-09-26, xử lý Review lần 1. Giữ ID P0-T06-TC01..TC21, 5 cột, trạng thái ⬜.

- **P0-T06-TC08 (lỗi chặn)**: chỉ xét log của logger provisioning: `$DC logs grafana | grep -E 'logger=provisioning[.a-z]*' | grep -iE 'level=(error|warn)'` phải rỗng. Kết quả mong đợi ghi rõ cảnh báo có sẵn từ `logger=migrator` (`Skipping migration ... drop index UQE_dashboard_public_config_uid`, `IDX_dashboard_public_config_org_id_dashboard_uid`) không thuộc phạm vi, không tính là fail. Vế thứ hai (`provisioning.dashboard` starting/finished) giữ nguyên.
- **Chuẩn bị**: thêm biến `RATE_RE='^(?!.*(error|lỗi|duration|latency|p50|p95|p99)).*(rate|request|req|throughput|traffic|route)'` vào `prep.sh` (regex Python, dùng lookahead loại trừ) để chọn panel nhóm Rate.
- **P0-T06-TC11**: thay regex tiêu đề `'rate'` bằng `"$RATE_RE"`: vẫn nhận các tên hợp lệ theo A4 ("Rate", "Request rate", "Requests/s", "Throughput") nhưng loại "Error rate" và panel Duration. Thêm cách dự phòng: panel lồng trong row "Rate" mà tiêu đề không khớp thì xác định theo `unit == "reqps"` bằng `jq` và dùng tiêu đề chính xác. Kết quả mong đợi thêm "chỉ in panel nhóm Rate (unit `reqps`), không lẫn giá trị error/duration".
- **P0-T06-TC12**: `dsq 'snaptix/p0t06-alpha' "$RATE_RE" | grep http_route`, chỉ xét panel Rate theo route; panel Error/Duration theo route (nếu có) không thuộc TC này.
- **P0-T06-TC02**: nhánh dự phòng không có `yq` thêm `| tr -d "\"'"` để bỏ dấu nháy khi YAML viết `path: "..."`/`path: '...'`; thêm `echo "$P"` và kết quả mong đợi "đường dẫn không có dấu nháy".

### Review lần 2 — APPROVED

Reviewer độc lập (execute-all), 2026-09-26. Đã kiểm chứng trên máy: `up -d --wait prometheus otel-collector tempo` từ compose; một image Grafana 13.2.2 tạm (build context trong scratchpad, không bind mount, cổng 3100, mạng `snaptix_default`) có dashboard RED mẫu hợp lệ + provider `disableDeletion: true`, `allowUiUpdates: false`, `path: "..."` có dấu nháy. `prep.sh`, `gen.py`, `dsq.py` trích nguyên văn từ file này và chạy trên bash 3.2 của macOS. Sau đó đã `down -v`, xoá container/image tạm và scratchpad: không còn container, volume hay tiến trình generator. Không sửa repo.

**Các điểm của Review lần 1 đã được xử lý đúng:**

- **P0-T06-TC08 (lỗi chặn)**: lệnh `grep -E 'logger=provisioning[.a-z]*' | grep -iE 'level=(error|warn)'` không in dòng nào khi dashboard hợp lệ (exit 1). Các cảnh báo `logger=migrator` (`UQE_dashboard_public_config_uid`...), `logger=settings` và `logger=secret-migrator` đều bị loại đúng. Lệnh vẫn bắt được lỗi thật. Với JSON hỏng, nó in `logger=provisioning.dashboard ... level=error msg="failed to load dashboard from " ... error="unexpected EOF"`. Với `options.path` sai, nó in `level=error msg="Cannot read directory"` và `level=warn msg="Failed to provision config"`. Vế thứ hai có "starting/finished to provision dashboards".
- **P0-T06-TC11/TC12 (`RATE_RE`)**: `RATE_RE` chỉ được truyền vào `dsq`, và `dsq` biên dịch nó bằng `re.compile(..., re.I)` của Python, là công cụ hỗ trợ lookahead. `grep -E` không hỗ trợ lookahead (thử thì báo `invalid syntax`, rc=2), nhưng không TC nào dùng `grep -E` với `RATE_RE`. Kết quả chạy: TC11 chỉ in panel "Request rate" với alpha 10.99 và beta 3.99; panel "Error rate (5xx)" không lọt vào. TC12 in `/v1/trips` 9.00 và `/v1/seats` 2.00, không có beta. Regex nhận "Rate", "Request rate", "Requests/s", "Throughput", "Traffic", "Rate theo route". Regex loại "Error rate", "Errors (%)", "Duration p50/p95/p99", "Latency p95", "Request duration", "Tỉ lệ lỗi".
- **P0-T06-TC02**: máy không có `yq`, nên nhánh dự phòng chạy. Với `path: "/etc/grafana/provisioning/dashboards/json"`, nó in đường dẫn không còn dấu nháy.

**Hồi quy (không phát sinh lỗi mới):** TC09 cho mọi target `NO DATA` và `0` dòng ERROR. TC10 trả đúng 2 job. TC13 cho alpha 0.0908 và beta `0`. TC14 cho alpha 0.01825/0.3623/0.4725 và beta 0.175/0.2425/0.2485. TC15 cho `0`/`0`, và service không tồn tại cho toàn `NO DATA`. Mọi giá trị nằm trong ±5 % so với A6. Không có lệnh ghi lịch sử git. Định dạng: tiêu đề đúng, 5 cột, ID P0-T06-TC01..TC21 liên tục, trạng thái ⬜, có dòng "Test nghiệm thu liên quan".

**Góp ý nhỏ (không chặn):**

- P0-T06-TC09: khi biến rỗng, `dsq --var` in đúng **một dòng trống** (`print("\n".join([]))`). Khi chạy nên hiểu "không in dòng nào" là không có giá trị nào, vd `dsq --var | grep -c .` = `0`.
- P0-T06-TC11/TC12: `RATE_RE` không khớp tiêu đề "RPS", và khớp nhầm tiêu đề kiểu "Failed requests %". Cách dự phòng theo `unit == "reqps"` đã ghi trong TC11 xử lý được cả hai trường hợp này. Người chạy cần đối chiếu unit khi thấy giá trị lạ.
