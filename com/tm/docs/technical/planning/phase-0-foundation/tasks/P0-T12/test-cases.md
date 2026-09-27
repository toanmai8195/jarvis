# Test cases — P0-T12: OTel cho Node, gọi thử `core /healthz` để kiểm tra trace xuyên service

> **Phạm vi**: gắn OpenTelemetry (trace + metric, OTLP/HTTP) vào `apps/bff`. SDK khởi tạo **trước** khi nạp fastify/http, cả bản build (`node dist/…`) lẫn khi dev (`tsx watch`). Fastify có span server theo route. Lời gọi HTTP từ bff sang core có span client và header `traceparent`, nên trên Tempo **một** trace có span bff (server + client), span core (server) và span PG của core. Log pino của bff có `trace_id`/`span_id` khớp Tempo. Metric RED của bff (`job="snaptix/bff"`) vào Prometheus và dashboard `snaptix-red`. Core chết/treo thì bff trả lỗi có kiểm soát trong hạn. Collector chết thì bff vẫn phục vụ. Căn cứ:
> - Phase 0 README:
>   - Dòng **P0-T12** "OTel cho Node, gọi thử `core /healthz` để kiểm tra trace xuyên service `[G10]`".
>   - **G10**, hoàn thành khi: "Một trace hiển thị đủ BFF → core → PG".
>   - **DoD**: "Gọi `bff /healthz?deep=1` → thấy **một trace** gồm span của bff và core trên Grafana".
>   - **P0-NFR1** "Log dạng JSON, có `trace_id`, `request_id`". **P0-NFR2** "Trace truyền từ `bff` sang `core` qua `traceparent`".
>   - Task sau giữ ranh giới: **P0-T13** (graceful shutdown Fastify, `close` hooks), **P0-T15** (`packages/config` dùng chung), **P0-T17** (khung Vitest chung). Phase 2 **P2-T05** là core client thật (undici keep-alive, retry, circuit breaker, challenge N2).
> - [acceptance-tests](../../acceptance-tests.md) **P0-AT07**, nguyên văn: "Gọi `bff /healthz?deep=1`, mở Grafana | Một trace chứa span bff → core → PG | P0-NFR2, G10".
> - [architecture](../../../../architecture.md) mục Observability:
>   - "trace ID truyền từ BFF sang core qua header `traceparent`".
>   - SDK Go (`pkg/otelx`) export OTLP/HTTP 4318, propagator W3C TraceContext + Baggage, cấu hình bằng env chuẩn `OTEL_*`, resource `service.name` + `service.namespace=snaptix` → Prometheus `job="<namespace>/<name>"`.
>   - Log luôn kèm `trace_id`, `request_id`. Dashboard RED (`snaptix-red`) dùng `http_server_request_duration_seconds_*`, biến `service` = label `job`, Rate theo `http_route`, Errors theo `http_response_status_code` 5xx.
>   - Stack bff: undici.
> - [project-structure](../../../../project-structure.md):
>   - Cây `apps/bff/src/`: `plugins/` (có `otel`), `core-client/` ("undici: timeout, retry, circuit breaker").
>   - Quy tắc 3 tầng: `app/packages/<tên>` chỉ khi "service thứ hai cần **và** là hạ tầng". Hiện chỉ bff là service Node, web chạy trên trình duyệt nên SDK khác. Vì vậy code OTel nằm trong `apps/bff`, không tạo `packages/otel`.
> - [local-setup](../../../../local-setup.md):
>   - `CORE_BASE_URL` (bff) ví dụ `http://localhost:8080`.
>   - Bảng `OTEL_*` của core (endpoint mặc định `http://localhost:4318`, `OTEL_SERVICE_NAME`, `OTEL_RESOURCE_ATTRIBUTES`, `OTEL_SDK_DISABLED`, `OTEL_BSP_SCHEDULE_DELAY`, `OTEL_METRIC_EXPORT_INTERVAL`).
>   - Cổng: OTLP HTTP 4318, Tempo 3200, Prometheus 9090, Grafana 3100.
> - [api.md](../../../../api.md): header `traceparent`, `X-Request-ID`. Core có `GET /healthz`, `/readyz`.
> - [execute-all-notes](../../../execute-all-notes.md):
>   - Mục P0-T10: core `/healthz` **không** chạm PG, chỉ `/readyz` có span PG (`pool.acquire` → `connect` qua `otelpgx`). P0-T12 phải chọn route core, "vd gọi `/readyz`".
>   - Mục P0-T11: `trace_id` trong log nên làm bằng `mixin` của pino hoặc `LogController`. `pnpm` phải chạy với cwd trong `com/tm/app` (shim corepack). tsx in "Force killing" khi restart, việc này để P0-T13.
>
> **Giả định** (reviewer xem kỹ):
> - A1. **Route trace BFF → core → PG** (chỗ task và AT07 lệch nhau):
>   - `GET /healthz?deep=1` của bff gọi **`GET {CORE_BASE_URL}/readyz`** của core. Core `/readyz` là health check của core có ping PG, nên ra span PG.
>   - Lý do:
>     - Dòng task ghi "gọi thử `core /healthz`" theo nghĩa health check của core.
>     - Nhưng AT07 và G10 đòi span PG, mà core `/healthz` không chạm PG (ghi chú P0-T10).
>     - Gọi `/readyz` thì **không sửa `com/tm/server`**. Thêm `?deep=1` cho core là đổi code core ngoài workstream bff của task.
>   - Không gọi thêm route core nào khác. Không gọi `/internal/v1/...`, vì API nghiệp vụ và service token là phase 2.
> - A2. **Hành vi `/healthz` của bff**:
>   - Không có `deep`, hoặc `deep` ≠ `1`: giữ như P0-T11. Trả `200 {"status":"ok"}`, **không** gọi core, không chạm Mongo.
>   - `deep=1`: gọi core `/readyz` **một lần**, hạn **2 s** (giống `/readyz`).
>     - Core trả `200` thì bff trả `200 {"status":"ok","core":"ok"}`.
>     - Core trả mã khác 2xx, lỗi kết nối, hoặc quá hạn thì bff trả `503 {"status":"unavailable","core":"unavailable"}`, kèm một dòng log `warn` có `request_id`, `trace_id`, loại lỗi. Không log stack thô, không lộ URL có userinfo.
>   - Không retry, không circuit breaker, không tuỳ chỉnh keep-alive agent (P2-T05).
>   - `/readyz` của bff vẫn chỉ ping Mongo (P0-T11 A3), không gọi core.
> - A3. **Config `CORE_BASE_URL`** (plugin config của P0-T11):
>   - Mặc định `http://localhost:8080`. Phải là URL `http://`/`https://` có host.
>   - Sai thì xử lý như lỗi config P0-T11: log JSON `fatal` nêu **tên** biến, không in giá trị, exit `1`, không listen.
>   - `.env.example` thêm `CORE_BASE_URL` và các `OTEL_*` dùng cho bff.
> - A4. **Khởi tạo SDK trước fastify (ESM)**:
>   - File entry riêng `src/instrumentation.ts`. tsup build thêm entry ra `dist/instrumentation.js`.
>   - Nạp bằng cờ `--import` của Node, **trước** `server.ts`:
>     - Script `start` = `node --import ./dist/instrumentation.js dist/server.js`.
>     - Script `dev` = `tsx watch --import ./src/instrumentation.ts src/server.ts`, hoặc cách tương đương đã kiểm chứng. tsx phải nạp instrumentation trước `server.ts`.
>   - Lý do: tsup gom `server.ts` và `app.ts` thành một bundle, `import` tĩnh được hoist. Nếu `sdk.start()` nằm trong cùng bundle thì fastify và `node:http` đã được nạp trước khi SDK patch. Nếu người làm chọn tên file khác thì sửa `bff()` trong prep cho khớp và ghi vào Ghi chú thực thi.
>   - Instrumentation cần có:
>     - Span server: `@opentelemetry/instrumentation-http`, và/hoặc `@fastify/otel` để có `http.route` và tên span `GET /healthz`. **Không** dùng `@opentelemetry/instrumentation-fastify`, gói này đã deprecated, npm ghi "in favor of @fastify/otel".
>     - Span client cho lời gọi core: `@opentelemetry/instrumentation-undici` (phủ cả `fetch` global lẫn gói `undici`), hoặc instrumentation-http nếu dùng `node:http`.
>     - Không dùng `@opentelemetry/auto-instrumentations-node`: chỉ bật instrumentation cần dùng, dễ hiểu từng mảnh.
>   - Code OTel khác (nếu có) nằm trong `src/plugins/otel.ts` hoặc `src/instrumentation.ts`. Lời gọi core đặt trong `src/core-client/` (theo project-structure), hoặc gọi thẳng trong route nếu chỉ vài dòng. Không tạo `packages/*`.
> - A5. **Env OTel** (giống ngữ nghĩa bảng core trong local-setup):
>   - `OTEL_EXPORTER_OTLP_ENDPOINT` mặc định `http://localhost:4318`, OTLP/HTTP (protobuf hoặc JSON). Giá trị không phải URL `http(s)://` có host thì exit `1`, log JSON `fatal` nêu tên biến, giống core.
>   - `service.name` mặc định **`bff`**, `OTEL_SERVICE_NAME` đè được. `service.namespace` mặc định **`snaptix`**, `OTEL_RESOURCE_ATTRIBUTES` thêm hoặc đè được. Nhờ vậy Prometheus có `job="snaptix/bff"`.
>   - Sampler mặc định `parentbased_always_on`. Propagator W3C TraceContext + Baggage.
>   - `OTEL_SDK_DISABLED=true`: không export gì, bff vẫn phục vụ bình thường.
>   - `OTEL_BSP_SCHEDULE_DELAY`, `OTEL_METRIC_EXPORT_INTERVAL` được tôn trọng. TC đặt `500`/`2000` để dữ liệu lên nhanh.
>   - Thiếu mọi biến `OTEL_*` (env sạch) vẫn chạy được.
> - A6. **Span**:
>   - Mỗi request bff có đúng **một** span `SERVER` của service `bff`. Có `traceparent` vào thì parent là span id trong header. Tên span `GET <route>` (vd `GET /healthz`), có thuộc tính `http.route` = pattern route, `http.response.status_code`.
>   - Lời gọi core là span `CLIENT` của `bff`, là con hoặc cháu của span server. Có `http.request.method`, `server.address`/`url.full` trỏ tới core, `http.response.status_code` khi có response. Span này gửi `traceparent` mang **span id của chính nó**.
>   - Server 5xx thì span server có status `ERROR`. Lỗi kết nối, quá hạn, hoặc core trả 5xx thì span client có status `ERROR`.
>   - Không có header nhạy cảm (`authorization`, `cookie`) trong thuộc tính span.
> - A7. **Log**:
>   - Mọi dòng log pino phát ra **trong** một request (`incoming request`, `request completed`, `warn` của deep/readyz) có `trace_id` (32 hex) và `span_id` (16 hex) của span bff đang active, cùng `request_id` như P0-T11. Tên khoá giống core (`trace_id`, `span_id`).
>   - Cách làm tuỳ người làm: `mixin` của pino đọc `trace.getActiveSpan()`, `@opentelemetry/instrumentation-pino`, hoặc child logger gắn trong hook `onRequest`. Miễn là có cả dòng `request completed`.
>   - Quy tắc P0-T11 A4 vẫn giữ: process **không** in dòng nào không phải JSON ra stdout/stderr, kể cả khi collector chết. Nếu có log lỗi export thì phải đi qua pino JSON (vd bridge `diag` sang pino ở mức `warn`) hoặc không in gì.
> - A8. **Metric RED**:
>   - bff có histogram `http.server.request.duration` (đơn vị **giây**), thuộc tính `http.request.method`, `http.route`, `http.response.status_code`. Trên Prometheus là `http_server_request_duration_seconds_*{job="snaptix/bff"}`, để dashboard `snaptix-red` có thêm service bff mà không sửa `red.json`.
>   - Có thể lấy từ instrumentation-http (semconv ổn định, vd `OTEL_SEMCONV_STABILITY_OPT_IN=http` đặt trong code) hoặc tự ghi trong hook `onResponse`. Env sạch vẫn phải ra đúng tên metric này.
>   - Route không khớp (404) **không** được dùng path thô làm `http_route`, để tránh bùng cardinality.
> - A9. **Phiên bản** (npm registry ngày 2026-09-27; pin `^`/`~`, không `latest`/`*`):
>   - `@opentelemetry/api` `1.9.x`.
>   - `@opentelemetry/sdk-node`, `exporter-trace-otlp-*`, `exporter-metrics-otlp-*`, `instrumentation-http` `0.222.x`. `sdk-metrics`, `resources`, `sdk-trace-base` `2.11.x`.
>   - `@opentelemetry/instrumentation-undici` `0.32.x`. `@fastify/otel` `0.21.x` (nếu dùng).
>   - Nếu dùng gói `undici` 8.x thì gói này đòi Node `>=22.19.0` (máy có 22.22.3). Khi đó nâng `engines.node` cho khớp. Dùng `fetch` global của Node 22 thì không phải nâng.
>   - pnpm 11 chặn build script: nếu dependency mới cần build thì duyệt tường minh trong `allowBuilds` của `pnpm-workspace.yaml` (P0-T11 A9), không để `ERR_PNPM_IGNORED_BUILDS`.
> - A10. **Unit test** (Vitest, không cần Docker, collector hay core thật):
>   - Dùng `InMemorySpanExporter` (+ `SimpleSpanProcessor`) và, nếu cần, `InMemoryMetricExporter`.
>   - Core giả là `http.createServer` trên cổng `0` trong test, hoặc một pinger/client giả qua option của `buildApp`, giống cách P0-T11 truyền `pinger`.
>   - Hạn gọi core truyền vào được, để test "core treo" trả trong hạn ngắn.
> - A11. **Không graceful shutdown** (P0-T13): không bắt SIGTERM/SIGINT, không gọi `sdk.shutdown()` theo tín hiệu. Vì vậy các TC chờ span được export (`OTEL_BSP_SCHEDULE_DELAY=500` + `waittrace`) rồi mới dừng process.
>
> **Không thuộc task này**:
> - Bắt SIGTERM, flush telemetry khi dừng, `close-with-grace` (P0-T13). `packages/config`, `packages/otel` (P0-T15, quy tắc 3 tầng). Khung Vitest chung (P0-T17).
> - Retry, circuit breaker, keep-alive agent, service token, `X-User-Id`, `/internal/v1/*` (P2-T05 trở đi).
> - Sửa `com/tm/server` (core, `pkg/otelx`). Sửa `red.json`, collector, compose. Dockerfile/image bff, service bff trong compose. Format lỗi `{"error":…}`. OTel trên web.
>
> **Môi trường khi viết**:
> - macOS arm64, Node 22.22.3, pnpm 11.18.0 (trong `com/tm/app`), Go 1.27.1 (build core bằng `go build`).
> - Docker Desktop + stack P0-T02 qua `dc`: `postgres-core`, `mongodb`, `otel-collector`, `tempo`, `prometheus`, `grafana`.
> - curl, jq, `/usr/bin/python3` 3.9, `lsof`, Chrome có Claude in Chrome.
>
> **Chuẩn bị**: lưu khối dưới thành `<scratchpad>/p0t12/prep.sh` một lần (tạo bằng Write tool, thay `<scratchpad>` bằng đường dẫn thật). Mỗi lần gọi Bash bắt đầu bằng `source <scratchpad>/p0t12/prep.sh`.
> - Chạy được trên cả bash 3.2 và zsh:
>   - Biến viết trong `${…}` khi đứng sát `:`. Không dùng biến tên `status`/`path`. Mảng env dùng `"${X[@]}"`.
>   - `pnpm` luôn chạy với cwd `$APP` (hàm `pa`), không dùng `pnpm --dir` từ gốc repo.
> - Process nền (core, bff, dev, core-treo) chạy trong process group riêng (`bgrun`/`bgstop`). `nap` dùng python. macOS không có `timeout`, nên dùng `runlimit`.
> - Phần chuẩn bị và các TC **không** có lệnh ghi lịch sử git (không `git commit`/`add`/`stash`).
> - TC01–TC05 và TC20 không cần stack.
> - Trước TC06: `dc up -d --wait` (toàn stack), `corebuild`, `pa --filter bff build`, `listen $P \| wc -l` và `listen $CP \| wc -l` đều `0`.
> - Core chạy bằng `corerun` ở `127.0.0.1:18080` (không đụng 8080 của dev), với `CORE_DATABASE_URL` của `postgres-core`. Core **không** cần migration (P0-T10).
> - Mỗi TC E2E chạy trong **một** lần gọi Bash, từ lúc khởi động tới `bgstop`.
> ```bash
> REPO=/Users/maiductoan/Documents/code/jarvis; cd "$REPO"
> APP=$REPO/com/tm/app; BFF=$APP/apps/bff; SRV=$REPO/com/tm/server
> S=<scratchpad>/p0t12; mkdir -p "$S"
> dc() { docker compose -f "$REPO/deploy/docker-compose.yml" "$@"; }
> pa() { (cd "$APP" && pnpm "$@"); }            # pnpm trong com/tm/app (shim corepack chọn 11.18.0 theo cwd)
> P=13000; B=http://127.0.0.1:${P}               # bff test
> CP=18080; C=http://127.0.0.1:${CP}             # core test
> HP=18081                                       # core giả "treo" (accept, không trả lời)
> TEMPO=http://localhost:3200; PROM=http://localhost:9090; G=http://localhost:3100
> MURI='mongodb://localhost:27017/snaptix'
> DSN='postgres://snaptix:snaptix@localhost:5432/core?sslmode=disable'
> EP=(OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318)
> FAST=(OTEL_BSP_SCHEDULE_DELAY=500 OTEL_METRIC_EXPORT_INTERVAL=2000)
> BENV=(MONGODB_URI=$MURI BFF_PORT=$P CORE_BASE_URL=$C)
> RED=$REPO/deploy/observability/grafana/dashboards/red.json
> nap()   { python3 -c 'import time,sys;time.sleep(float(sys.argv[1]))' "$1"; }
> now()   { python3 -c 'import time;print("%.3f"%time.time())'; }
> since() { python3 -c "import time;print('%.2f'%(time.time()-$1))"; }
> hex()   { python3 -c 'import secrets,sys;print(secrets.token_hex(int(sys.argv[1])))' "$1"; }
> # tp TID SID [FLAGS] — header traceparent (W3C), mặc định sampled (01)
> tp()    { echo "traceparent: 00-$1-$2-${3:-01}"; }
> # bgrun NAME CMD... — chạy nền trong process group riêng; log $S/NAME.log (stdout+stderr), pid $S/NAME.pid
> bgrun() { local n=$1; shift
>           python3 -c 'import os,sys; os.setsid(); os.execvp(sys.argv[1], sys.argv[1:])' "$@" >"$S/$n.log" 2>&1 &
>           echo $! >"$S/$n.pid"; }
> # bgstop NAME — SIGTERM cả group, chờ ≤ 5 s rồi SIGKILL
> bgstop() { [ -f "$S/$1.pid" ] || return 0
>            python3 -c 'import os,signal,sys,time
> g=int(sys.argv[1])
> for s in (signal.SIGTERM, signal.SIGKILL):
>     try: os.killpg(g, s)
>     except ProcessLookupError: sys.exit(0)
>     for _ in range(50):
>         try: os.killpg(g, 0); time.sleep(0.1)
>         except ProcessLookupError: sys.exit(0)' "$(cat "$S/$1.pid")"; rm -f "$S/$1.pid"; }
> alive() { [ -f "$S/$1.pid" ] || { echo "no-pid"; return 0; }; python3 -c 'import os,sys
> try: os.killpg(int(sys.argv[1]),0); print("alive")
> except ProcessLookupError: print("dead")' "$(cat "$S/$1.pid")"; }
> # corebuild — build binary core (không sửa repo); corerun [VAR=val...] — core nền ở $C với env sạch
> corebuild() { (cd "$SRV" && go build -o "$S/core" ./services/core/cmd/server); }
> corerun() { bgstop core; bgrun core env -i PATH="$PATH" HOME="$HOME" CORE_DATABASE_URL="$DSN" CORE_HTTP_ADDR=127.0.0.1:${CP} "${EP[@]}" "${FAST[@]}" "$@" "$S/core"; }
> # bff NAME [VAR=val...] — bản build, đúng lệnh của script start (A4), cwd $BFF, env SẠCH + biến truyền vào
> bff()   { local n=$1; shift; (cd "$BFF" && bgrun "$n" env -i PATH="$PATH" HOME="$HOME" "$@" node --import ./dist/instrumentation.js dist/server.js); }
> # hang — core giả: nhận kết nối TCP ở $HP nhưng không bao giờ trả lời
> hang()  { bgrun hang python3 -c 'import socket,time
> s=socket.socket(); s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1); s.bind(("127.0.0.1",'"$HP"')); s.listen(64); c=[]
> while True: c.append(s.accept()[0])'; }
> # runlimit SECS OUT CMD... — chạy tối đa SECS giây, output vào OUT, in exit=<code> (124 = quá hạn, đã kill)
> runlimit() { python3 - "$@" <<'PY'
> import os,signal,subprocess,sys,time
> secs=float(sys.argv[1]); out=sys.argv[2]; cmd=sys.argv[3:]; t=time.time()
> with open(out,"w") as f:
>     p=subprocess.Popen(cmd,stdout=f,stderr=subprocess.STDOUT,start_new_session=True)
>     try: rc=p.wait(timeout=secs)
>     except subprocess.TimeoutExpired:
>         os.killpg(p.pid,signal.SIGKILL); p.wait(); rc=124
> print("exit=%d elapsed=%.1fs"%(rc,time.time()-t))
> PY
> }
> code()  { curl -s -m 10 -o /dev/null -w '%{http_code}\n' "$@"; }
> ctime() { curl -s -m 10 -o /dev/null -w '%{http_code} %{time_total}s\n' "$@"; }
> body()  { curl -s -m 10 "$@"; echo; }
> waitcode() { local s=$(date +%s) c; while :; do c=$(curl -s -m 5 -o /dev/null -w '%{http_code}' "$1")
>              [ "$c" = "$2" ] && { echo "$c after $(( $(date +%s)-s ))s"; return 0; }
>              [ $(( $(date +%s)-s )) -ge "$3" ] && { echo "$c TIMEOUT"; return 1; }; nap 0.3; done; }
> listen() { lsof -nP -iTCP:"$1" -sTCP:LISTEN | awk 'NR>1{print $9}' | sort -u; }
> # jsonlines FILE [only] — mọi dòng không rỗng là JSON có time/level/msg (như P0-T11); đối số 2 → chỉ xét dòng bắt đầu bằng `{`
> jsonlines() { python3 -c 'import json,sys
> n=bad=0
> for l in open(sys.argv[1]):
>     l=l.strip()
>     if not l: continue
>     if len(sys.argv)>2 and not l.startswith("{"): continue
>     n+=1
>     try: o=json.loads(l); assert isinstance(o,dict) and {"time","level","msg"}<=o.keys()
>     except Exception: bad+=1; print("BAD:",l[:200])
> print("lines=%d bad=%d"%(n,bad))' "$@"; }
> # jl FILE 'JQ' — áp jq lên các dòng JSON của log
> jl()    { jq -cR "fromjson? | select(type==\"object\") | $2" "$1"; }
> # coreacc PATH — số dòng access log của core ("http request") có path = PATH
> coreacc() { jl "$S/core.log" "select(.msg==\"http request\" and .path==\"$1\")" | wc -l | tr -d ' '; }
> # Tempo: tcode TID — HTTP code; waittrace TID N — chờ tối đa N giây tới khi 200
> tcode()    { curl -s -m 5 -o /dev/null -w '%{http_code}' -H 'Accept: application/json' "$TEMPO/api/traces/$1"; }
> waittrace() { local s=$(date +%s) c; while :; do c=$(tcode "$1"); [ "$c" = 200 ] && { echo "trace 200 after $(( $(date +%s)-s ))s"; return 0; }
>              [ $(( $(date +%s)-s )) -ge "$2" ] && { echo "trace $c TIMEOUT"; return 1; }; nap 1; done; }
> # tspans TID — mỗi span một dòng JSON: trace,id,parent,name,kind,status,scope,attrs,resource (như P0-T10)
> cat >"$S/tspans.py" <<'EOF'
> import base64, json, sys, urllib.request
> req = urllib.request.Request("http://localhost:3200/api/traces/" + sys.argv[1], headers={"Accept": "application/json"})
> d = json.load(urllib.request.urlopen(req, timeout=10))
> d = d.get("trace", d)
> def hid(v):
>     if not v: return ""
>     if len(v) in (16, 32) and all(c in "0123456789abcdefABCDEF" for c in v): return v.lower()
>     return base64.b64decode(v).hex()
> def val(v):
>     for k in ("stringValue", "intValue", "boolValue", "doubleValue"):
>         if k in v: return v[k]
>     return v
> def attrs(a): return dict((x["key"], val(x.get("value", {}))) for x in (a or []))
> KIND = {1: "INTERNAL", 2: "SERVER", 3: "CLIENT", 4: "PRODUCER", 5: "CONSUMER"}
> for r in d.get("batches") or d.get("resourceSpans") or []:
>     ra = attrs((r.get("resource") or {}).get("attributes"))
>     for ss in r.get("scopeSpans") or r.get("instrumentationLibrarySpans") or []:
>         for s in ss.get("spans", []):
>             k = s.get("kind", "")
>             k = KIND.get(k, k) if isinstance(k, int) else str(k).replace("SPAN_KIND_", "")
>             print(json.dumps({"trace": hid(s.get("traceId")), "id": hid(s.get("spanId")), "parent": hid(s.get("parentSpanId")),
>                 "name": s.get("name"), "kind": k, "status": str((s.get("status") or {}).get("code", "")),
>                 "scope": (ss.get("scope") or {}).get("name", ""), "attrs": attrs(s.get("attributes")), "resource": ra}, sort_keys=True))
> EOF
> tspans()   { python3 "$S/tspans.py" "$1"; }
> # xchain TID — tóm tắt chuỗi bff → core → PG của trace (JSON một dòng)
> cat >"$S/xchain.py" <<'EOF'
> import json, sys
> sp = [json.loads(l) for l in sys.stdin if l.strip()]
> by = dict((s["id"], s) for s in sp)
> def svc(s): return s["resource"].get("service.name")
> def under(s, pid):
>     n = 0
>     while s and s["parent"] and n < 100:
>         if s["parent"] == pid: return True
>         s = by.get(s["parent"]); n += 1
>     return False
> bs = [s for s in sp if svc(s) == "bff" and s["kind"] == "SERVER"]
> bc = [s for s in sp if svc(s) == "bff" and s["kind"] == "CLIENT"]
> cs = [s for s in sp if svc(s) == "core" and s["kind"] == "SERVER"]
> pg = [s for s in sp if svc(s) == "core" and (s["attrs"].get("db.system.name") or s["attrs"].get("db.system")) == "postgresql"]
> bcid = set(c["id"] for c in bc)
> print(json.dumps({
>   "traces": sorted(set(s["trace"] for s in sp)), "services": sorted(set(svc(s) for s in sp)),
>   "ns": sorted(set(s["resource"].get("service.namespace", "") for s in sp)),
>   "bff_server": [{"name": s["name"], "parent": s["parent"], "status": s["status"], "route": s["attrs"].get("http.route"),
>                   "code": s["attrs"].get("http.response.status_code")} for s in bs],
>   "bff_client": [{"name": s["name"], "under_server": any(under(s, b["id"]) for b in bs), "status": s["status"],
>                   "method": s["attrs"].get("http.request.method"), "url": s["attrs"].get("url.full"),
>                   "host": s["attrs"].get("server.address"), "code": s["attrs"].get("http.response.status_code")} for s in bc],
>   "core_server": [{"name": s["name"], "parent_is_bff_client": s["parent"] in bcid, "status": s["status"]} for s in cs],
>   "pg_under_core": sum(1 for p in pg if any(under(p, c["id"]) for c in cs))}, sort_keys=True))
> EOF
> xchain()   { tspans "$1" | python3 "$S/xchain.py"; }
> # bffids TID — span id của mọi span service bff trong trace
> bffids()   { tspans "$1" | jq -r 'select(.resource["service.name"]=="bff") | .id'; }
> # prom QUERY — mỗi series một dòng "<labels JSON> <giá trị>"; waitprom QUERY N — chờ tới khi có kết quả
> prom()     { python3 -c 'import json,sys,urllib.request,urllib.parse
> d=json.load(urllib.request.urlopen("http://localhost:9090/api/v1/query?"+urllib.parse.urlencode({"query":sys.argv[1]}),timeout=10))
> if d.get("status")!="success": sys.exit("ERROR "+json.dumps(d))
> for r in d["data"]["result"]: print(json.dumps(r["metric"],sort_keys=True), r["value"][1])' "$1"; }
> waitprom() { local s=$(date +%s); while :; do [ -n "$(prom "$1")" ] && { echo "prom ok after $(( $(date +%s)-s ))s"; return 0; }
>              [ $(( $(date +%s)-s )) -ge "$2" ] && { echo "prom EMPTY TIMEOUT"; return 1; }; nap 1; done; }
> # dsq.py DASH --var | DASH VALUE — (nguyên văn P0-T10) query mọi target của dashboard qua Grafana, thay $service bằng VALUE
> cat >"$S/dsq.py" <<'EOF'
> import json, re, sys, urllib.request, urllib.parse, urllib.error
> G = "http://localhost:3100"; dash = json.load(open(sys.argv[1]))
> v = [x for x in dash["templating"]["list"] if x.get("type") == "query"][0]; var = v["name"]
> if sys.argv[2] == "--var":
>     q = v.get("definition") or (v["query"]["query"] if isinstance(v["query"], dict) else v["query"])
>     m = re.fullmatch(r"\s*label_values\((.+),\s*(\w+)\)\s*", q)
>     u = G + "/api/datasources/uid/prometheus/resources/api/v1/label/%s/values?" % m.group(2) + urllib.parse.urlencode({"match[]": m.group(1)})
>     print("\n".join(json.load(urllib.request.urlopen(u))["data"])); sys.exit()
> val = sys.argv[2]
> def sub(e): return e.replace("${%s}" % var, val).replace("$" + var, val)
> def walk(ps):
>     for p in ps:
>         yield p
>         for c in walk(p.get("panels", [])): yield c
> for p in walk(dash.get("panels", [])):
>     for t in p.get("targets", []):
>         if not t.get("expr"): continue
>         rid = t.get("refId", "A")
>         body = {"from": "now-5m", "to": "now", "queries": [{"refId": rid, "datasource": {"type": "prometheus", "uid": "prometheus"},
>                 "expr": sub(t["expr"]), "instant": True, "range": False}]}
>         req = urllib.request.Request(G + "/api/ds/query", data=json.dumps(body).encode(), headers={"Content-Type": "application/json"})
>         try: r = json.load(urllib.request.urlopen(req))["results"][rid]
>         except urllib.error.HTTPError as ex: print("%s | %s | ERROR HTTP %s" % (p["title"], rid, ex.code)); continue
>         if r.get("error"): print("%s | %s | ERROR %s" % (p["title"], rid, r["error"])); continue
>         n = 0
>         for f in r.get("frames", []):
>             vs = f["data"]["values"]
>             if len(vs) < 2 or not vs[1]: continue
>             print("%s | %s | %s | %s" % (p["title"], rid, json.dumps(f["schema"]["fields"][1].get("labels") or {}, sort_keys=True), vs[1][-1])); n += 1
>         if n == 0: print("%s | %s | NO DATA" % (p["title"], rid))
> EOF
> # exploreurl TID — URL Grafana Explore (datasource tempo) mở trace TID
> exploreurl() { python3 -c 'import json,sys,urllib.parse
> p={"a":{"datasource":"tempo","queries":[{"refId":"A","datasource":{"type":"tempo","uid":"tempo"},"queryType":"traceql","query":sys.argv[1]}],"range":{"from":"now-1h","to":"now"}}}
> print("http://localhost:3100/explore?schemaVersion=1&orgId=1&panes="+urllib.parse.quote(json.dumps(p,separators=(",",":"))))' "$1"; }
> ```

| ID | Loại | Kịch bản | Kết quả mong đợi | Trạng thái |
|---|---|---|---|---|
| P0-T12-TC01 | Cấu trúc | Vị trí code và script (A4, quy tắc 3 tầng): `jq '{engines,scripts}' $BFF/package.json`; `ls $BFF/src/instrumentation.ts`; `grep -nE "entry" $BFF/tsup.config.ts`; `ls -A $APP/packages`; `find $BFF/src -type d \| sort`; `grep -rlE "@opentelemetry/sdk-node\|NodeSDK" $BFF/src \| grep -v '\.test\.ts$'`; `grep -rnE "from ['\"]fastify['\"]\|from ['\"](node:)?http['\"]" $BFF/src/instrumentation.ts \| wc -l`; `grep -rnE "@opentelemetry/instrumentation-fastify\|auto-instrumentations-node" $BFF/src $BFF/package.json \| wc -l`; `bash scripts/check-structure.sh; echo $?` | `start` = `node --import ./dist/instrumentation.js dist/server.js`. `dev` chứa `tsx watch` và nạp `src/instrumentation.ts` trước `src/server.ts` (vd `--import ./src/instrumentation.ts`). tsup có entry `src/instrumentation.ts` bên cạnh `src/server.ts`. `packages/` chỉ có `.gitkeep`, không có `otel` hay package mới nào. Code khởi tạo SDK chỉ nằm trong `src/instrumentation.ts` và/hoặc `src/plugins/otel.ts`. Lời gọi core nằm trong `src/core-client/` hoặc route health. Không có thư mục kiểu Java. `instrumentation.ts` không import fastify/http (`0`). Không có instrumentation-fastify deprecated hay auto-instrumentations (`0`). `check-structure.sh` exit `0` | ✅ |
| P0-T12-TC02 | Dependency | Dependency OTel đúng bản, install sạch (A9): `jq '{dependencies,devDependencies}' $BFF/package.json`; `jq -r '(.dependencies//{}) + (.devDependencies//{}) \| to_entries[] \| select(.value=="latest" or .value=="*" or .value=="") \| .key' $BFF/package.json \| wc -l`; `pa --filter bff list --depth 0 \| grep -E '@opentelemetry\|@fastify/otel\|undici'`; `shasum $APP/pnpm-lock.yaml > $S/lock.before`; `pa install --frozen-lockfile > $S/install.txt 2>&1; echo $?`; `grep -ciE 'unmet peer\|ERR_PNPM\|ignored build scripts\|WARN\|deprecated' $S/install.txt`; `shasum -c $S/lock.before`; `grep -c 'set this to' $APP/pnpm-workspace.yaml`; `jq -r '(.dependencies//{}) + (.devDependencies//{}) \| keys[]' $BFF/package.json \| grep -cE '^(@nestjs/\|ioredis$\|redis$\|@fastify/(session\|secure-session\|cookie\|csrf-protection\|rate-limit\|oauth2)$\|opossum$\|cockatiel$\|pino-pretty$)'` | `dependencies` có `@opentelemetry/api` (1.9.x), `@opentelemetry/sdk-node` (0.222.x), exporter trace và metric OTLP HTTP/proto (0.222.x), `@opentelemetry/instrumentation-http` (0.222.x) và/hoặc `@fastify/otel` (0.21.x), `@opentelemetry/instrumentation-undici` (0.32.x) nếu gọi core bằng `fetch`/undici. Mọi gói OTel runtime nằm trong `dependencies`, không nằm trong `devDependencies` (vì `pnpm deploy --prod`). Không có spec `latest`/`*`/rỗng (`0`). Install frozen exit `0`, không cảnh báo peer, lỗi, build bị bỏ qua, `WARN` hay deprecated (`0`). Lockfile không đổi (`OK`). Không còn mục giữ chỗ pnpm (`0`). Không có dependency của task sau (session, Redis, rate limit, circuit breaker, NestJS, pino-pretty) (`0`) | ✅ |
| P0-T12-TC03 | Build | Typecheck, lint, build ra hai entry ESM: `pa --filter bff typecheck; echo $?`; `pa --filter bff lint > $S/lint.txt 2>&1; echo $?`; `grep -ciE '[0-9]+ (problems?\|warnings?\|errors?)' $S/lint.txt`; `rm -rf $BFF/dist; pa --filter bff build; echo $?`; `find $BFF/dist -type f \| sort`; `find $BFF/dist -name '*test*' \| wc -l`; `node --check $BFF/dist/instrumentation.js; echo $?`; `node --check $BFF/dist/server.js; echo $?`; `grep -c '@opentelemetry/sdk-node' $BFF/dist/server.js`; `(cd $BFF && runlimit 10 $S/tc03.txt env -i PATH="$PATH" HOME="$HOME" node --import ./dist/instrumentation.js dist/server.js)`; `jsonlines $S/tc03.txt` | Typecheck exit `0`. Lint exit `0`, không có dòng tổng kết problem/warning/error (`0`). Build exit `0`, `dist/` có `instrumentation.js` và `server.js` (có thể thêm `.map`/chunk), không có file test (`0`). Cả hai file qua `node --check` (`0`, `0`). `server.js` không tự khởi động SDK (`0`): SDK chỉ ở entry `--import`. Chạy đúng lệnh `start` khi thiếu `MONGODB_URI`: `exit=1`, không `124`, không `ERR_MODULE_NOT_FOUND`/`ERR_REQUIRE_ESM`. Output toàn JSON (`bad=0`) | ✅ |
| P0-T12-TC04 | Unit | Vitest với in-memory exporter, không cần Docker/collector/core (A10): `(cd $APP && env -i PATH="$PATH" HOME="$HOME" pnpm --filter bff exec vitest run --reporter=verbose) > $S/vt.txt 2>&1; echo $?`; `grep -E 'Test Files\|Tests ' $S/vt.txt`; `for k in traceparent CORE_BASE_URL deep trace_id span timeout 503 OTEL_SERVICE_NAME OTEL_SDK_DISABLED; do printf '%s=%s ' $k $(grep -c -- "$k" $S/vt.txt); done; echo`; `grep -rlE 'InMemorySpanExporter' $BFF/src \| wc -l`; `grep -rnE '\.(skip\|only\|todo)\(' $BFF/src \| wc -l`; `lsof -nP -iTCP:4318 -sTCP:LISTEN \| wc -l` | Exit `0` với env sạch, không có stack (ghi lại cổng 4318 có mở hay không, vì test không được phụ thuộc collector). `Tests` ≥ `61` passed (P0-T11 có 53, thêm ≥ 8), `0` failed/skipped. Tên test phủ, mỗi từ khoá ≥ `1`: span server có `http.route`; outbound gửi `traceparent` cùng trace ID, parent là span client; `deep=1` gọi core và không `deep` thì không gọi; core lỗi/treo trả `503` trong `timeout` ngắn, span `ERROR`; log có `trace_id`/`span_id` khớp span; `CORE_BASE_URL` mặc định và sai scheme (nêu tên, không lộ giá trị); resource mặc định `bff`/`snaptix` và `OTEL_SERVICE_NAME` đè; `OTEL_SDK_DISABLED`. Có dùng `InMemorySpanExporter` (≥ `1`). Không `.skip/.only/.todo` (`0`) | ✅ |
| P0-T12-TC05 | Config (negative) | Config sai thì thoát `1`, nêu tên biến, không lộ giá trị (A3, A5): mỗi lần `(cd $BFF && runlimit 10 $S/c.txt env -i PATH="$PATH" HOME="$HOME" MONGODB_URI=$MURI BFF_PORT=$P <ENV> node --import ./dist/instrumentation.js dist/server.js)` với `<ENV>` = (a) `CORE_BASE_URL=ftp://core:1`; (b) `CORE_BASE_URL=khong-phai-url`; (c) `CORE_BASE_URL=http://`; (d) `OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4318`; (e) `OTEL_EXPORTER_OTLP_ENDPOINT=ftp://x:1 CORE_BASE_URL=ftp://tcuser:tc-pw-SECRET@y:1`. Sau mỗi lần: `jsonlines $S/c.txt`; `jl $S/c.txt 'select(.level=="error" or .level=="fatal")' \| grep -oE 'CORE_BASE_URL\|OTEL_EXPORTER_OTLP_ENDPOINT' \| sort -u \| tr '\n' ' '`; `grep -c 'tc-pw-SECRET' $S/c.txt`; `listen $P \| wc -l`. (f) Không đặt `CORE_BASE_URL` và mọi `OTEL_*`: `bff b5 MONGODB_URI=$MURI BFF_PORT=$P`; `waitcode $B/healthz 200 10`; `bgstop b5`; `jsonlines $S/b5.log`; `grep -cE '^CORE_BASE_URL=\|^OTEL_' $BFF/.env.example` | (a)–(e): `exit=1`, `elapsed` < `5`s, output `bad=0`. Dòng `error`/`fatal` nêu đúng biến sai: (a)(b)(c) `CORE_BASE_URL`, (d) `OTEL_EXPORTER_OTLP_ENDPOINT`, (e) ít nhất một trong hai biến (lý tưởng là cả hai; nếu instrumentation thoát trước khi config bff chạy thì chỉ có `OTEL_EXPORTER_OTLP_ENDPOINT`, ghi nhận). Không lộ mật khẩu (`tc-pw-SECRET` = `0`), không listen (`0`). (f) Env sạch: dùng mặc định, `/healthz` `200`, log `bad=0`. `.env.example` có `CORE_BASE_URL` và ít nhất `OTEL_EXPORTER_OTLP_ENDPOINT` (≥ `2`) | ✅ |
| P0-T12-TC06 | E2E | **Trace BFF → core → PG** với `traceparent` từ client (A1, A6; G10, P0-AT07, P0-NFR2): `corerun`; `waitcode $C/readyz 200 15`; `bff b6 "${BENV[@]}" "${EP[@]}" "${FAST[@]}"`; `waitcode $B/healthz 200 10`; `TID=$(hex 16); SID=$(hex 8)`; `body -H "$(tp $TID $SID)" -H 'X-Request-ID: tc06' "$B/healthz?deep=1"`; `waittrace $TID 30; nap 3`; `xchain $TID`; `tspans $TID \| jq -c '{svc:.resource["service.name"],kind,name,parent,status}'`; `coreacc /readyz`; `jl $S/core.log 'select(.msg=="http request" and .path=="/readyz") \| .trace_id'`; `tspans $TID \| grep -ciE 'snaptix:snaptix@\|password\|authorization\|cookie'`; `bgstop b6; bgstop core` | Body `{"status":"ok","core":"ok"}` (`200`). `xchain`: `traces` = `[$TID]` (một trace duy nhất), `services` ⊇ `bff`, `core`, `ns` = `["snaptix"]`. `bff_server` đúng **1** span: `parent` = `$SID`, `route` = `/healthz`, `code` = `200`, tên chứa `GET` và `/healthz`, status không `ERROR`. `bff_client` ≥ 1 span: `under_server` = `true`, `method` = `GET`, `url`/`host` trỏ `127.0.0.1:18080` và `/readyz`, `code` = `200`. `core_server` có span `GET /readyz` với `parent_is_bff_client` = `true`. `pg_under_core` ≥ `1`. Core nhận đúng **1** request `/readyz` từ lần gọi này, và access log của nó có `trace_id` = `$TID`. Không có mật khẩu DSN hay header nhạy cảm trong trace (`0`) | ✅ |
| P0-T12-TC07 | E2E | Không có `traceparent` vào → bff tự mở trace mới, vẫn nối tới core: `corerun`; `waitcode $C/readyz 200 15`; `bff b7 "${BENV[@]}" "${EP[@]}" "${FAST[@]}"`; `waitcode $B/healthz 200 10`; `code -H 'X-Request-ID: tc07-a' "$B/healthz?deep=1"`; `code -H 'X-Request-ID: tc07-b' "$B/healthz?deep=1"`; `nap 1`; `TA=$(jl $S/b7.log 'select(.request_id=="tc07-a" and .msg=="request completed") \| .trace_id' \| tr -d '"'); TB=$(jl $S/b7.log 'select(.request_id=="tc07-b" and .msg=="request completed") \| .trace_id' \| tr -d '"'); echo "$TA $TB"`; `waittrace $TA 30; waittrace $TB 30; nap 3`; `xchain $TA`; `xchain $TB \| jq -c '.bff_server'`; `bgstop b7; bgstop core` | Hai request `200`. `TA`, `TB` là 32 hex, khác nhau, khác `0…0`. Trace `$TA` lên Tempo: `bff_server` là **root** (`parent` = `""`), có `bff_client` `under_server` = `true`, `core_server` `parent_is_bff_client` = `true`, `pg_under_core` ≥ `1`, `traces` = `[$TA]`. Trace `$TB` cũng có span server bff là root | ✅ |
| P0-T12-TC08 | E2E | Propagation khi **không sample** (`traceparent` flag `00`, sampler parent-based, A5): `corerun`; `waitcode $C/readyz 200 15`; `bff b8 "${BENV[@]}" "${EP[@]}" "${FAST[@]}"`; `waitcode $B/healthz 200 10`; `TID=$(hex 16)`; `code -H "$(tp $TID $(hex 8) 00)" "$B/healthz?deep=1"`; `nap 10`; `tcode $TID; echo`; `jl $S/core.log 'select(.msg=="http request" and .path=="/readyz") \| .trace_id' \| grep -c $TID`; `jl $S/b8.log 'select(.msg=="request completed") \| .trace_id' \| grep -c $TID`; `bgstop b8; bgstop core` | `200`. Tempo **không** có trace `$TID` (`404`) sau 10 s: cả bff và core tôn trọng cờ không sample. Nhưng ngữ cảnh vẫn được truyền: access log core có `trace_id` = `$TID` (`1`), log bff mang cùng `$TID` (`1`) | ✅ |
| P0-T12-TC09 | E2E | Resource và thuộc tính span (A5, A6): (a) `corerun`; `waitcode $C/readyz 200 15`; `bff b9 "${BENV[@]}" "${EP[@]}" "${FAST[@]}"`; `waitcode $B/healthz 200 10`; `TID=$(hex 16)`; `code -H "$(tp $TID $(hex 8))" -H 'Authorization: Bearer tc09-tok-SECRET' -H 'Cookie: sid=tc09-ck-SECRET' "$B/healthz?deep=1"`; `waittrace $TID 30; nap 2`; `tspans $TID \| jq -c 'select(.resource["service.name"]=="bff") \| {kind,name,res:(.resource \| {n:.["service.name"],ns:.["service.namespace"],lang:.["telemetry.sdk.language"]}),route:.attrs["http.route"],code:.attrs["http.response.status_code"]}'`; `tspans $TID \| grep -c 'tc09-'`; `bgstop b9`. (b) Đè bằng env: `bff b9o "${BENV[@]}" "${EP[@]}" "${FAST[@]}" OTEL_SERVICE_NAME=bff-tc09 OTEL_RESOURCE_ATTRIBUTES=deployment.environment.name=tc09`; `waitcode $B/healthz 200 10`; `T2=$(hex 16); code -H "$(tp $T2 $(hex 8))" "$B/healthz?deep=1"`; `waittrace $T2 30; nap 2`; `tspans $T2 \| jq -c 'select(.kind=="SERVER") \| .resource \| {n:.["service.name"],ns:.["service.namespace"],env:.["deployment.environment.name"]}' \| sort -u`; `bgstop b9o; bgstop core` | (a) Mọi span của bff có resource `service.name` = `bff`, `service.namespace` = `snaptix`, `telemetry.sdk.language` = `nodejs`. Span `SERVER` có `route` = `/healthz`, `code` = `200`. Token/cookie không có trong trace (`0`). (b) Span server của bff có `n` = `bff-tc09`, `ns` = `snaptix` (mặc định vẫn giữ), `env` = `tc09`. Span của core vẫn là `core` | ✅ |
| P0-T12-TC10 | Log | `trace_id`/`span_id` trong log pino khớp Tempo (P0-NFR1, A7): `corerun`; `waitcode $C/readyz 200 15`; `bff b10 "${BENV[@]}" "${EP[@]}" "${FAST[@]}"`; `waitcode $B/healthz 200 10`; `TID=$(hex 16)`; `code -H "$(tp $TID $(hex 8))" -H 'X-Request-ID: tc10' "$B/healthz?deep=1"`; `code -H 'X-Request-ID: tc10-plain' $B/healthz`; `waittrace $TID 30; nap 2`; `jsonlines $S/b10.log`; `jl $S/b10.log 'select(.request_id=="tc10") \| {msg,trace_id,span_id}'`; `bffids $TID > $S/ids10`; `jl $S/b10.log 'select(.request_id=="tc10") \| .span_id' \| tr -d '"' \| while read x; do grep -qx "$x" $S/ids10 && echo in \|\| echo OUT; done \| sort \| uniq -c`; `jl $S/b10.log 'select(.request_id=="tc10-plain") \| .trace_id' \| sort -u \| wc -l`; `jl $S/b10.log 'select(.request_id!=null and (.trace_id==null or .span_id==null))' \| wc -l`; `bgstop b10; bgstop core` | Log `bad=0`. Request `tc10` có ít nhất `incoming request` và `request completed`. Mọi dòng có `trace_id` = `$TID` (32 hex) và `span_id` (16 hex). Mọi `span_id` là span của bff trong trace `$TID` trên Tempo (`in`, không `OUT`). Request `tc10-plain` (không `traceparent`) có đúng **1** `trace_id` chung cho các dòng của nó. Không có dòng log mang `request_id` mà thiếu `trace_id`/`span_id` (`0`) | ✅ |
| P0-T12-TC11 | E2E | `/healthz` thường không gọi core, `/readyz` không đổi (A2): `corerun`; `waitcode $C/readyz 200 15`; `bff b11 "${BENV[@]}" "${EP[@]}" "${FAST[@]}"`; `waitcode $B/healthz 200 10`; `N0=$(coreacc /readyz)`; `TID=$(hex 16)`; `body -H "$(tp $TID $(hex 8))" $B/healthz`; `code "$B/healthz?deep=0"`; `code "$B/healthz?x=1"`; `body $B/readyz`; `nap 1; echo "$N0 -> $(coreacc /readyz)"`; `waittrace $TID 30; nap 2`; `tspans $TID \| jq -c '{svc:.resource["service.name"],kind,name}'`; `bgstop core`; `ctime $B/healthz`; `bgstop b11` | `/healthz` → `{"status":"ok"}` (`200`), `deep=0` và `x=1` `200`. `/readyz` (Mongo) `200 {"status":"ok"}`. Core không nhận thêm `/readyz` nào (`N0 -> N0`). Trace `$TID` chỉ có span của `bff`, không có span `CLIENT` hay span core. Khi core đã dừng, `/healthz` vẫn `200` < `1`s | ✅ |
| P0-T12-TC12 | E2E (negative) | Core **không chạy** (từ chối kết nối) → `503` nhanh, span lỗi (A2, A6): `bgstop core`; `listen $CP \| wc -l`; `bff b12 "${BENV[@]}" "${EP[@]}" "${FAST[@]}"`; `waitcode $B/healthz 200 10`; `TID=$(hex 16)`; `curl -s -m 10 -w ' %{http_code} %{time_total}s\n' -H "$(tp $TID $(hex 8))" -H 'X-Request-ID: tc12' "$B/healthz?deep=1"`; `for i in 1 2 3; do ctime "$B/healthz?deep=1"; done`; `waittrace $TID 30; nap 2`; `xchain $TID \| jq -c '{bff_server,bff_client}'`; `jl $S/b12.log 'select(.request_id=="tc12" and .level=="warn") \| {msg,trace_id,request_id}'`; `jsonlines $S/b12.log`; `grep -ciE 'unhandled\|uncaught' $S/b12.log`; `alive b12`; `code $B/healthz`; `bgstop b12` | Cổng core trống (`0`). Body `{"status":"unavailable","core":"unavailable"}`, `503`, `time_total` < `1`s. Ba lần sau đều `503` < `1`s. Span server bff: `code` = `503`, status `ERROR`. Span client (nếu instrumentation tạo cho lỗi kết nối): status `ERROR`. Có dòng `warn` với `request_id` = `tc12`, `trace_id` = `$TID`. Log `bad=0`, không có unhandled/uncaught (`0`). Process `alive`, `/healthz` `200` | ✅ |
| P0-T12-TC13 | E2E (negative) | Core **treo** (nhận kết nối, không trả lời) → `503` trong hạn 2 s, không treo (A2): `hang`; `nap 0.5`; `bff b13 MONGODB_URI=$MURI BFF_PORT=$P CORE_BASE_URL=http://127.0.0.1:${HP} "${EP[@]}" "${FAST[@]}"`; `waitcode $B/healthz 200 10`; `TID=$(hex 16)`; `curl -s -m 10 -w ' %{http_code} %{time_total}s\n' -H "$(tp $TID $(hex 8))" -H 'X-Request-ID: tc13' "$B/healthz?deep=1"`; `for i in 1 2 3; do ctime "$B/healthz?deep=1"; done`; `ctime $B/healthz`; `waittrace $TID 30; nap 2`; `xchain $TID \| jq -c '{bff_server,bff_client}'`; `jl $S/b13.log 'select(.request_id=="tc13" and .level=="warn") \| {msg,trace_id}'`; `jsonlines $S/b13.log`; `grep -ciE 'unhandled\|uncaught' $S/b13.log`; `alive b13`; `bgstop b13; bgstop hang` | Mỗi `deep=1` trả `503` body `{"status":"unavailable","core":"unavailable"}`, `time_total` trong khoảng `1.9`–`3`s (hạn 2 s, không treo tới 10 s của curl). `/healthz` thường vẫn `200` < `1`s. Span server bff `503` `ERROR`, span client status `ERROR`. Có dòng `warn` `trace_id` = `$TID`. Log `bad=0`, không có unhandled/uncaught (`0`). Process `alive` | ✅ |
| P0-T12-TC14 | E2E (negative) | Core trả `503` (PG dừng) → bff `503`, **không retry**, tự hồi phục (A2): `corerun`; `waitcode $C/readyz 200 15`; `bff b14 "${BENV[@]}" "${EP[@]}" "${FAST[@]}"`; `waitcode $B/healthz 200 10`; `dc stop postgres-core`; `N0=$(coreacc /readyz)`; `TID=$(hex 16)`; `curl -s -m 10 -w ' %{http_code} %{time_total}s\n' -H "$(tp $TID $(hex 8))" "$B/healthz?deep=1"`; `nap 1; echo "$N0 -> $(coreacc /readyz)"`; `waittrace $TID 30; nap 2`; `xchain $TID \| jq -c '{bff_server,bff_client,core_server}'`; `dc start postgres-core`; `dc up -d --wait postgres-core`; `waitcode "$B/healthz?deep=1" 200 30`; `bgstop b14; bgstop core` | Khi PG dừng: `503` `{"status":"unavailable","core":"unavailable"}`, `time_total` ≤ `3`s. Core nhận **đúng 1** `/readyz` cho request đó (`N0 -> N0+1`, không retry). Trace có span client bff `code` = `503` status `ERROR`, span core `GET /readyz` với `parent_is_bff_client` = `true`, span server bff `503` `ERROR`. Sau khi PG chạy lại, `deep=1` lên `200` trong ≤ 30 s mà không restart bff/core | ✅ |
| P0-T12-TC15 | E2E (negative) | **Collector chết** → bff vẫn phục vụ, không in rác, tự gửi lại khi collector lên (A7): `corerun`; `waitcode $C/readyz 200 15`; `dc stop otel-collector`; `bff b15 "${BENV[@]}" "${EP[@]}" "${FAST[@]}"`; `waitcode $B/healthz 200 10`; `for i in 1 2 3; do ctime "$B/healthz?deep=1"; done`; `nap 8`; `alive b15`; `ctime $B/healthz`; `jsonlines $S/b15.log`; `grep -ciE 'unhandled\|uncaught' $S/b15.log`; `jl $S/b15.log 'select(.level=="warn" or .level=="error") \| .msg' \| sort \| uniq -c`; `dc start otel-collector`; `dc up -d --wait otel-collector`; `nap 3`; `TID=$(hex 16)`; `code -H "$(tp $TID $(hex 8))" "$B/healthz?deep=1"`; `waittrace $TID 30`; `xchain $TID \| jq -c '.services'`; `bgstop b15; bgstop core` | Khi collector dừng: bff khởi động được, `deep=1` `200` với `time_total` < `1`s (export không chặn request). Sau 8 s (nhiều chu kỳ export lỗi) process vẫn `alive`, `/healthz` `200`. Log `bad=0`: không có dòng non-JSON (stack thô, `console.error` của exporter). Không có unhandled/uncaught (`0`). Ghi nhận các dòng `warn`/`error` JSON về export, nếu có. Collector lên lại thì **không cần restart** bff: trace `$TID` lên Tempo trong ≤ 30 s, có cả `bff` và `core` | ✅ |
| P0-T12-TC16 | E2E | `OTEL_SDK_DISABLED=true` (A5): `corerun`; `waitcode $C/readyz 200 15`; `bff b16 "${BENV[@]}" "${EP[@]}" "${FAST[@]}" OTEL_SDK_DISABLED=true`; `waitcode $B/healthz 200 10`; `TID=$(hex 16)`; `body -H "$(tp $TID $(hex 8))" "$B/healthz?deep=1"`; `nap 10`; `tcode $TID; echo`; `tspans $TID 2>/dev/null \| jq -r '.resource["service.name"]' \| sort -u`; `jsonlines $S/b16.log`; `jl $S/b16.log 'select(.msg=="request completed") \| .trace_id' \| tail -1`; `bgstop b16; bgstop core` | bff chạy bình thường, `deep=1` `200 {"status":"ok","core":"ok"}`. Trên Tempo không có span nào của `bff` cho `$TID`: `404`, hoặc `200` chỉ có span `core` (core tự export nếu `traceparent` được chuyển tiếp). Ghi nhận trường hợp nào. Log `bad=0`. Ghi nhận `trace_id` trong log khi SDK tắt (có `$TID` hoặc không có khoá) | ✅ |
| P0-T12-TC17 | Metric | Metric RED của bff vào Prometheus và dashboard `snaptix-red` (A8): `corerun`; `waitcode $C/readyz 200 15`; `bff b17 "${BENV[@]}" "${EP[@]}" "${FAST[@]}"`; `waitcode $B/healthz 200 10`; `for i in $(seq 30); do code $B/healthz; code "$B/healthz?deep=1"; code $B/readyz; code "$B/khong-co-$i"; nap 0.2; done >/dev/null`; `bgstop core`; `code "$B/healthz?deep=1"` (503 đầu tiên, sinh series `http_response_status_code="503"`); `nap 3` (> 1 chu kỳ export 2 s, để series 503 có mẫu đầu); `for i in $(seq 12); do code "$B/healthz?deep=1"; nap 0.5; done >/dev/null` (503 rải qua ≥ 3 chu kỳ export); `waitprom 'http_server_request_duration_seconds_count{job="snaptix/bff"}' 40`; `nap 15`; `prom 'sum by (http_route, http_request_method, http_response_status_code) (http_server_request_duration_seconds_count{job="snaptix/bff"})'`; `prom 'count(count by (http_route) (http_server_request_duration_seconds_count{job="snaptix/bff"}))'`; `prom 'sum(http_server_request_duration_seconds_sum{job="snaptix/bff",http_route="/healthz"}) / sum(http_server_request_duration_seconds_count{job="snaptix/bff",http_route="/healthz"})'`; `prom 'count(http_server_request_duration_seconds_count{job="bff"})'`; `python3 $S/dsq.py $RED --var`; `python3 $S/dsq.py $RED snaptix/bff`; `bgstop b17` | Có series `job="snaptix/bff"` trong ≤ 40 s. Theo route: `/healthz` có status `200` và `503` (lần deep khi core dừng), `/readyz` `200`, method `GET`. Route 404 **không** mang path thô: tổng số giá trị `http_route` ≤ `3` (`/healthz`, `/readyz`, và tuỳ chọn một nhãn cố định/rỗng cho 404), không có `/khong-co-*`. Trung bình duration `/healthz` là số dương < `1` (đơn vị giây, không phải ms). Không có series `job="bff"` (thiếu namespace) (kết quả rỗng). `--var` có `snaptix/bff` (và vẫn có `snaptix/core`). Với `snaptix/bff`, mọi target của `red.json` có giá trị, không có `NO DATA`/`ERROR`. Panel Error rate (5xx) (đo bằng `dsq` như trên) > `0` vì có 503. Lý do kịch bản rải 503: `rate()` chỉ > 0 khi series 503 có ≥ 2 mẫu khác nhau trong cửa sổ; nếu mọi 503 rơi vào cùng một chu kỳ export OTLP thì mẫu đầu tiên của series đã là tổng cuối, `rate()` = 0 dù code đúng | ✅ |
| P0-T12-TC18 | Dev | Chế độ dev (`tsx watch`) cũng có trace xuyên service (A4): `corerun`; `waitcode $C/readyz 200 15`; `(cd $APP && bgrun dev env -i PATH="$PATH" HOME="$HOME" "${BENV[@]}" "${EP[@]}" "${FAST[@]}" pnpm --filter bff dev)`; `waitcode $B/healthz 200 30`; `TID=$(hex 16)`; `code -H "$(tp $TID $(hex 8))" -H 'X-Request-ID: tc18' "$B/healthz?deep=1"`; `waittrace $TID 30; nap 2`; `xchain $TID \| jq -c '{services,bff_server:[.bff_server[]\|{parent,route}],bff_client:[.bff_client[]\|.under_server],core:[.core_server[]\|.parent_is_bff_client],pg_under_core}'`; `jsonlines $S/dev.log only`; `jl $S/dev.log 'select(.request_id=="tc18") \| .trace_id' \| sort -u`; `bgstop dev; bgstop core`; `nap 1; listen $P \| wc -l`; `pgrep -f 'tsx watch' \| wc -l` | Dev lên trong ≤ 30 s, `deep=1` `200`. Trace `$TID` có `services` ⊇ `bff`, `core`. Span server bff có `route` = `/healthz` và parent là span id đã gửi. Span client `under_server` = `true`. Core `parent_is_bff_client` = `true`, `pg_under_core` ≥ `1`. Nghĩa là instrumentation được nạp trước fastify cả khi chạy bằng tsx. Log ứng dụng (dòng `{`) `bad=0`, request `tc18` có `trace_id` = `$TID`. Sau `bgstop`, cổng trống và không còn `tsx watch` (`0`, `0`) | ✅ |
| P0-T12-TC19 | UI | Xem trên Grafana bằng Claude in Chrome (G10, P0-AT07, DoD). Chỉ dùng `localhost`, mở **tab mới** (`tabs_create_mcp`), không đụng tab khác, đóng tab khi xong (`tabs_close_mcp`). Chuẩn bị: `corerun`; `waitcode $C/readyz 200 15`; `bff b19 "${BENV[@]}" "${EP[@]}" "${FAST[@]}"`; `waitcode $B/healthz 200 10`; `TID=$(hex 16)`; `code -H "$(tp $TID $(hex 8))" "$B/healthz?deep=1"`; `for i in $(seq 30); do code $B/healthz; code "$B/healthz?deep=1"; nap 0.3; done >/dev/null`; `waittrace $TID 30; nap 15`; `exploreurl $TID`. (1) Mở URL từ `exploreurl`, chờ tải, mở rộng cây span, chụp màn hình. (2) Mở `http://localhost:3100/d/snaptix-red?var-service=snaptix%2Fbff&from=now-15m&to=now`, chờ panel tải, chụp màn hình. Xong: đóng tab; `bgstop b19; bgstop core` | (1) Explore hiện **một** trace `$TID` dạng cây: span server `GET /healthz` của service `bff` → span client HTTP của `bff` → `GET /readyz` của service `core` → (các) span PG (`pool.acquire`/`connect`/…) của `core`. Tức BFF → core → PG trong một trace. (2) Dashboard RED với `service` = `snaptix/bff`: panel Rate, Errors, Duration có dữ liệu, không hiện "No data". Có ảnh chụp cả hai. Tab đã đóng | ✅ |
| P0-T12-TC20 | Hồi quy | `make test` suite app, script CI, không đụng server: `bash scripts/test-all.sh app > $S/tc20.txt 2>&1; echo $?`; `grep -E 'Tests \|Test Files\|FAIL\|bước lỗi' $S/tc20.txt`; `bash scripts/test-all.sh scripts; echo $?`; `bash scripts/ci-app_test.sh; echo $?`; `bash scripts/ci-changes_test.sh; echo $?`; `git -C $REPO status --porcelain --untracked-files=all \| grep -E 'dist/\|node_modules\|tsbuildinfo\|\.log$' \| wc -l` | `test-all.sh app` exit `0` (install frozen, lint, typecheck, test, build của bff đều pass, không `FAIL`). `test-all.sh scripts` exit `0`. `ci-app_test.sh`, `ci-changes_test.sh` exit `0`. Không sinh file rác ngoài ignore (`0`). Không chạy suite server vì task không sửa `com/tm/server` (TC22) | ✅ |
| P0-T12-TC21 | Tài liệu | Docs khớp code: `grep -nE 'CORE_BASE_URL' com/tm/docs/technical/local-setup.md`; `grep 'OTEL_' com/tm/docs/technical/local-setup.md \| grep -c bff`; `grep -nE 'deep=1\|--import\|instrumentation' com/tm/docs/technical/local-setup.md`; `grep -nE 'SDK Node\|snaptix/bff\|instrumentation\|@fastify/otel\|instrumentation-undici\|instrumentation-http' com/tm/docs/technical/architecture.md`; `grep -nE 'instrumentation.ts\|core-client' com/tm/docs/technical/project-structure.md`; `ls com/tm/docs/technical/handbook/phase-0/P0-T12.md && grep -c 'P0-T12' com/tm/docs/technical/handbook/README.md`; `grep -n 'P0-AT07' com/tm/docs/technical/planning/phase-0-foundation/acceptance-tests.md`; `grep -n '^| G10' com/tm/docs/technical/planning/phase-0-foundation/README.md` | `local-setup.md`: `CORE_BASE_URL` ghi mặc định và quy tắc validate cho bff. Bảng `OTEL_*` ghi rõ áp dụng cho cả bff (mặc định `service.name` = `bff`), tức các dòng `OTEL_*` có nhắc `bff` (≥ `1`). Có hướng dẫn chạy bff với OTel (`start`/`dev` nạp `--import` instrumentation) và thử `curl localhost:3000/healthz?deep=1` để xem trace trên Grafana. `architecture.md` có đoạn SDK Node: entry `--import`, instrumentation đã chọn, resource `bff`/`snaptix` → `job="snaptix/bff"`, route `/healthz?deep=1` → core `/readyz`. Cây `project-structure.md` có `instrumentation.ts` (và `core-client/` nếu dùng). Handbook `P0-T12.md` tồn tại, `handbook/README.md` có P0-T12 ở cả hai bảng (≥ `2`). P0-AT07 ✅ và G10 ✅ sau khi TC06 + TC19 pass | ✅ |
| P0-T12-TC22 | Phạm vi | Không làm việc của task khác (A2, A11): `git -C $REPO status --porcelain --untracked-files=all`; `git -C $REPO status --porcelain com/tm/server deploy scripts .github Makefile \| wc -l`; `ls -A $APP/packages $APP/apps`; `grep -rnE "process\.on\(['\"](SIGTERM\|SIGINT)\|process\.once\(['\"](SIGTERM\|SIGINT)\|close-with-grace" $BFF/src \| wc -l`; `grep -rniE "retry\|circuit\|breaker\|/internal/v1\|X-Service-Token\|CORE_SERVICE_TOKEN\|X-User-Id" $BFF/src \| wc -l`; `ls $BFF/Dockerfile 2>/dev/null \| wc -l`; `git -C $REPO diff --stat -- com/tm/app/apps/bff/src/plugins/mongo.ts` | Thay đổi chỉ nằm trong: `com/tm/app/apps/bff/**`, `com/tm/app/pnpm-lock.yaml`, `com/tm/app/pnpm-workspace.yaml` (chỉ khi duyệt build script), `com/tm/docs/technical/{local-setup.md,architecture.md,project-structure.md,api.md}`, handbook (`phase-0/P0-T12.md`, `README.md`), planning (`planning/README.md`, `phase-0-foundation/README.md`, `acceptance-tests.md`, `tasks/P0-T12/**`), `planning/execute-all-notes.md`. `com/tm/server`, `deploy`, `scripts`, `.github`, `Makefile` không đổi (`0`). `packages/` chỉ có `.gitkeep`, `apps/` chỉ có `bff`. Không bắt tín hiệu (`0`, P0-T13). Không retry, circuit breaker, service token, API nội bộ (`0`, P2-T05). Không có Dockerfile bff (`0`). Plugin mongo không đổi hành vi | ✅ |
| P0-T12-TC23 | Dọn dẹp | `for n in b5 b6 b7 b8 b9 b9o b10 b11 b12 b13 b14 b15 b16 b17 b19 dev core hang; do bgstop $n; done`; `listen $P \| wc -l; listen $CP \| wc -l; listen $HP \| wc -l; listen 3000 \| wc -l`; `pgrep -f 'dist/server.js' \| wc -l`; `pgrep -f 'tsx watch' \| wc -l`; `pgrep -f "$S/core" \| wc -l`; `dc ps --format '{{.Service}} {{.State}}'`; `dc down`; `docker ps -q --filter label=com.docker.compose.project=snaptix \| wc -l \| tr -d ' '`; `rm -rf $S; ls -d $S 2>/dev/null \| wc -l \| tr -d ' '`. Tab Chrome của TC19 đã đóng | Không còn process bff/tsx/core/core-giả (`0`, `0`, `0`). Cổng 13000, 18080, 18081, 3000 trống (`0`×4). Trước `down`, mọi service (kể cả `postgres-core`, `otel-collector` sau TC14/TC15) ở trạng thái `running`, không bị bỏ ở trạng thái dừng. Sau `dc down` không còn container của project (`0`), volume giữ nguyên (không `-v`). Scratchpad đã xoá (`0`). Không còn tab Chrome do test mở | ✅ |

Test nghiệm thu liên quan:
- **P0-AT07** (gọi `bff /healthz?deep=1`, một trace chứa span bff → core → PG, P0-NFR2, G10): TC06 kiểm qua Tempo API, TC19 kiểm trên Grafana, TC18 kiểm với dev. Task này làm AT07 pass → ✅ khi TC06 và TC19 pass. Kéo theo **G10** → ✅ ("Một trace hiển thị đủ BFF → core → PG").
- **P0-NFR1** (log JSON có `trace_id`, `request_id`), phần bff: TC10. P0-AT06 (log core) đã ✅ từ trước và không đổi.
- **P0-NFR2** (`traceparent` từ bff sang core): TC06, TC07, TC08.
- **DoD** "Gọi `bff /healthz?deep=1` → thấy một trace gồm span của bff và core trên Grafana": TC19.
- **P0-AT05 / P0-AT11** (CI app, không chạy Bazel khi chỉ sửa `com/tm/app/**`): TC20 kiểm cục bộ, TC22 xác nhận không đụng `com/tm/server`. CI thật chạy khi đóng phase, để ⬜.

## Review

### Review lần 1 — APPROVED

Review agent độc lập (execute-all), 2026-09-27. Đối chiếu dòng P0-T12, G10, DoD, P0-NFR1/NFR2, P0-AT07 (nguyên văn), ghi chú execute-all P0-T10/P0-T11, code `apps/bff` hiện tại, `red.json`, access log core (`httpx/logging.go`).

**Phủ phạm vi**: đủ dòng task (OTel Node, gọi core, trace xuyên service), G10/AT07 ("một trace chứa span bff → core → PG": TC06, TC18, TC19), DoD (TC19), NFR1 (TC10), NFR2 (TC06–TC08). Ranh giới P0-T13/P0-T15/P0-T17/P2-T05 được nêu rõ và có kiểm (TC22). Không vượt phạm vi.

**Kiểm chứng trong scratchpad** (workspace pnpm 11.18.0 tạm, fastify 5.12, `@opentelemetry/sdk-node` 0.222.0, `@fastify/otel` 0.21.0, `instrumentation-undici` 0.32.0, `instrumentation-http` 0.222.0, tsup ESM 2 entry, core build bằng `go build`, stack `otel-collector tempo prometheus postgres-core`; đã `down` không `-v`, dọn process/cổng/scratchpad):
- `node --import ./dist/instrumentation.js dist/server.js`: span `SERVER` duy nhất của `instrumentation-http` tên `GET /healthz`, có `http.route=/healthz` (route do `@fastify/otel` cấp qua RPC metadata). `@fastify/otel` chỉ sinh span `INTERNAL` (`request`, `handler - …`), nên không làm hỏng điều kiện "đúng 1 span SERVER".
- `fetch` global → span `CLIENT` của undici, `traceparent` gửi đi cùng trace ID, span id của chính span client.
- Trên Tempo: một trace có `bff` (SERVER, INTERNAL×2, CLIENT) → core `GET /readyz` (SERVER) → `pool.acquire` (+ `connect` ở lần đầu), `pool.acquire` có `db.system.name=postgresql`, nên `pg_under_core` ≥ 1 cả khi pool đã có kết nối sẵn. **A1 hợp lý và cần thiết**: core `/healthz` không có span PG.
- **A8 khả thi**: có `http.server.request.duration` với `http.route` (đặt `OTEL_SEMCONV_STABILITY_OPT_IN=http` trong code), 404 không có nhãn `http_route` (không path thô). Prometheus có `http_server_request_duration_seconds_count{job="snaptix/bff"}`.
- `OTEL_SDK_DISABLED=true`: NodeSDK tôn trọng, không export, cũng **không** chuyển tiếp `traceparent` (core nhận `None`) → TC16 sẽ rơi vào nhánh `404`.
- Collector chết (endpoint cổng đóng): request vẫn ~3–40 ms, process **không in gì** ra stdout/stderr (không có diag logger mặc định).
- `tsx watch --import ./src/instrumentation.ts src/server.ts`: có span `GET /healthz` + `http.route` + span client, `traceparent` tới core.
- Vitest 5: `NodeSDK` + `InMemorySpanExporter` + import động fastify sau `sdk.start()`, listen cổng 0 → có span `GET /healthz` kind SERVER với `http.route` (A10 khả thi).
- Exporter `*-otlp-proto` kéo `protobufjs` có build script → pnpm 11 báo `ERR_PNPM_IGNORED_BUILDS`; pnpm tự chèn dòng `protobufjs: set this to true or false` vào `pnpm-workspace.yaml`. A9 và TC02 (`grep -c 'set this to'` = 0) đã bắt đúng. Sau khi đặt `protobufjs: false` thì install/`--frozen-lockfile` sạch, không WARN/deprecated.

**Giả định**:
- A1: lệch chữ với dòng task ("gọi thử `core /healthz`") nhưng khớp mục đích (health check của core) và là cách duy nhất có span PG mà không sửa `com/tm/server`; ghi chú P0-T10 đã mở đường ("vd gọi `/readyz`"). TC21 buộc ghi vào `architecture.md`. Chấp nhận.
- A2 (hạn 2 s, gọi một lần, không retry, body 503), A3 (`CORE_BASE_URL` mặc định `http://localhost:8080`, khớp local-setup), A5, A7, A9: có căn cứ, không ràng buộc vô lý.
- A4: cấm `instrumentation-fastify` có căn cứ (npm đánh dấu deprecated "in favor of @fastify/otel"). Cấm `auto-instrumentations-node` là lựa chọn thiết kế (bật tối thiểu, tránh span mongodb/fs/dns ngoài ý muốn và dependency thừa), không làm task fail vô lý vì cách thay thế đã rõ và đã kiểm chứng. Tên file có lối thoát (sửa `bff()` + ghi chú).

**Góp ý không chặn** (người làm lưu ý khi thực thi):
1. A4 "instrumentation-http **và/hoặc** `@fastify/otel`": thực tế chỉ `@fastify/otel` thì **không** có span `SERVER` (chỉ `INTERNAL`) và không trích `traceparent` → TC06/TC09/TC10 fail. Nên dùng cả hai (đã kiểm chứng) — A6 và TC đã ràng buộc đúng.
2. TC22 `grep -rniE "retry|circuit|…" $BFF/src` = `0` cũng bắt cả comment/tên test như "không retry". Tránh các từ này trong comment/tên test (TC14 đã kiểm "không retry" ở E2E), hoặc ghi nhận nếu chỉ là comment/tên test khi chạy.
3. TC06 "core nhận đúng 1 request `/readyz` từ lần gọi này": `coreacc /readyz` còn đếm lần `waitcode $C/readyz`; khi chạy, đối chiếu bằng lệnh `trace_id` kế tiếp (đúng một dòng mang `$TID`).
4. TC10/A7: nếu dùng `mixin` đọc `trace.getActiveSpan()`, kiểm dòng `request completed` (hook `onResponse`) vẫn còn context; nếu mất thì dùng child logger gắn ở `onRequest`.
5. TC18/TC23 `pgrep -f 'tsx watch'` có thể đếm cả `tsx watch` của người dùng chạy ngoài test; nếu ≠ 0 thì kiểm thêm theo cwd/pid group trước khi kết luận fail.
6. Nên ghi thêm quyết định A1 (bff gọi core `/readyz` thay vì `/healthz`) vào `execute-all-notes.md` mục P0-T12 để báo cáo cuối thấy chỗ lệch chữ với dòng task.

### Sửa lần 1 (sau khi chạy bước 5)

Agent sinh test case (execute-all), 2026-09-27. Chỉ sửa **TC17**, không đổi ID, không đổi TC khác, không đổi hành vi yêu cầu với code. Chờ review lại.

- **Lý do** (kết quả chạy thật, xem `## Ghi chú thực thi` mục TC17): chạy đúng kịch bản cũ thì mọi vế khớp trừ "Error rate > 0" (đo được `0`). Không phải lỗi code: series `http_response_status_code="503"` chỉ xuất hiện khi có request 503 đầu tiên; 3 request `deep=1` gửi liền nhau rơi vào cùng một chu kỳ export OTLP 2 s (`OTEL_METRIC_EXPORT_INTERVAL=2000` trong `FAST`), nên mẫu đầu của series đã là `3`, mọi mẫu sau cũng `3`, `rate()` = 0. Kiểm bổ sung: 24 request 503 cách nhau 0.5 s thì `dsq` Error rate = `0.5`. Kịch bản cũ không kiểm được điều nó định kiểm.
- **TC17 đổi gì**:
  - Kịch bản: sau `bgstop core`, thay `for i in 1 2 3; do code "$B/healthz?deep=1"; done` bằng: 1 request `deep=1` (tạo series 503) → `nap 3` (> 1 chu kỳ export) → 12 request `deep=1` cách nhau 0.5 s (503 rải qua ≥ 3 chu kỳ export). Các bước sau (`waitprom`, `nap 15`, query, `dsq`) giữ nguyên. Metric đi vào Prometheus qua OTLP push (không phụ thuộc `scrape_interval` 15 s), cửa sổ `$__rate_interval` của dashboard trong khoảng `now-5m` vẫn phủ các mẫu này.
  - Kết quả mong đợi: giữ mọi vế cũ; vế Error rate giữ mức `> 0`, đo bằng `dsq` như cũ, ghi thêm lý do kỹ thuật (`rate()` cần ≥ 2 mẫu khác nhau của series 503 trong cửa sổ).
- Kiểm cú pháp: khối Chuẩn bị + lệnh TC17 đã sửa qua `bash -n` và `zsh -n`.

**Định dạng**: tiêu đề đúng, bảng 5 cột, ID `P0-T12-TC01`..`TC23` liên tục, trạng thái ⬜, có "Test nghiệm thu liên quan". Lệnh chạy được trên bash 3.2/zsh (biến `${…}`, không `status`/`path`, `runlimit`/`nap` bằng python), không có lệnh ghi lịch sử git. TC19 chỉ `localhost`, tab mới, đóng tab.

### Review lần 2 — APPROVED

Review agent độc lập (execute-all), 2026-09-27. Chỉ xét thay đổi ở TC17 (Sửa lần 1).

1. **Phạm vi sửa**: kịch bản TC17 chỉ thay đoạn tạo 503 sau `bgstop core` (1 request tạo series → `nap 3` → 12 request cách 0.5 s). Phần trước (`corerun`, vòng 30 lần 200/404) và phần sau (`waitprom … 40`, `nap 15`, 4 query `prom`, `dsq --var`, `dsq snaptix/bff`, `bgstop b17`) giữ nguyên. Kết quả mong đợi còn đủ các vế mà Ghi chú thực thi đã đối chiếu: series ≤ 40 s, `/healthz` 200 + 503, `/readyz` 200, GET, `http_route` ≤ 3 và không có path thô, trung bình duration dương < 1 s, `job="bff"` rỗng, `--var` có `snaptix/bff` + `snaptix/core`, mọi target không `NO DATA`/`ERROR`. Vế Error rate vẫn `> 0` (không nới), đo bằng `dsq` như cũ. Chỉ thêm câu giải thích lý do.
2. **`rate()` > 0 có ổn định không** (suy luận trên cấu hình thật):
   - Panel Error rate trong `red.json` là `sum by (job)(rate(...{http_response_status_code=~"5.."}[$__rate_interval])) or … * 0` chia cho tổng rate.
   - Datasource Prometheus provision **không** đặt `timeInterval`, nên Grafana dùng scrape interval mặc định 15 s. `dsq` query instant với khoảng `now-5m`, nên `$__rate_interval` = max(interval + 15 s, 4 × 15 s) = **60 s**.
   - Mốc thời gian: gọi t0 là lúc có 503 đầu tiên. Mẫu đầu của series 503 (giá trị 1) ở chu kỳ export kế tiếp, tức ≤ t0+2 s (thêm ≤ 200 ms do `batch` của collector). `nap 3` bảo đảm ít nhất một mẫu nữa vẫn là 1. 12 request trong ~6–7 s (503 nhanh vì bị từ chối kết nối) làm counter tăng qua ≥ 3 chu kỳ, lên 13 lúc khoảng t0+10 s. Sau đó bff vẫn export giá trị 13 mỗi 2 s.
   - `waitprom` trả về ngay (series có từ vòng 30 lần). Sau `nap 15` và vài query, `dsq` chạy lúc khoảng t0+26–30 s. Cửa sổ 60 s là [≈t0−34, ≈t0+30], phủ trọn đoạn tăng từ 1 → 13 và có nhiều mẫu. Còn dư khoảng 30 s trước khi mẫu đầu rơi khỏi cửa sổ. Không có rủi ro series 503 chỉ có 1 mẫu, hay chỉ có mẫu phẳng.
   - Kiểm chứng thực nghiệm trong Ghi chú (24 × 503 / 0.5 s → `0.5`) cùng cơ chế, khớp suy luận.
3. **Cú pháp**: đoạn lệnh mới qua `bash -n` (GNU bash 3.2.57) và `zsh -n`. Dùng `$(seq 12)`, `nap`, không có biến trùng tên đặc biệt. Không có lệnh ghi lịch sử git.
4. **Định dạng**: dòng TC17 vẫn 5 cột (không có `|` chưa escape), ID `TC01`..`TC23` đủ và không đổi, TC17 vẫn ⬜, các TC khác không bị sửa. Mục Review lần 1 và Ghi chú thực thi còn nguyên.

**Góp ý không chặn**:
- Kịch bản còn phụ thuộc giả định `$__rate_interval` ≥ ~30 s (đúng với datasource hiện tại). Nếu sau này datasource đặt `timeInterval` nhỏ (vd 5 s → cửa sổ 20 s), có thể cần rút `nap 15`. Khi chạy lại nên ghi giá trị Error rate đo được vào Ghi chú.
- Lần chạy lại TC17 nên dùng bff mới (`b17`), để không lẫn series 503 còn lại từ lần kiểm bổ sung trước đó.

## Ghi chú thực thi

Agent triển khai (execute-all), 2026-09-27, macOS arm64, Node 22.22.3, pnpm 11.18.0, stack qua `dc up -d --wait`, core build bằng `go build`. `prep.sh` lấy nguyên văn khối Chuẩn bị. Tên entry giữ `src/instrumentation.ts`, nên không sửa `bff()`.

- **TC04**: 104 test pass (53 → 104). Lúc chạy TC04, cổng 4318 đang mở (stack chạy). Sau `dc down` ở TC23 (4318 đóng, `0`) chạy lại `vitest run` với env sạch: vẫn 104 pass. Test không phụ thuộc collector.
- **TC05 (e)**: chỉ nêu `OTEL_EXPORTER_OTLP_ENDPOINT`, vì `instrumentation.ts` thoát trước khi config bff chạy (nhánh A5 đã dự kiến). Không lộ `tc-pw-SECRET`.
- **TC06**: `coreacc /readyz` = `2` (1 lần từ `waitcode $C/readyz`, 1 lần từ request). Theo góp ý 3 của review, đối chiếu bằng `trace_id`: đúng 1 dòng access log core mang `$TID`. Trace mẫu `d22153d797a605be02eac2720fa5923d`: bff `GET /healthz` (SERVER, parent = SID gửi vào) → `request` → `handler - healthRoutes` → `GET` (CLIENT, `http://127.0.0.1:18080/readyz`) → core `GET /readyz` → `pool.acquire` (`pg_under_core` = 1).
- **TC13**: `time_total` 2.007–2.008 s. Dòng warn có `error_type` = `TimeoutError`: `withTimeout` của route về trước `AbortSignal.timeout` cùng hạn. Span client vẫn ERROR vì fetch bị abort.
- **TC15**: khi collector dừng, log có 2 dòng JSON `level=error`, `component=otel`, msg `otel: PeriodicExportingMetricReader: metrics export timed out after 2000ms`. Không có dòng non-JSON. Lỗi export trace không in gì.
- **TC16**: rơi vào nhánh `404`. SDK tắt thì không chuyển tiếp `traceparent`: core ghi trace khác `$TID`. Log `request completed` **không có khoá** `trace_id`.
- **TC17**:
  - Lịch sử: lần chạy theo kịch bản cũ (3 request 503 liền nhau) mọi vế khớp trừ Error rate = `0`, vì cả 3 request rơi vào cùng một chu kỳ export 2 s nên series 503 không tăng trong cửa sổ và `rate()` = 0 (không phải lỗi code). Kịch bản được sửa (Sửa lần 1, Review lần 2 APPROVED).
  - **Chạy lại theo kịch bản đã duyệt** (bff mới `b17`, core mới, nguyên văn lệnh), mọi vế khớp:
    - `waitprom` → `prom ok after 0s`.
    - Theo route: `/healthz` GET `200` = 61, `/healthz` GET `503` = 13, `/readyz` GET `200` = 30, 404 GET không có nhãn `http_route` = 30. Không có `/khong-co-*`. `count` route = `3` (gồm nhóm rỗng của 404).
    - Trung bình duration `/healthz` = `0.0026` s. `count(...{job="bff"})` rỗng.
    - `--var`: `bff`, `snaptix/bff`, `snaptix/bff-tc09`, `snaptix/core`, `snaptix/core-tc11` (có `snaptix/bff` và `snaptix/core`; `bff` là series cũ từ lần kiểm chứng của reviewer, xem dưới).
    - `dsq snaptix/bff`: Request rate theo service `2.03`, theo route (`/healthz` `1.13`, `/readyz` `0.45`, 404 `0.45`), **Error rate (5xx) = `0.1105`**, Duration p50/p95/p99 = `0.0025`/`0.0048`/`0.0050` s. Không có `NO DATA`/`ERROR`.
  - `--var` còn giá trị `bff` (không namespace). Mẫu cuối của series đó lúc 01:33:48 UTC, **trước** lần chạy bff đầu tiên của người làm (01:43 UTC), nên là dữ liệu từ lần kiểm chứng của reviewer. Query `count(...{job="bff"})` hiện tại rỗng. Ngoài ra còn `snaptix/bff-tc09` (TC09 b) và `snaptix/core-tc11` (dữ liệu cũ).
- **TC18**: `pgrep -f 'tsx watch'` = 0 cả trước lẫn sau.
- **TC19**:
  - Trace `4a3ae3f615704948db4f3ff18b749d52`. Ảnh 1 (Explore): tiêu đề "snaptix/bff: GET /healthz", Services 2, 7 span dạng cây: `snaptix/bff GET /healthz` → `request` → `onRequest - healthRoutes`, `handler - healthRoutes` → `GET` → `snaptix/core GET /readyz` → `pool.acquire`.
  - Ảnh 2 (dashboard `snaptix-red`, service `snaptix/bff`, 15 phút): Request rate theo service/route, Error rate (5xx) (max 56.7%), Duration p50/p95/p99 đều có dữ liệu, không có "No data".
  - Ảnh lưu ở thư mục tạm của Claude in Chrome. Đã đóng tab test và tab trống do `tabs_context_mcp` tạo.
- **TC22**: grep `retry|circuit|…` = `0` (không có comment nào chứa các từ này).
- **Lệch với review**: góp ý nói A8 cần `OTEL_SEMCONV_STABILITY_OPT_IN=http`. Đọc code `@opentelemetry/instrumentation-http` 0.222.0 thì bản này chỉ còn semconv ổn định (không đọc biến opt-in), luôn ghi `http.server.request.duration` đơn vị giây. Vì vậy code không đặt biến. TC17 và unit test metric xác nhận.
