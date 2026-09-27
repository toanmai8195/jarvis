# Test cases — P0-T11: Skeleton `com/tm/app/apps/bff` Fastify + TS (ESM, strict, `tsx` khi dev, `tsup` khi build): plugin config, logger pino JSON, `/healthz`, `/readyz`

> **Phạm vi**: package `bff` đầu tiên của pnpm workspace. Gồm Fastify + TypeScript strict ESM, dev bằng `tsx watch`, build bằng `tsc --noEmit` + `tsup` ra `dist/` chạy được bằng `node`. Có plugin config đọc và validate env, logger pino JSON có `request_id`, `/healthz` (process sống) và `/readyz` (ping được MongoDB, có hạn thời gian). Unit test bằng Vitest (`fastify.inject`), lint bằng ESLint, typecheck, và chạy được trong `make test` và CI app. Căn cứ:
> - Phase 0 README: dòng P0-T11.
>   - **P0-FR3**: "`core` và `bff` có `/healthz` (sống) và `/readyz` (kết nối được DB)".
>   - **P0-NFR1**: "Log dạng JSON, có `trace_id`, `request_id`".
>   - **P0-FR4**: CI chạy lint, build, test cho TS.
>   - Task sau giữ ranh giới: **P0-T12** (OTel cho Node, gọi `core /healthz`, trace xuyên service, DoD `bff /healthz?deep=1`), **P0-T13** (graceful shutdown Fastify, `close` hooks), **P0-T14** (web), **P0-T15** (`packages/config`: eslint, prettier, tsconfig dùng chung), **P0-T17** (khung Vitest cho TS).
> - [project-structure](../../../../project-structure.md) mục `com/tm/app`:
>   - Dev BFF: `tsx watch`. Build BFF: `tsc --noEmit` + `tsup` (esbuild) → `dist/`. Test: Vitest.
>   - Image dùng `pnpm deploy --filter bff`, nên tên package là `bff`.
>   - Cây `apps/bff/src/`: `server.ts` ("khởi tạo Fastify, graceful shutdown"), `plugins/` (hạ tầng: mongo, redis, …), `core-client/`, `routes/`.
>   - "Plugin là đơn vị module; dependency gắn bằng `fastify.decorate`, không dùng DI container", không NestJS, "TypeScript strict, ESM, Node 22", không tách `controllers/`, `services/`, `repositories/`.
>   - Mục CI: `pnpm install --frozen-lockfile` → `pnpm --filter "...[base]" --filter '!snaptix-app' --if-present run '/^(lint|test|build)$/'` → `git diff --exit-code pnpm-lock.yaml`. Khi `pnpm-lock.yaml` đổi thì chạy mọi package.
> - [local-setup](../../../../local-setup.md):
>   - Cổng bff `3000`. `MONGODB_URI` (bff) `mongodb://localhost:27017/snaptix`.
>   - `pnpm --filter bff dev`, `pnpm --filter bff test`.
>   - `make test` suite app: `pnpm install --frozen-lockfile ; pnpm lint ; pnpm typecheck ; pnpm test ; pnpm build`.
>   - Core không tự đọc `.env` (`set -a; . .env; set +a`).
> - [architecture](../../../../architecture.md) / [services](../../../../services.md): BFF lưu dữ liệu mềm trong **MongoDB** (DB của BFF). Log JSON luôn kèm `request_id`. [api.md](../../../../api.md) mục "Quy ước chung": header `X-Request-ID` (giữ nếu khớp `^[A-Za-z0-9._:-]{1,128}$`, ngược lại sinh UUID v7, mọi response kể cả lỗi đều có header này, log mang cùng `request_id`).
> - Core (P0-T07, handbook [P0-T07](../../../../handbook/phase-0/P0-T07.md)) làm mẫu cho tính nhất quán:
>   - Config validate hết trước khi listen. Lỗi nêu tên biến, không lộ mật khẩu, exit `1`.
>   - Pool lười: khởi động được khi DB chưa lên, `/readyz` tự hồi phục.
>   - `/readyz` có hạn 2 s. Body `{"status":"ok"}` / `{"status":"unavailable"}`.
> - Handbook [P0-T01b](../../../../handbook/phase-0/P0-T01b.md) và [execute-all-notes](../../../execute-all-notes.md) mục P0-T01b/P0-T04: pnpm 11 tự `install` trước `run` nếu deps lệch, không liệt kê nhiều script sau `pnpm --filter` (dùng regex `run '/^(…)$/'`). P0-AT05 cần package TS có test thật.
>
> **Giả định** (reviewer xem kỹ):
> - A1. **Package và script**. `com/tm/app/apps/bff/package.json` có:
>   - `name` = `bff`, `private: true`, `type: "module"`, `engines.node` `>=22`.
>   - Script `dev` = `tsx watch src/server.ts`, `build` = `tsc --noEmit && tsup`, `start` = `node dist/server.js`, `test` = `vitest run` (không watch), `lint` = `eslint .` (có thể thêm `--max-warnings=0`), `typecheck` = `tsc --noEmit`.
>   - Entry `src/server.ts` đọc config, dựng app rồi listen. Hàm dựng app (ví dụ `buildApp(opts)` trong `src/app.ts`) nhận config và "pinger" Mongo để test bằng `inject` mà không cần Mongo thật (interface phía dùng). Plugin nằm trong `src/plugins/`, route trong `src/routes/`.
> - A2. **Env** (bff không tự đọc `.env`, giống core; `apps/bff/.env.example` liệt kê đủ biến):
>
>   | Biến | Mặc định | Quy tắc |
>   |---|---|---|
>   | `MONGODB_URI` | — | **Bắt buộc**, là URL `mongodb://` hoặc `mongodb+srv://` |
>   | `BFF_PORT` | `3000` | Số nguyên `1..65535` |
>   | `BFF_HOST` | `0.0.0.0` | — |
>   | `BFF_LOG_LEVEL` | `info` | `trace`, `debug`, `info`, `warn`, `error`, `fatal` |
>
>   Tiền tố `BFF_` theo cách core dùng `CORE_`. `MONGODB_URI` giữ tên đã có trong local-setup.
>
>   Config lỗi thì process in ít nhất một dòng log JSON `level` `error`/`fatal` nêu **tên** mọi biến sai (không in giá trị), rồi thoát `1` mà không listen.
>
>   Plugin config là Fastify plugin (bọc `fastify-plugin`) gắn config bằng `fastify.decorate('config', …)` và có khai báo type cho `FastifyInstance`. Thư viện validate tuỳ người làm (ví dụ `zod`, `@fastify/env`/`env-schema`), miễn có unit test.
>
>   Các biến `CORE_BASE_URL`, `REDIS_URL`, `SESSION_SECRET`, `AUTH_GOOGLE_*`, `ANALYTICS_DATABASE_URL` **chưa** được đọc, vì thuộc task sau.
> - A3. **`/readyz` = ping MongoDB**, theo P0-FR3 "kết nối được DB" và vì DB của BFF là MongoDB.
>   - Không gọi core: gọi core và trace BFF→core là P0-T12 (`/healthz?deep=1`). Không kiểm Redis: Redis dùng cho session/rate limit ở phase sau.
>   - Client Mongo (driver `mongodb`) tạo trong plugin `src/plugins/mongo.ts` (hoặc tên tương đương), là client **lười**: khởi động không chờ Mongo, không crash khi Mongo chết, kể cả sau 30 s (mặc định cũ của `serverSelectionTimeoutMS`).
>   - **Cách dùng driver `mongodb` 7.x đã kiểm chứng** (review lần 1):
>     - Đặt `serverSelectionTimeoutMS` ≤ hạn ping, ví dụ `new MongoClient(uri, { serverSelectionTimeoutMS: 2000 })`. Chỉ đặt `timeoutMS: 2000` là **không đủ**: lệnh đầu tiên tự connect và vẫn chờ chọn server 30 s.
>     - Mỗi lần ping gọi `await client.connect()` trước (idempotent khi đã nối), rồi `client.db().command({ ping: 1 }, { timeoutMS: 2000 })`. Lý do: sau lần chọn server thất bại đầu tiên (auto-connect), topology đóng hẳn, mọi lệnh sau trả `MongoTopologyClosedError` kể cả khi Mongo đã lên, không tự hồi phục. Cách tương đương cũng được nếu đã kiểm chứng qua TC15/TC16.
>     - Đã đo: Mongo dừng → ping lỗi sau ~2 s; start lại → `ok` sau ~1 s, không tạo lại client. `docker pause` → lỗi ~2 s (`MongoOperationTimeoutError`), unpause → hồi phục ngay.
>     - Lỗi từ `client.connect()` (kể cả lần đầu) phải được bắt trong handler `/readyz`, không để thành unhandled rejection.
>   - `/readyz` chạy `ping` với hạn **2 s** (như core). Kết quả: `200 {"status":"ok"}`, hoặc `503 {"status":"unavailable"}` kèm log `warn` có `request_id`, lỗi không chứa mật khẩu. `/readyz` tự hồi phục khi Mongo lên lại.
>   - `/healthz` trả `200 {"status":"ok"}`, không chạm Mongo.
>   - Plugin mongo đóng client trong hook `onClose` để `app.close()` trong test không treo. Đây là vòng đời plugin, không phải graceful shutdown.
> - A4. **Log**: logger của Fastify là pino, ghi ra stdout, mỗi dòng một object JSON, kể cả khi `dev` (không `pino-pretty` mặc định).
>   - `time` là chuỗi ISO 8601 (`pino.stdTimeFunctions.isoTime`), `level` là nhãn chữ thường (`formatters.level`), có `msg`. Chọn định dạng này để đọc thống nhất với core (slog `time`/`level`/`msg`).
>   - Mỗi request có dòng `incoming request` và `request completed` mặc định của Fastify, mang khoá `request_id`. Dòng `request completed` có mã status (`res.statusCode` hoặc `status`) và thời gian xử lý (`responseTime` hoặc `latency_ms`).
>   - Không log header (`authorization`, `cookie`, …): serializer mặc định của Fastify không log header, và người làm không thêm vào. `trace_id` thêm ở P0-T12.
>   - **Đổi nhãn `reqId` → `request_id` bằng `logController`**, không dùng option top-level `requestIdLogLabel`: ở fastify 5.12.x option này đã deprecated, tạo instance sẽ in `[FSTDEP024] FastifyDeprecation: requestIdLogLabel option is deprecated…` ra stderr (không phải JSON) → `jsonlines` báo `bad ≥ 1` (TC07–TC16). Dùng `import Fastify, { LogController } from 'fastify'` và `Fastify({ …, logController: new LogController({ requestIdLogLabel: 'request_id' }) })` (hoặc cách tương đương đúng API fastify 5.12 mà không phát cảnh báo). Reviewer đã đo: hết cảnh báo, khoá `request_id`, hai dòng `incoming request`/`request completed` giữ nguyên.
>   - **Quy tắc chung**: process (build `dist/server.js`) không được in dòng nào không phải JSON ra stdout **hoặc** stderr — kể cả cảnh báo deprecation/experimental của Fastify hay Node, `console.log`, stack trace thô. Lỗi khởi động (config, `EADDRINUSE`) cũng log qua pino dạng JSON.
> - A5. **Request ID** theo api.md (`^[A-Za-z0-9._:-]{1,128}$`, sai/thiếu → UUID v7):
>   - Đặt **`requestIdHeader: false`** và `genReqId(req)` **tự đọc** `req.headers['x-request-id']`: là chuỗi khớp regex thì giữ nguyên; thiếu, rỗng, không khớp (ký tự lạ, dài > 128) hoặc header lặp (mảng) thì sinh UUID v7 (thư viện tuỳ chọn, ví dụ `uuid` ≥ 10 `v7`).
>   - **Không** dùng `requestIdHeader: 'x-request-id'` + `genReqId`: khi header có mặt, Fastify lấy nguyên giá trị header và chỉ gọi `genReqId` khi thiếu header, nên id sai (`bad id!`) không bị thay (reviewer đã đo: trả lại `bad id!`, log `"request_id":"bad id!"` → TC12 fail).
>   - Fastify **không** tự trả header này. Phải tự gắn `X-Request-ID: <req.id>` vào response, ví dụ hook `onRequest` gọi `reply.header('x-request-id', req.id)` (hoặc `onSend`), đăng ký ở root (không bọc trong plugin encapsulated) để áp cho **mọi** response, kể cả 404 (not-found handler), 503 và lỗi 500.
> - A6. **404/405**: route lạ trả `404` (body JSON mặc định của Fastify cũng được). Sai method trên `/healthz` trả `404` hoặc `405`, không `2xx`/`5xx`. Format lỗi `{"error":{…}}` của api.md thuộc task có plugin `error-handler` (như ghi chú P0-T08 cho core).
> - A7. **Phiên bản** (npm registry ngày 2026-09-27):
>   - fastify `5.x` (5.12.5, kéo pino `^9.14 \|\| ^10.1`), mongodb `7.x` (7.6.0).
>   - **`fastify-plugin` `5.x`** (ví dụ `^5.1.0`), không dùng `6.0.0` (latest, đi cùng fastify 6 alpha).
>   - tsx `4.x`, tsup `8.5.x`, vitest `5.x` (cần Node `^22.12`, máy có 22.22.3), eslint `10.x` + `@eslint/js` + `typescript-eslint` `8.70.x`, `@types/node` `22.x`.
>   - **`typescript ~6.0.3`** (TypeScript 6.0.x), không dùng `7.0` (latest 7.0.2), vì `typescript-eslint` 8.70.1 có peer `typescript >=4.8.4 <6.1.0`. Reviewer đã đo: `~6.0.3` cài không cảnh báo peer, `tsc -v` in `Version 6.0.3`.
>   - Version được pin cụ thể (exact hoặc `^`/`~`), không có `latest`/`*`. Lockfile quyết định bản cài.
> - A8. **Cấu hình Vitest/ESLint/tsconfig tối thiểu, đặt riêng trong `apps/bff`**: `tsconfig.json` (strict, `module`/`moduleResolution` `NodeNext`), `eslint.config.js` (flat config: `@eslint/js` recommended + `typescript-eslint` recommended), `vitest.config.ts` nếu cần, `tsup.config.ts`. Bản dùng chung ở `packages/config` là P0-T15, khung Vitest chung là P0-T17. Không tạo package trong `packages/`.
> - A9. **pnpm 11 chặn build script**: nếu dependency (ví dụ `esbuild`) cần build script thì duyệt tường minh trong `pnpm-workspace.yaml` (`pnpm approve-builds`/`allowBuilds`), không để `pnpm ignored-builds` còn mục chưa xử lý.
>   - Đã xác nhận: pnpm 11.18.0 `install` **exit 1** với `ERR_PNPM_IGNORED_BUILDS` (esbuild của tsup/tsx), đồng thời **tự ghi** vào `pnpm-workspace.yaml` mục giữ chỗ `allowBuilds:` / `esbuild: set this to true or false`.
>   - Sửa **chính mục giữ chỗ đó** thành `allowBuilds: { esbuild: true }` (hoặc dạng block tương đương). **Không** thêm khoá `allowBuilds` thứ hai (YAML trùng khoá → lần chạy sau lỗi). Sau khi sửa: `grep -c 'set this to' com/tm/app/pnpm-workspace.yaml` = `0`, `grep -c '^allowBuilds' com/tm/app/pnpm-workspace.yaml` = `1`, `pnpm install --frozen-lockfile` exit `0`.
> - A10. **Không có graceful shutdown ở task này** (P0-T13): không bắt SIGTERM/SIGINT. Khi test, process dừng bằng tín hiệu mặc định của Node. Quy ước `shutdown_step` (ghi chú P0-T09) để P0-T13.
>
> **Không thuộc task này**: OTel Node, `traceparent`, `trace_id` trong log, gọi core, `?deep=1`, `core-client/` (P0-T12); bắt SIGTERM, drain, `close-with-grace` (P0-T13); `packages/config` dùng chung (P0-T15); khung Vitest chung, testcontainers cho TS (P0-T17); web (P0-T14); auth, session, Redis, rate limit, CSRF, route nghiệp vụ, `api/bff.openapi.yaml`, Dockerfile/image BFF, service bff trong compose, format lỗi `{"error":…}`.
>
> **Môi trường khi viết**: macOS arm64, Node 22.22.3, pnpm 11.18.0 (trong `com/tm/app`, theo `packageManager`), Docker Desktop + stack P0-T02 (`mongodb` `mongo:8.0.32`, không auth), curl, jq, `/usr/bin/python3` 3.9, `lsof`. Mongo **luôn** chạy qua `dc` (compose), không `docker run mongo:8.0.32` trực tiếp: image chạy trần thoát ngay (kernel 7.0.12, SERVER-121912), compose đã đặt `GLIBC_TUNABLES` để tránh lỗi này. Mọi TC dưới đây chỉ dùng `dc`.
>
> **Chuẩn bị**: lưu khối dưới thành `<scratchpad>/p0t11/prep.sh` một lần (tạo bằng Write tool, thay `<scratchpad>` bằng đường dẫn thật). Mỗi lần gọi Bash bắt đầu bằng `source <scratchpad>/p0t11/prep.sh`.
> - Khối chuẩn bị chạy được trên cả bash 3.2 lẫn zsh: biến luôn viết trong ngoặc `${…}` khi đứng sát `:`, không dùng biến tên `status`/`path`, không glob không khớp (zsh báo `no matches`).
> - Process nền chạy trong process group riêng (`bgrun`/`bgstop`), nên dừng được cả cây `pnpm → tsx → node`. `nap` dùng python (harness có thể chặn `sleep`). macOS không có `timeout`, nên dùng `runlimit`.
> - Phần chuẩn bị và các TC **không** có lệnh ghi lịch sử git (không `git commit`/`add`/`stash`).
> - File tạm `src/__tc_*` (TC04, TC05, TC18) luôn xoá bằng `rmtmp` ngay trong TC đó.
> - TC01–TC07, TC17–TC20 không cần stack Docker. Trước TC08: `dc up -d --wait mongodb` và `pa --filter bff build`.
> ```bash
> REPO=/Users/maiductoan/Documents/code/jarvis; cd "$REPO"
> APP=$REPO/com/tm/app; BFF=$APP/apps/bff
> S=<scratchpad>/p0t11; mkdir -p "$S"
> dc() { docker compose -f "$REPO/deploy/docker-compose.yml" "$@"; }
> pa() { (cd "$APP" && pnpm "$@"); }            # pnpm trong com/tm/app (pnpm 11.18.0 theo packageManager)
> P=13000; B=http://127.0.0.1:${P}               # cổng test, tránh 3000 của dev
> MURI='mongodb://localhost:27017/snaptix'
> MBAD='mongodb://tcuser:tc-pw-SECRET@127.0.0.1:27999/snaptix'   # không tới được, có mật khẩu giả
> UUID7='^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
> nap()   { python3 -c 'import time,sys;time.sleep(float(sys.argv[1]))' "$1"; }
> now()   { python3 -c 'import time;print("%.3f"%time.time())'; }
> since() { python3 -c "import time;print('%.2f'%(time.time()-$1))"; }
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
> # bff NAME [VAR=val...] — chạy dist/server.js nền với env SẠCH (chỉ PATH, HOME + biến truyền vào)
> bff()   { local n=$1; shift; bgrun "$n" env -i PATH="$PATH" HOME="$HOME" "$@" node "$BFF/dist/server.js"; }
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
> rid()   { curl -s -m 10 -D - -o /dev/null "$@" | tr -d '\r' | awk 'tolower($1)=="x-request-id:"{print $2}'; }
> waitcode() { local s=$(date +%s) c; while :; do c=$(curl -s -m 5 -o /dev/null -w '%{http_code}' "$1")
>              [ "$c" = "$2" ] && { echo "$c after $(( $(date +%s)-s ))s"; return 0; }
>              [ $(( $(date +%s)-s )) -ge "$3" ] && { echo "$c TIMEOUT"; return 1; }; nap 0.3; done; }
> listen() { lsof -nP -iTCP:"$1" -sTCP:LISTEN | awk 'NR>1{print $9}' | sort -u; }
> # jsonlines FILE [only] — mọi dòng không rỗng là JSON có time (ISO 8601), level (nhãn), msg;
> #   thêm đối số thứ 2 → chỉ xét dòng bắt đầu bằng `{` (bỏ banner của pnpm/tsx)
> jsonlines() { python3 -c 'import json,re,sys
> n=bad=0
> T=re.compile(r"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})$")
> for l in open(sys.argv[1]):
>     l=l.strip()
>     if not l: continue
>     if len(sys.argv)>2 and not l.startswith("{"): continue
>     n+=1
>     try:
>         o=json.loads(l); assert isinstance(o,dict) and {"time","level","msg"}<=o.keys()
>         assert o["level"] in ("trace","debug","info","warn","error","fatal")
>         assert isinstance(o["time"],str) and T.match(o["time"])
>     except Exception: bad+=1; print("BAD:",l[:200])
> print("lines=%d bad=%d"%(n,bad))' "$@"; }
> # jl FILE 'JQ' — áp jq lên các dòng JSON của log
> jl()    { jq -cR "fromjson? | select(type==\"object\") | $2" "$1"; }
> # rmtmp — xoá file tạm src/__tc_* của TC, in số file còn lại (phải là 0)
> rmtmp() { find "$BFF/src" -name '__tc_*' -delete 2>/dev/null; find "$BFF/src" -name '__tc_*' 2>/dev/null | wc -l | tr -d ' '; }
> ```

| ID | Loại | Kịch bản | Kết quả mong đợi | Trạng thái |
|---|---|---|---|---|
| P0-T11-TC01 | Cấu trúc | Package và cây thư mục (A1, A8): `find $BFF -path $BFF/node_modules -prune -o -path $BFF/dist -prune -o -type f -print \| sort`; `jq '{name,private,type,engines,scripts}' $BFF/package.json`; `find $BFF/src -type d \| sort`; `find $BFF/src -type d \| grep -ciE '/(controllers?\|services?\|repositor(y\|ies)\|models?\|utils?\|common\|helpers\|shared)$'`; `grep -rlE "@nestjs\|tsyringe\|inversify" $BFF/src $BFF/package.json \| wc -l`; `grep -rnE "decorate\(['\"]config['\"]" $BFF/src \| wc -l`; `grep -rn "fastify-plugin" $BFF/src \| wc -l`; `bash scripts/check-structure.sh; echo $?` | Có `package.json`, `tsconfig.json`, `tsup.config.ts`, `eslint.config.js` (flat), `.env.example`, `src/server.ts`, ít nhất một file trong `src/plugins/` (config, mongo) và `src/routes/` (health), cùng file `*.test.ts`. `name` = `bff`, `private` = `true`, `type` = `module`, `engines.node` bắt đầu bằng `>=22`. Script khớp A1: `dev` chứa `tsx watch src/server.ts`, `build` chứa `tsc --noEmit` và `tsup`, `start` = `node dist/server.js`, `test` chứa `vitest run`, `lint` chứa `eslint`, `typecheck` chứa `tsc --noEmit`. Không có thư mục kiểu Java (`0`), không NestJS/DI container (`0`). Có `decorate('config'` (≥ `1`) và plugin bọc `fastify-plugin` (≥ `1`). `check-structure.sh` exit `0` | ✅ |
| P0-T11-TC02 | Dependency | Dependency đúng công cụ và phiên bản (A7): `jq '{dependencies,devDependencies}' $BFF/package.json`; `jq -r '(.dependencies//{}) + (.devDependencies//{}) \| to_entries[] \| select(.value=="latest" or .value=="*" or .value=="") \| .key' $BFF/package.json \| wc -l`; `pa --filter bff list --depth 0`; `pa --filter bff exec tsc -v`; `pa --filter bff exec node -e "import('fastify').then(m=>console.log(m.default.version ?? 'ok'))"`; `jq -r '(.dependencies//{}) + (.devDependencies//{}) \| keys[]' $BFF/package.json \| grep -cE '^(@nestjs/\|@opentelemetry/\|undici$\|axios$\|ioredis$\|redis$\|@fastify/(session\|secure-session\|cookie\|csrf-protection\|rate-limit\|passport\|oauth2)$\|pino-pretty$)'` | `dependencies` có ít nhất `fastify` (5.x), `fastify-plugin`, `mongodb` (7.x). `devDependencies` có `typescript`, `tsx`, `tsup`, `vitest`, `eslint`, `@eslint/js`, `typescript-eslint`, `@types/node` (22.x). Không có spec `latest`/`*`/rỗng (`0`). `tsc -v` in `Version 6.0.x` (không phải 7.x, A7). `list` khớp `package.json`. Không có dependency của task sau (OTel, HTTP client gọi core, Redis, session/cookie/CSRF/rate-limit/auth), không NestJS, không `pino-pretty` (`0`) | ✅ |
| P0-T11-TC03 | Install | Lockfile và install sạch (A9): `shasum $APP/pnpm-lock.yaml > $S/lock.before`; `pa install --frozen-lockfile > $S/install.txt 2>&1; echo $?`; `grep -ciE 'unmet peer\|ERR_PNPM\|ignored build scripts\|WARN' $S/install.txt`; `pa ignored-builds`; `shasum -c $S/lock.before`; `grep -n "apps/bff:" $APP/pnpm-lock.yaml`; `git -C $REPO status --porcelain --untracked-files=all com/tm/app \| grep -E 'node_modules\|/dist/\|tsbuildinfo\|\.env$' \| wc -l` | Install exit `0`. Output không có cảnh báo peer, lỗi, build script bị bỏ qua hay `WARN` (`0`), nên peer `typescript <6.1.0` của typescript-eslint thoả. `pnpm ignored-builds` không liệt kê package nào chưa duyệt (nếu có duyệt thì nằm trong `pnpm-workspace.yaml`). Lockfile không đổi sau install (`OK`). Lockfile có importer `apps/bff`. `node_modules`, `dist`, `*.tsbuildinfo`, `.env` không xuất hiện trong `git status` (`0`) | ✅ |
| P0-T11-TC04 | Typecheck | tsconfig strict + ESM, bắt lỗi type (A8): `pa --filter bff exec tsc --showConfig \| jq '.compilerOptions \| {strict,module,moduleResolution,noEmit,target}'`; `pa --filter bff typecheck; echo $?`; `printf 'export const tcN: number = "x";\nexport function tcF(a) { return a; }\n' > $BFF/src/__tc_type.ts`; `pa --filter bff typecheck > $S/tc04.txt 2>&1; echo $?`; `grep -cE 'TS2322\|TS7006' $S/tc04.txt`; `pa --filter bff build > $S/tc04b.txt 2>&1; echo $?`; `rmtmp` | `strict` = `true`. `module` và `moduleResolution` = `nodenext` (không phân biệt hoa thường, ESM của Node). Typecheck sạch exit `0`. Có file tạm: typecheck exit ≠ `0`, output có cả `TS2322` (gán sai kiểu) và `TS7006` (implicit any), tức `≥ 2`. `build` cũng exit ≠ `0` (vì build chạy `tsc --noEmit` trước `tsup`). `rmtmp` in `0` | ✅ |
| P0-T11-TC05 | Lint | ESLint flat config bắt lỗi (A8): `pa --filter bff lint > $S/lint.txt 2>&1; echo $?`; `grep -ciE 'warning\|error' $S/lint.txt`; `printf 'export const tcAny: any = 1;\nconst tcUnused = 2;\n' > $BFF/src/__tc_lint.ts`; `pa --filter bff lint > $S/tc05.txt 2>&1; echo $?`; `grep -oE '@typescript-eslint/(no-explicit-any\|no-unused-vars)' $S/tc05.txt \| sort -u`; `rmtmp`; `pa --filter bff exec eslint --print-config src/server.ts \| jq '.rules \| has("@typescript-eslint/no-floating-promises") or has("@typescript-eslint/no-unused-vars")'` | Lint sạch exit `0`, không có warning/error (`0`). Có file tạm: exit ≠ `0` và báo đúng hai rule `@typescript-eslint/no-explicit-any`, `@typescript-eslint/no-unused-vars`. `rmtmp` in `0`. `--print-config` in `true` (rule typescript-eslint được áp cho `src/**/*.ts`; lint cũng phủ file test) | ✅ |
| P0-T11-TC06 | Unit | Vitest với `fastify.inject`, không cần Mongo/Docker (A3, A5): `(cd $APP && env -i PATH="$PATH" HOME="$HOME" pnpm --filter bff exec vitest run --reporter=verbose) > $S/vt.txt 2>&1; echo $?`; `grep -E 'Test Files\|Tests ' $S/vt.txt`; `for k in MONGODB_URI BFF_PORT BFF_LOG_LEVEL /healthz /readyz timeout X-Request-ID 404; do printf '%s=%s ' $k $(grep -c -- "$k" $S/vt.txt); done; echo`; `grep -rnE '\.(skip\|only\|todo)\(' $BFF/src \| wc -l`; `grep -rn '\.inject(' $BFF/src --include='*.test.ts' \| wc -l`; `pa --filter bff exec vitest list --filesOnly` | Exit `0` khi env không có `MONGODB_URI` và không cần stack. `Tests` ≥ `15` passed, `0` failed/skipped. Tên test (verbose) phủ đủ các nhóm, mỗi từ khoá ≥ `1`: config (thiếu/rỗng/sai scheme `MONGODB_URI`, `BFF_PORT` không phải số/ngoài `1..65535`, `BFF_LOG_LEVEL` lạ, giá trị mặc định, lỗi nêu tên biến mà không chứa giá trị/mật khẩu); `/healthz` 200 không gọi pinger; `/readyz` 200 khi pinger ok, 503 khi pinger lỗi, 503 khi pinger treo quá `timeout` (timeout ngắn truyền vào khi test, trả trong hạn); `X-Request-ID` giữ giá trị hợp lệ, sinh UUID v7 khi thiếu/sai/dài > 128; `404` route lạ có header. Không có `.skip/.only/.todo` (`0`). Có dùng `.inject(` (≥ `1`). `vitest list --filesOnly` chỉ liệt kê file trong `src/` (không có `dist/`) | ✅ |
| P0-T11-TC07 | Build | tsup ra `dist/` ESM chạy được bằng `node`: `rm -rf $BFF/dist; pa --filter bff build; echo $?`; `find $BFF/dist -type f \| sort`; `find $BFF/dist -name '*test*' \| wc -l`; `head -c 400 $BFF/dist/server.js; echo`; `node --check $BFF/dist/server.js; echo $?`; `git -C $REPO check-ignore -q com/tm/app/apps/bff/dist/server.js; echo $?`; `runlimit 10 $S/tc07.txt env -i PATH="$PATH" HOME="$HOME" node $BFF/dist/server.js`; `jsonlines $S/tc07.txt` | Build exit `0`, có `dist/server.js` (có thể thêm `.map`/chunk). `dist` không chứa file test (`0`). Đầu file là ESM (`import … from`), `node --check` exit `0`. `dist` bị git ignore (`0`). Chạy không có env (thiếu `MONGODB_URI`): `exit=1`, không phải `124` (không treo, không crash kiểu import lỗi `ERR_MODULE_NOT_FOUND`/`ERR_REQUIRE_ESM`). Output toàn JSON (`bad=0`) | ✅ |
| P0-T11-TC08 | E2E | Chạy bản build, `/healthz` và `/readyz` khi Mongo chạy (P0-FR3, A3): `dc up -d --wait mongodb`; `bff b8 MONGODB_URI=$MURI BFF_PORT=$P`; `waitcode $B/healthz 200 10`; `body -i $B/healthz \| tr -d '\r' \| grep -iE '^HTTP\|^content-type\|^\{'`; `waitcode $B/readyz 200 10`; `body $B/readyz`; `code -I $B/healthz`; `listen $P`; `jl $S/b8.log 'select(.msg\|test("listening";"i")) \| .msg'`; `alive b8`; `bgstop b8` | `/healthz` `200` trong ≤ 10 s, header `content-type: application/json…`, body `{"status":"ok"}`. `/readyz` `200`, body `{"status":"ok"}`. `HEAD /healthz` `200`. Listen `*:13000` (host mặc định `0.0.0.0`). Có dòng log `Server listening at …13000`. `alive` | ✅ |
| P0-T11-TC09 | E2E | Cổng/host mặc định và ghi đè (A2): `listen 3000 \| wc -l`; `bff b9 MONGODB_URI=$MURI`; `waitcode http://127.0.0.1:3000/healthz 200 10`; `listen 3000`; `bgstop b9`; `bff b9h MONGODB_URI=$MURI BFF_PORT=$P BFF_HOST=127.0.0.1`; `waitcode $B/healthz 200 10`; `listen $P`; `bgstop b9h`; `listen 3000 \| wc -l; listen $P \| wc -l` | Trước khi chạy, cổng 3000 trống (`0`; nếu bận thì ghi lại process và bỏ nửa đầu). Không đặt `BFF_PORT`: `/healthz` `200` trên `3000`, listen `*:3000`. `BFF_HOST=127.0.0.1`: listen đúng `127.0.0.1:13000` (không `*`). Sau khi dừng, hai cổng trống (`0`, `0`) | ✅ |
| P0-T11-TC10 | Config (negative) | Config sai thì thoát `1`, nêu tên biến, không lộ giá trị (A2): chạy lần lượt `runlimit 10 $S/c.txt env -i PATH="$PATH" HOME="$HOME" <ENV> node $BFF/dist/server.js` với `<ENV>` = (a) rỗng; (b) `MONGODB_URI=`; (c) `MONGODB_URI=http://tcuser:tc-pw-SECRET@x:1/db`; (d) `MONGODB_URI=$MURI BFF_PORT=abc`; (e) `MONGODB_URI=$MURI BFF_PORT=70000`; (f) `MONGODB_URI=$MURI BFF_PORT=0`; (g) `MONGODB_URI=$MURI BFF_LOG_LEVEL=verbose`; (h) `BFF_PORT=abc` (thiếu cả URI). Sau mỗi lần: `jsonlines $S/c.txt`; `jl $S/c.txt 'select(.level=="error" or .level=="fatal")' \| grep -oE 'MONGODB_URI\|BFF_PORT\|BFF_LOG_LEVEL' \| sort -u \| tr '\n' ' '`; `grep -c 'tc-pw-SECRET' $S/c.txt`; `listen $P \| wc -l`. Thêm (i) cổng bận: `bff b10 MONGODB_URI=$MURI BFF_PORT=$P; waitcode $B/healthz 200 10; runlimit 10 $S/c.txt env -i PATH="$PATH" HOME="$HOME" MONGODB_URI=$MURI BFF_PORT=$P node $BFF/dist/server.js; grep -c EADDRINUSE $S/c.txt; bgstop b10` | Cả (a)–(h): `exit=1`, `elapsed` < `5`s (không treo, không `124`). Output `bad=0` (toàn JSON). Dòng `error`/`fatal` nêu đúng biến sai: (a)(b)(c) `MONGODB_URI`, (d)(e)(f) `BFF_PORT`, (g) `BFF_LOG_LEVEL`, (h) **cả** `MONGODB_URI` và `BFF_PORT` (validate hết một lần). Không lộ mật khẩu (`tc-pw-SECRET` = `0`), không để lại cổng listen (`0`). (i) Process thứ hai `exit=1`, có log JSON `error`/`fatal` chứa `EADDRINUSE` (≥ `1`). Process đầu vẫn chạy tới `bgstop` | ✅ |
| P0-T11-TC11 | Log | Logger pino JSON, mức log (A4, P0-NFR1): `bff b11 MONGODB_URI=$MURI BFF_PORT=$P`; `waitcode $B/readyz 200 10`; `code -H 'X-Request-ID: tc11-a' $B/healthz`; `nap 0.5`; `jsonlines $S/b11.log`; `jl $S/b11.log 'select(.request_id=="tc11-a") \| {level,msg,st:(.res.statusCode // .status),rt:(.responseTime // .latency_ms)}'`; `jl $S/b11.log 'select(has("reqId"))' \| wc -l`; `bgstop b11`; `bff b11w MONGODB_URI=$MURI BFF_PORT=$P BFF_LOG_LEVEL=warn`; `waitcode $B/healthz 200 10`; `code $B/healthz`; `nap 0.5`; `jl $S/b11w.log 'select(.level=="info" or .level=="debug")' \| wc -l`; `bgstop b11w`; `bff b11d MONGODB_URI=$MURI BFF_PORT=$P BFF_LOG_LEVEL=debug`; `waitcode $B/healthz 200 10`; `bgstop b11d` | Mọi dòng log là JSON có `time` ISO 8601, `level` là nhãn, có `msg` (`bad=0`). Request `tc11-a` có đúng 2 dòng `info`: `incoming request` và `request completed`. Dòng `completed` có `st` = `200` và `rt` là số ≥ `0`. Không còn khoá `reqId` mặc định (`0`), khoá là `request_id`. `BFF_LOG_LEVEL=warn`: không có dòng `info`/`debug` (`0`), vẫn phục vụ `200`. `debug` được chấp nhận (healthz `200`) | ✅ |
| P0-T11-TC12 | Request ID | `X-Request-ID` theo api.md (A5): `bff b12 MONGODB_URI=$MURI BFF_PORT=$P`; `waitcode $B/healthz 200 10`; `rid -H 'X-Request-ID: tc12.ok_A:1-z' $B/healthz`; `rid -H 'X-Request-ID: bad id!' $B/healthz \| grep -cE "$UUID7"`; `L=$(python3 -c 'print("a"*129)'); rid -H "X-Request-ID: ${L}" $B/healthz \| grep -cE "$UUID7"`; `A=$(rid $B/healthz); Z=$(rid $B/healthz); echo "$A $Z"; [ "$A" != "$Z" ] && echo distinct`; `echo "$A" \| grep -cE "$UUID7"`; `rid -H 'X-Request-ID: tc12-404' $B/khong-co`; `nap 0.5`; `jl $S/b12.log "select(.request_id==\"$A\") \| .msg"`; `jl $S/b12.log 'select(.request_id=="tc12-404") \| (.res.statusCode // .status)'`; `bgstop b12` | Giá trị hợp lệ `tc12.ok_A:1-z` được trả lại nguyên văn. `bad id!` và chuỗi 129 ký tự bị thay bằng UUID v7 (`1`, `1`), không trả `400`. Không gửi header: hai request có hai id khác nhau (`distinct`), đều là UUID v7 (`1`). Route 404 vẫn trả `tc12-404`. Log của request `$A` có `incoming request` và `request completed` cùng `request_id`. Log `tc12-404` có status `404` | ✅ |
| P0-T11-TC13 | Bảo mật | Không log header nhạy cảm và mật khẩu Mongo (A2, A4): `bff b13 MONGODB_URI=$MBAD BFF_PORT=$P BFF_LOG_LEVEL=debug`; `waitcode $B/healthz 200 10`; `code -H 'Authorization: Bearer tc13-tok-SECRET' -H 'Cookie: sid=tc13-cookie-SECRET' -H 'X-Service-Token: tc13-svc-SECRET' $B/healthz`; `code -H 'Authorization: Bearer tc13-tok-SECRET' $B/readyz`; `nap 1`; `grep -cE 'tc13-\|tc-pw-SECRET' $S/b13.log`; `grep -ciE '"(headers\|authorization\|cookie)"' $S/b13.log`; `grep -c 'tcuser' $S/b13.log`; `bgstop b13` | Ở mức `debug`, log không chứa token/cookie/service token hay mật khẩu trong `MONGODB_URI` (`0`), cũng không có object header nào (`0`). `/readyz` với URI không tới được trả `503` (TC15), và lỗi driver trong log không chứa mật khẩu. Username có thể xuất hiện hoặc không, chỉ ghi nhận | ✅ |
| P0-T11-TC14 | HTTP | 404/405 (A6): `bff b14 MONGODB_URI=$MURI BFF_PORT=$P`; `waitcode $B/healthz 200 10`; `code $B/khong-co`; `code $B/`; `body $B/khong-co \| jq -e type`; `code -X POST $B/healthz`; `code -X DELETE $B/readyz`; `code "$B/healthz?x=1"`; `code $B/healthz/`; `nap 0.5; jl $S/b14.log 'select(.level=="error" or .level=="fatal")' \| wc -l`; `bgstop b14` | `/khong-co` và `/` trả `404` với body JSON (`jq` in `"object"`). `POST /healthz` và `DELETE /readyz` trả `404` hoặc `405`, không `2xx`/`5xx`. `/healthz?x=1` `200`. `/healthz/` trả `404` hoặc `200` (ghi nhận hành vi). Không có log `error`/`fatal` (`0`) | ✅ |
| P0-T11-TC15 | E2E (negative) | Mongo không tới được: khởi động lười, `/readyz` có hạn, không crash (A3): `bff b15 MONGODB_URI=$MBAD BFF_PORT=$P`; `waitcode $B/healthz 200 10`; `for i in 1 2 3; do ctime $B/readyz; done`; `body $B/readyz`; `ctime $B/healthz`; `nap 35`; `alive b15`; `code $B/healthz; ctime $B/readyz`; `jsonlines $S/b15.log`; `jl $S/b15.log 'select(.level=="warn") \| {msg,request_id}' \| head -3`; `jl $S/b15.log 'select(.level=="error" or .level=="fatal")' \| wc -l`; `grep -ciE 'unhandled\|uncaught' $S/b15.log`; `bgstop b15` | BFF listen và `/healthz` `200` trong ≤ 10 s dù Mongo không tới được. Mỗi `/readyz` trả `503` với `time_total` ≤ `3`s (hạn 2 s, không treo tới 30 s của driver). Body `{"status":"unavailable"}`. `/healthz` `200` trong < `1`s. Sau 35 s (quá `serverSelectionTimeoutMS` mặc định) process vẫn `alive`, `/healthz` `200`, `/readyz` `503` ≤ `3`s. Log `bad=0`. Có dòng `warn` cho readyz lỗi, kèm `request_id`. Không có `error`/`fatal`, không có unhandled rejection (`0`, `0`) | ✅ |
| P0-T11-TC16 | E2E | Mongo treo/dừng rồi hồi phục, không cần restart BFF (A3): `dc up -d --wait mongodb`; `bff b16 MONGODB_URI=$MURI BFF_PORT=$P`; `waitcode $B/readyz 200 15`; `lsof -nP -iTCP:$P -sTCP:LISTEN -t > $S/pid16`; (a) `dc pause mongodb`; `ctime $B/readyz; ctime $B/readyz`; `ctime $B/healthz`; `dc unpause mongodb`; `waitcode $B/readyz 200 30`. (b) `dc stop mongodb`; `ctime $B/readyz`; `ctime $B/healthz`; `dc start mongodb`; `waitcode $B/readyz 200 60`. (c) `lsof -nP -iTCP:$P -sTCP:LISTEN -t \| diff - $S/pid16 && echo same-pid`; `alive b16`; `jsonlines $S/b16.log`; `bgstop b16`. (d) BFF khởi động khi Mongo đang dừng: `dc stop mongodb`; `bff b16d MONGODB_URI=$MURI BFF_PORT=$P`; `waitcode $B/healthz 200 10`; `ctime $B/readyz`; `dc start mongodb`; `waitcode $B/readyz 200 60`; `bgstop b16d`; `dc up -d --wait mongodb` | Ban đầu `200`. (a) Khi Mongo bị pause (treo): `/readyz` `503` với `time_total` ≤ `3`s, `/healthz` `200` < `1`s. Sau unpause, `/readyz` `200` trong ≤ 30 s. (b) Khi Mongo dừng: `503` ≤ `3`s, `/healthz` `200`. Sau start, `200` trong ≤ 60 s. (c) Cùng PID (`same-pid`), `alive`, log `bad=0`. (d) BFF khởi động được khi Mongo dừng (`/healthz` `200`, `/readyz` `503`), rồi tự lên `200` khi Mongo chạy, không restart. `mongodb` healthy lại ở cuối | ✅ |
| P0-T11-TC17 | Dev | `tsx watch` khi dev (A1, A4): `bgrun dev env -i PATH="$PATH" HOME="$HOME" MONGODB_URI=$MURI BFF_PORT=$P pnpm --dir $APP --filter bff dev`; `waitcode $B/healthz 200 30`; `jsonlines $S/dev.log only`; `grep -c 'tsx' $S/dev.log`; `N1=$(grep -ciE 'listening' $S/dev.log)`; `touch $BFF/src/server.ts`; `nap 5`; `waitcode $B/healthz 200 20`; `N2=$(grep -ciE 'listening' $S/dev.log); echo "$N1 -> $N2"`; `bgstop dev`; `nap 1; listen $P \| wc -l`; `pgrep -f 'tsx watch src/server.ts' \| wc -l` | Dev lên, `/healthz` `200` trong ≤ 30 s. Log ứng dụng (dòng bắt đầu bằng `{`) là JSON (`bad=0`, không `pino-pretty`). Script dev dùng `tsx` (dòng lệnh của pnpm in `tsx watch`, ≥ `1`). Sau `touch`, process tự chạy lại: `N2` > `N1`, `/healthz` `200` lại. Sau `bgstop`, không còn process giữ cổng 13000 và không còn `tsx watch` (`0`, `0`), tức dừng được cả cây pnpm → tsx → node. `touch` chỉ đổi mtime, không đổi nội dung file | ✅ |
| P0-T11-TC18 | CI | Lệnh app của CI chạy đủ lint/test/build, và bắt được test fail (P0-FR4, P0-AT05 cục bộ): `shasum $APP/pnpm-lock.yaml > $S/lock.before`; `pa --filter bff run '/^(lint\|test\|build)$/' > $S/tc18a.txt 2>&1; echo $?`; `grep -E 'eslint\|Test Files\|Build success' $S/tc18a.txt`; `pa --recursive --filter '!snaptix-app' --if-present run '/^(lint\|test\|build)$/' > $S/tc18b.txt 2>&1; echo $?`; `grep -E 'eslint\|Test Files\|Build success' $S/tc18b.txt`; `shasum -c $S/lock.before`; `printf 'import { expect, test } from "vitest";\ntest("tc18 fail", () => { expect(1).toBe(2); });\n' > $BFF/src/__tc_fail.test.ts`; `pa --filter bff test > $S/tc18c.txt 2>&1; echo $?`; `pa --recursive --filter '!snaptix-app' --if-present run '/^(lint\|test\|build)$/' > $S/tc18d.txt 2>&1; echo $?`; `grep -c 'tc18 fail' $S/tc18c.txt`; `rmtmp`; `bash scripts/ci-app_test.sh; echo $?`; `bash scripts/ci-changes_test.sh; echo $?` | Lệnh một app exit `0` và chạy đủ ba script, không phải chỉ `lint`: output có dấu vết của cả ba (`eslint`, `Test Files` của Vitest, `Build success` của tsup). Lệnh dạng CI khi chạy mọi package exit `0`, cũng có đủ ba dấu vết của bff. Commit của task đổi `pnpm-lock.yaml` nên CI sẽ đi nhánh "chạy mọi package" của `ci-app.sh`. Lockfile không bị script viết lại (`OK`). Có test fail tạm: cả hai lệnh exit ≠ `0`, output nêu `tc18 fail` (≥ `1`). `rmtmp` in `0`. `ci-app_test.sh`, `ci-changes_test.sh` exit `0`. Nhánh `...[base]` trên commit thật kiểm khi đóng phase (P0-AT05) | ✅ |
| P0-T11-TC19 | Hồi quy | `make test` suite app chạy thật: `bash scripts/test-all.sh app > $S/tc19.txt 2>&1; echo $?`; `grep -E 'Tests \|Test Files\|FAIL\|bước lỗi' $S/tc19.txt`; `grep -c 'bff' $S/tc19.txt`; `bash scripts/test-all.sh scripts; echo $?`; `git -C $REPO status --porcelain --untracked-files=all \| grep -E 'dist/\|node_modules\|tsbuildinfo\|\.log$' \| wc -l` | `test-all.sh app` exit `0`. Output có kết quả Vitest của bff (`Tests` passed, không `FAIL`), tức install frozen, lint, typecheck, test, build đều chạy trên package thật (bff xuất hiện, ≥ `1`). `test-all.sh scripts` exit `0` (gồm `check-structure.sh`). Không sinh file rác ngoài ignore (`0`) | ✅ |
| P0-T11-TC20 | Tài liệu | Docs khớp code: `grep -nE 'BFF_PORT\|BFF_HOST\|BFF_LOG_LEVEL\|MONGODB_URI' com/tm/docs/technical/local-setup.md`; `grep -nE 'apps/bff/\.env\|pnpm --filter bff (dev\|build\|start)\|3000/(healthz\|readyz)' com/tm/docs/technical/local-setup.md`; `grep -cE '^(MONGODB_URI\|BFF_PORT\|BFF_HOST\|BFF_LOG_LEVEL)=' $BFF/.env.example`; `git -C $REPO check-ignore -q com/tm/app/apps/bff/.env.example; echo $?`; `grep -nE 'server.ts\|plugins/\|routes/' com/tm/docs/technical/project-structure.md \| head`; `ls com/tm/docs/technical/handbook/phase-0/P0-T11.md && grep -c 'P0-T11' com/tm/docs/technical/handbook/README.md` | Bảng biến môi trường của `local-setup.md` có `BFF_PORT`, `BFF_HOST`, `BFF_LOG_LEVEL` (kèm mặc định, quy tắc) và dòng `MONGODB_URI` ghi rõ bắt buộc với bff, lỗi thì thoát. Có hướng dẫn chạy bff (sao chép `apps/bff/.env.example`, nạp env, `pnpm --filter bff dev`, `/healthz` và `/readyz` ở cổng 3000, `/readyz` ping Mongo). `.env.example` có đủ 4 biến (`4`) và không bị ignore (`1`). Cây BFF trong `project-structure.md` khớp file thật (nếu khác thì đã sửa). Handbook `P0-T11.md` tồn tại, `handbook/README.md` có dòng P0-T11 ở cả hai bảng (≥ `2`) | ✅ |
| P0-T11-TC21 | Phạm vi | Không làm việc của task khác (A3, A8, A10): `git -C $REPO status --porcelain --untracked-files=all`; `git -C $REPO status --porcelain com/tm/server deploy scripts .github Makefile \| wc -l`; `ls -A $APP/packages $APP/apps`; `grep -rnE "process\.on\(['\"](SIGTERM\|SIGINT)\|close-with-grace" $BFF/src \| wc -l`; `grep -rnE "@opentelemetry\|traceparent\|CORE_BASE_URL\|REDIS_URL\|SESSION_SECRET\|fetch\(\|undici\|deep" $BFF/src \| wc -l`; `ls $APP/api` | Thay đổi chỉ nằm trong: `com/tm/app/apps/bff/**`, `com/tm/app/pnpm-lock.yaml`, `com/tm/app/pnpm-workspace.yaml` (chỉ khi duyệt build script, A9), `com/tm/app/apps/.gitkeep` (có thể xoá), `com/tm/docs/technical/{local-setup.md,project-structure.md,architecture.md}`, handbook (`phase-0/P0-T11.md`, `README.md`), planning (`planning/README.md`, `phase-0-foundation/README.md`, `tasks/P0-T11/**`, `acceptance-tests.md`), `planning/execute-all-notes.md`. `com/tm/server`, `deploy`, `scripts`, `.github`, `Makefile` không đổi (`0`). `packages/` chỉ có `.gitkeep` (không có `config`, là P0-T15). `apps/` chỉ có `bff` (và `.gitkeep`). Không bắt tín hiệu (`0`, P0-T13). Không OTel, không gọi core/Redis/session, không `deep` (`0`, P0-T12). `api/` không có `bff.openapi.yaml` mới | ✅ |
| P0-T11-TC22 | Dọn dẹp | `for n in b8 b9 b9h b10 b11 b11w b11d b12 b13 b14 b15 b16 b16d dev; do bgstop $n; done`; `listen $P \| wc -l; listen 3000 \| wc -l`; `pgrep -f "$BFF/dist/server.js" \| wc -l`; `pgrep -f 'tsx watch src/server.ts' \| wc -l`; `rmtmp`; `docker inspect -f '{{.State.Status}} {{.State.Health.Status}}' $(dc ps -aq mongodb)`; `dc down`; `docker ps -q --filter label=com.docker.compose.project=snaptix \| wc -l \| tr -d ' '`; `rm -rf $S; ls -d $S 2>/dev/null \| wc -l \| tr -d ' '` | Không còn process bff/tsx (`0`, `0`), cổng 13000 và 3000 trống (`0`, `0`). Không còn file tạm `src/__tc_*` (`0`). Trước `down`, `mongodb` là `running healthy` (không bị bỏ ở trạng thái pause/stop). Sau `dc down` không còn container của project (`0`), volume giữ nguyên (không `-v`). Scratchpad đã xoá (`0`) | ✅ |

Test nghiệm thu liên quan:
- **P0-FR3** (phần `bff`): TC08, TC15, TC16. Acceptance-tests chưa có dòng riêng cho `bff /readyz` (P0-AT02/AT03 chỉ nói core).
- **P0-AT05** (PR có test TS fail → CI fail ở bước test): task này tạo package TS có test thật đầu tiên (ghi chú P0-T04). TC18 kiểm cục bộ rằng lệnh CI exit ≠ 0 khi có test fail. CI thật trên GitHub chạy khi đóng phase (nhánh `ci-check/*`), nên để ⬜.
- **P0-AT11** (PR chỉ sửa `com/tm/app/**` → không chạy job Bazel): commit của task này chỉ đổi `com/tm/app/**` và docs (TC21). Kiểm trên CI thật khi đóng phase, để ⬜.
- **P0-AT07** / **P0-NFR1** (`trace_id`) / **P0-NFR2**: cần OTel Node và gọi core (P0-T12). Task này chỉ lo `request_id` trong log bff (TC11, TC12), để ⬜.

## Review

### Review lần 1 — CHANGES_REQUESTED

Reviewer độc lập (execute-all). Bảng test đúng định dạng: tiêu đề đúng, 5 cột, ID `P0-T11-TC01..TC22` liên tục, trạng thái ⬜, có dòng "Test nghiệm thu liên quan". Phạm vi bám dòng P0-T11 và P0-FR3, ranh giới P0-T12/T13/T15/T17 rõ (TC21). Không có lệnh ghi lịch sử git, không liên quan tới tiền. Các kỳ vọng của TC đều đúng. Hai giả định **hướng dẫn cách làm sai**: người làm theo đúng giả định thì TC fail. Chỉ cần sửa phần giả định, không phải sửa kỳ vọng của TC.

Kiểm chứng trong scratchpad, đã dọn: workspace pnpm 11.18.0 tạm, Node 22.22.3, fastify 5.12.5, mongodb 7.6.0, container `mongo:8.0.32` riêng ở cổng 27888.

**Phải sửa:**

1. **A5 (ảnh hưởng TC12, TC06)**. Cặp `requestIdHeader: 'x-request-id'` + `genReqId` kiểm regex **không** thay được id sai.
   - Khi header có mặt, Fastify dùng nguyên giá trị header. `genReqId` chỉ được gọi khi thiếu header.
   - Đã đo: gửi `X-Request-ID: bad id!` thì nhận lại `bad id!` nguyên văn, còn log ghi `"request_id":"bad id!"`. TC12 sẽ fail (`bad id!` → UUID v7).
   - Sửa A5 thành: `requestIdHeader: false`, và `genReqId(req)` tự đọc `req.headers['x-request-id']`, kiểm `^[A-Za-z0-9._:-]{1,128}$`, sai hoặc thiếu thì sinh `v7()`. Đã đo cách này: `bad id!` → UUID v7, `good.1` giữ nguyên, 404 vẫn có header.
   - Nêu thêm: header response `X-Request-ID` phải tự gắn (ví dụ hook `onRequest` gọi `reply.header`), vì Fastify không tự trả header này.
2. **A4 (ảnh hưởng TC08–TC16, cụ thể `bad=0` của `jsonlines`)**. Ở fastify 5.12.5, option top-level `requestIdLogLabel` đã **deprecated**.
   - Khi tạo instance, Fastify in cảnh báo `[FSTDEP024] FastifyDeprecation: requestIdLogLabel option is deprecated…` ra stderr. Cảnh báo này không phải JSON.
   - Hàm `bff()`/`bgrun` gộp stderr vào `$S/bXX.log`, nên `jsonlines` báo `bad ≥ 1`.
   - Sửa A4 thành `logController: new LogController({ requestIdLogLabel: 'request_id' })` (`import Fastify, { LogController } from 'fastify'`). Đã đo: không còn cảnh báo, log có khoá `request_id`, dòng `incoming request`/`request completed` giữ nguyên.
   - Nên ghi thêm quy tắc chung: process không được in cảnh báo không phải JSON (deprecation, experimental) ra stdout/stderr.

**Góp ý, không chặn** (nên đưa vào giả định để người làm đỡ mất công):

3. **A3, driver `mongodb` 7.6.0**. Đã đo với URI không tới được (`MBAD`):
   - Lệnh đầu tiên tự connect và chờ `serverSelectionTimeoutMS` 30 s, bỏ qua `timeoutMS: 2000`. Kết quả: `ERR 30005ms MongoServerSelectionError`.
   - Sau lần đó topology **đóng vĩnh viễn**: mọi `ping` sau đó trả `MongoTopologyClosedError`, kể cả khi Mongo đã lên. Như vậy vi phạm TC15 (≤ 3 s) và TC16(d) (tự hồi phục).
   - Cách đã kiểm là chạy được: `new MongoClient(uri, { serverSelectionTimeoutMS: 2000 })`, mỗi lần ping gọi `await client.connect()` rồi `db().command({ ping: 1 }, { timeoutMS: 2000 })`. Mongo dừng thì ping lỗi sau ~2 s. Mongo start lại thì `ok` sau ~1 s, không cần tạo lại client.
   - Với `docker pause`, ping lỗi sau ~2 s (`MongoOperationTimeoutError`). Unpause thì hồi phục ngay.
   - TC15/TC16 bắt được lỗi này, nên chỉ cần thêm gợi ý vào A3. Câu "kể cả sau `serverSelectionTimeoutMS` mặc định 30 s" nên đổi thành "đặt `serverSelectionTimeoutMS` ≤ hạn ping".
4. **A9, đã xác nhận**. pnpm 11.18.0 `install` **exit 1** với `ERR_PNPM_IGNORED_BUILDS` (esbuild của tsup/tsx).
   - pnpm còn **tự ghi** vào `pnpm-workspace.yaml` một mục giữ chỗ `allowBuilds: esbuild: set this to true or false`. Mục này làm lần chạy sau lỗi YAML nếu thêm tay một khoá `allowBuilds` thứ hai.
   - Có `allowBuilds: { esbuild: true }` thì install exit 0, không có `WARN`/peer, `--frozen-lockfile` exit 0.
   - Nên ghi trong A9: sửa chính mục giữ chỗ đó, và kiểm file không còn chuỗi "set this to".
5. **A7, đã xác nhận đúng**:
   - `typescript` latest là 7.0.2. `typescript-eslint@8.70.1` có peer `typescript >=4.8.4 <6.1.0`. Với `typescript ~6.0.3` install không có cảnh báo peer, `tsc -v` in `Version 6.0.3`.
   - eslint 10.11.0 / `@eslint/js` 10.0.1 / vitest 5.0.2 (engines `^22.12.0`) / tsup 8.5.1 / tsx 4.23.15 cài sạch.
   - Lưu ý: `fastify-plugin` latest là **6.0.0** (2026-06, cùng thời với fastify 6 alpha). Nên ghi rõ dùng `fastify-plugin` `5.x` (đã thử 5.1.0) cho fastify 5.
6. **Đã kiểm thực tế, khớp kỳ vọng**:
   - `tsc --showConfig` in `nodenext`.
   - ESLint flat báo đúng `no-explicit-any` và `no-unused-vars`. `--print-config` in `true`.
   - Vitest 5 hỗ trợ `--reporter=verbose` (in tên test) và `list --filesOnly`.
   - tsup in `Build success`. `dist/server.js` là ESM.
   - `tsx watch` chạy lại khi `touch` (có dòng `[tsx] change in ./src/server.ts Restarting...`, `listening` 1 → 2). `killpg` dừng được cả cây.
   - `pino` với `timestamp` ISO + `formatters.level` cho `level` dạng nhãn chạy được.
   - `HEAD /healthz` `200`, `POST /healthz` `404`.
7. **Môi trường**: `mongo:8.0.32` chạy trần thì thoát ngay (kernel 7.0.12, SERVER-121912). Compose đã có `GLIBC_TUNABLES`, nên TC dùng `dc` là đúng, không dùng `docker run` riêng.
8. **TC06**: việc bắt tên test phải chứa từ khoá (`timeout`, `404`, …) hơi áp đặt nhưng làm được. Giữ nguyên.

### Sửa lần 1

Chỉ sửa phần giả định/môi trường; bảng TC01..TC22 (kỳ vọng, lệnh, ⬜) giữ nguyên vì kỳ vọng đã đúng.

1. **A5 (chặn, TC06/TC12)**: đổi cách làm thành `requestIdHeader: false` + `genReqId(req)` tự đọc `x-request-id`, kiểm regex api.md, hợp lệ giữ / thiếu, rỗng, sai, dài > 128, header lặp → UUID v7. Ghi lý do không dùng `requestIdHeader: 'x-request-id'` (Fastify lấy nguyên header, chỉ gọi `genReqId` khi thiếu). Ghi rõ phải tự gắn header `X-Request-ID` vào response (hook `onRequest`/`onSend` ở root) cho mọi response kể cả 404/503/500.
2. **A4 (chặn, `bad=0` ở TC07–TC16)**: bỏ `requestIdLogLabel` top-level (deprecated FSTDEP024 ở fastify 5.12.x, in cảnh báo non-JSON ra stderr), dùng `logController: new LogController({ requestIdLogLabel: 'request_id' })`. Thêm quy tắc chung: không dòng non-JSON nào trên stdout/stderr (deprecation, experimental, `console.log`, stack thô; lỗi khởi động cũng qua pino JSON).
3. **A3 (góp ý)**: thêm mục "cách dùng driver `mongodb` 7.x đã kiểm chứng" — `serverSelectionTimeoutMS: 2000`; `await client.connect()` trước mỗi ping rồi `command({ ping: 1 }, { timeoutMS: 2000 })`; lý do `MongoTopologyClosedError` sau lần chọn server lỗi đầu; `timeoutMS` một mình không đủ; bắt lỗi `connect()` trong handler. Câu "kể cả sau `serverSelectionTimeoutMS` mặc định 30 s" đổi thành "kể cả sau 30 s".
4. **A9 (góp ý)**: ghi pnpm 11.18 exit 1 `ERR_PNPM_IGNORED_BUILDS` và tự ghi mục giữ chỗ `esbuild: set this to true or false`; sửa chính mục đó thành `allowBuilds: { esbuild: true }`, không thêm khoá `allowBuilds` thứ hai; cách kiểm (`set this to` = 0, một khoá `allowBuilds`).
5. **A7 (góp ý)**: ghi rõ `typescript ~6.0.3` và `fastify-plugin` `5.x` (không 6.0.0, đi với fastify 6 alpha); mongodb 7.6.0.
6. **Môi trường**: ghi Mongo luôn chạy qua `dc` (compose có `GLIBC_TUNABLES`), không `docker run mongo:8.0.32` trực tiếp (SERVER-121912). Đã rà: không TC nào dùng `docker run mongo`, nên không phải đổi lệnh TC.

### Review lần 2 — APPROVED

Reviewer độc lập (execute-all), lần 2. Bảng TC01..TC22 không đổi so với lần 1: 5 cột, ID `P0-T11-TC01..TC22` liên tục, trạng thái ⬜, còn dòng "Test nghiệm thu liên quan". Không có lệnh ghi lịch sử git. Cả hai điểm chặn và ba góp ý của Review lần 1 đã đưa vào giả định A3, A4, A5, A7, A9 đúng nội dung.

Kiểm chứng lại trong scratchpad theo **đúng** hướng dẫn A3/A4/A5 mới: fastify 5.12.5 (pino `^9.14 || ^10.1`), mongodb 7.6.0, uuid `v7`, Node 22.22.3, Mongo chạy qua `dc` (`down` không `-v` ở cuối, không còn process, scratchpad đã xoá).

- **(a) A5**: dùng `requestIdHeader: false` và `genReqId` tự đọc `x-request-id`, kiểm regex api.md, hook `onRequest` ở root gọi `reply.header('x-request-id', req.id)`. Kết quả:
  - `tc12.ok_A:1-z` được giữ nguyên.
  - `bad id!`, chuỗi 129 ký tự và request không có header đều nhận UUID v7, và các id khác nhau.
  - Route lạ trả `404` và vẫn trả lại `tc12-404` trong header.
- **(b) A4**: `LogController` được export có tên từ `'fastify'` (`import Fastify, { LogController } from 'fastify'`, `typeof` = `function`). Với `logController: new LogController({ requestIdLogLabel: 'request_id' })`:
  - Không có `FSTDEP` nào (`0`).
  - Log có 37 dòng, dòng nào cũng là JSON (`bad=0`).
  - Khoá log là `request_id`, không có `reqId`.
  - `time` theo ISO, `level` là nhãn chữ.
- **(c) A3**: dùng `serverSelectionTimeoutMS: 2000`, gọi `connect()` trước mỗi `ping`, `timeoutMS: 2000`:
  - Khi Mongo bị `pause`: `/readyz` trả `503` sau 2.01 s và 2.01 s. `/healthz` trả `200` sau 0.002 s. Sau `unpause`, `/readyz` trả `200` ngay lần thử đầu.
  - Khi Mongo bị `stop`: `/readyz` trả `503` sau 2.01 s và 2.00 s. Sau `start`, `/readyz` trả `200` ngay lần thử đầu, không phải restart.
  - Khởi động BFF khi Mongo đang dừng (TC16d): `/healthz` trả `200`, `/readyz` trả `503` sau 2.01 s. Sau `start`, `/readyz` lên `200` và không có unhandled rejection.

Góp ý nhỏ (không chặn):
- A5 viết "header lặp (mảng)". Thực tế Node gộp header `x-request-id` bị lặp thành **một chuỗi** `"a, b"`, không phải mảng. Chuỗi này vẫn không khớp regex (vì có `, `) nên được thay bằng UUID v7. Kết quả vẫn đúng, người làm chỉ cần không giả định kiểu `string[]`.

## Ghi chú thực thi

Chạy ngày 2026-09-27, execute-all, bằng `/bin/bash` + `prep.sh`, trên macOS arm64, Node 22.22.3, pnpm 11.18.0, stack `dc` (mongo 8.0.32). TC01–TC22 đều pass.

- **TC01/TC02**: dependency gồm fastify 5.12.5, fastify-plugin 5.1.0, mongodb 7.6.0, pino 10.3.1 (bản fastify đã kéo về, thêm trực tiếp để `server.ts` log lỗi config trước khi có app). Dev dependency gồm typescript 6.0.3, tsx 4.23.15, tsup 8.5.1, vitest 5.0.2, eslint 10.11.0, @eslint/js 10.0.1, typescript-eslint 8.70.1, @types/node 22.20.4. UUID v7 tự viết (`src/request-id.ts`), không thêm `uuid`. `node -e import('fastify')` in `ok` (`default.version` không có).
- **TC03**: `pnpm ignored-builds` in `Cannot identify as no node_modules found` và không liệt kê package nào. `allowBuilds: esbuild: true` là mục giữ chỗ do pnpm tự ghi (A9). `set this to` = 0, `^allowBuilds` = 1.
- **TC04/TC05/TC18**: file tạm `src/__tc_*` được tạo bằng công cụ Write, nội dung giống hệt lệnh `printf` của TC, không tạo qua Bash. Sau đó chạy đúng các lệnh còn lại của TC, và `rmtmp` in `0`.
- **TC05**: script `lint` ban đầu là `eslint . --max-warnings=0`. Dòng pnpm in ra (`$ eslint . --max-warnings=0`) chứa chữ "warning", nên grep đếm `1`. Đã đổi script thành `eslint .` (A1 cho phép cả hai), sau đó đếm `0`.
- **TC06**: 53 test pass, 4 file, env không có `MONGODB_URI`, không cần Docker. `.inject(` xuất hiện 15 lần.
- **TC08**: có hai dòng `Server listening at …` (127.0.0.1 và IP LAN) vì listen `0.0.0.0`.
- **TC13/TC15**: `/readyz` trả 503 sau 2.004 s. Log `warn` có `error_type` `TimeoutError` (hạn của route 2 s bằng `serverSelectionTimeoutMS`). Không có mật khẩu, header, `tcuser`, unhandled rejection.
- **TC16**: (a) pause → 503 2.006 s, unpause → 200 ngay. (b) stop → 503 2.002 s, start → 200 sau 1 s. (c) `same-pid`. (d) khởi động khi Mongo dừng → 503, start → 200 sau 1 s.
- **TC17**: chạy từ gốc repo, `pnpm --dir $APP …` báo `ERR_PNPM_BAD_PM_VERSION`: shim corepack ở `~/.local/bin/pnpm` chọn pnpm 12.6.0 theo cwd (gốc repo không có `packageManager`). Đã chạy đúng lệnh TC nhưng với cwd `$APP` (`cd $APP; bgrun dev … pnpm --dir $APP --filter bff dev`). Kết quả: `/healthz` 200 sau 1 s, `bad=0`, `tsx` = 1, `listening` 2 → 4 (mỗi lần 2 dòng), sau `bgstop` cổng và `tsx watch` đều `0`, sha của `server.ts` không đổi. tsx in `Previous process hasn't exited yet. Force killing...` khi restart: graceful shutdown là việc của P0-T13.
- **TC18**: bước 4 chạy `scripts/ci-app.sh origin/main`: lint/test/build pass, nhưng `git diff --exit-code pnpm-lock.yaml` exit 1, vì lockfile chưa commit (so với HEAD). Sau khi commit thì bước này sạch. `pnpm --filter "...[origin/main]"` chưa chọn package nào khi `bff` còn untracked.
- **TC19**: `make test` (cả ba bộ) exit 0.
- **TC22**: `mongodb` `running healthy` trước `dc down`. Không còn container. Volume giữ nguyên (7). Scratchpad đã xoá.
- **Quy trình**: trong bước 5, agent lỡ tạo một file tạm `apps/bff/tsconfig.tc-notypes.json` bằng `node -e` qua Bash (để thử `types`), rồi xoá ngay bằng `rm`. File không được dùng, không nằm trong diff.
