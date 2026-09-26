# Test cases — P0-T10a: Image OCI cho core (`cmd/server`, `cmd/worker`) bằng macro `com_tm_go_image`

> **Phạm vi**: đóng gói hai binary của core thành image OCI distroless bằng macro `com_tm_go_image` có từ P0-T01a. Build được cho linux-arm64 và linux-amd64, `bazel run …_docker` nạp image vào Docker, image core chạy được trong mạng compose (PG, otel-collector), dừng sạch khi `docker stop`. Căn cứ:
> - Phase 0 README: dòng P0-T10a "Image OCI cho core: dùng macro `com_tm_go_image` (có từ P0-T01a) cho `cmd/server`, `cmd/worker` `[G14]`". Challenge **G14** (🟨): hoàn thành khi "Code build được bằng cả `go` và Bazel; CI chỉ test target bị ảnh hưởng".
> - [project-structure](../../../../project-structure.md) mục "Thiết lập Bazel": macro sinh `<name>`, `<name>_image`, `<name>_docker` (tag `com.tm.go.<name>:v1.0.0`), `<name>_push` chỉ khi có `repository`. Base `gcr.io/distroless/static-debian12:nonroot` pin digest, binary tại `/app/<name>`, image `target_compatible_with` Linux (`bazel build //...` trên macOS bỏ qua image). Lệnh build image: `bazel run --config=linux-arm64 //path:<name>_docker`, `--config=linux-amd64`. Cây thư mục: `services/core/cmd/server/main.go` (wiring), `services/core/cmd/worker/main.go` ("hết hạn hold, relay outbox, sinh slot, đối soát"), `services/stats-worker/cmd/worker/main.go`. Mục CI: `scripts/ci-affected.sh` tính `tests(rdeps(//..., set(...)))`, mọi `*.bzl` là file toàn cục.
> - Handbook [P0-T01a](../../../../handbook/phase-0/P0-T01a.md): toolchain bsdtar cho `bazel run …_docker` khi cross-build, "luôn kiểm kết quả thật (`docker image ls`, `docker run`), không tin exit code của `bazel run`", kiểm distroless bằng `--entrypoint /bin/sh` (exit 127).
> - [execute-all-notes](../../../execute-all-notes.md): mục P0-T01a — Gazelle đặt tên target theo thư mục, `cmd/server` → tag `com.tm.go.server`, `cmd/worker` → `com.tm.go.worker`, "quyết định ở P0-T10a". Mục P0-T09 — `stop_grace_period`/`terminationGracePeriodSeconds` của container core phải > `CORE_SHUTDOWN_TIMEOUT` + 1 s. Mục P0-T10 — container core cần `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318` (mạng compose). Mục P0-T07 — `local-setup.md` có `go run ./services/core/cmd/worker` nhưng binary chưa có.
> - Code hiện có: `services/core/cmd/server/BUILD.bazel` **đã** là `com_tm_go_image(name = "server", …)` (Gazelle `map_kind`), nên tag mặc định là `com.tm.go.server:v1.0.0`. `services/core/cmd/worker` **chưa có**. Compose (`deploy/docker-compose.yml`, project `snaptix`) không khai báo `networks` → mạng mặc định `snaptix_default`, hostname `postgres-core`, `otel-collector`.
> - Planning: việc thật của core worker thuộc task sau — **P4-T08** (worker hết hạn hold, `SKIP LOCKED` + `errgroup`, G4) và **P6-T01** (outbox relay trong `core worker`). Service triển khai nhiều instance core + worker ở phase 7. Docs **không** giao service `core` trong compose, push registry hay ngưỡng kích thước image cho task nào ở phase 0.
>
> **Giả định** (reviewer xem kỹ):
> - A1. **Tên image**. Tag mặc định `com.tm.go.server` / `com.tm.go.worker` không nói service nào, và `services/stats-worker/cmd/worker` (task sau) cũng sẽ ra `com.tm.go.worker` → **trùng tag**. Quyết định: tag = `com.tm.go.<service>-<binary>:v1.0.0`, tức `com.tm.go.core-server:v1.0.0` và `com.tm.go.core-worker:v1.0.0` (stats-worker sau này: `com.tm.go.stats-worker-worker` hoặc tên do task đó chọn, miễn không trùng). **Tên target giữ theo Gazelle** (`//services/core/cmd/server:server_docker`, `…:server_image`, `//services/core/cmd/worker:worker_docker`), entrypoint giữ `/app/server`, `/app/worker`. Cách làm tuỳ người làm (ví dụ thêm tham số tên image vào macro, đặt trong BUILD và được Gazelle giữ nguyên), với ràng buộc: Gazelle `-mode=diff` vẫn exit `0`; lời gọi không truyền tên (như `tools/smoke`) giữ nguyên hành vi cũ `com.tm.go.<name>:v1.0.0`; hàm thuần mới (nếu có) có unit test skylib như `image_names`/`image_tag`. Quy tắc đặt tên ghi vào bảng "Thiết lập Bazel" của `project-structure.md`.
> - A2. **`cmd/worker` là skeleton tối thiểu, thuộc task này**. Dòng task giao rõ image cho `cmd/worker`, macro chỉ sinh image từ một `go_binary`, và project-structure/local-setup đã mô tả `services/core/cmd/worker`. Nên task tạo `services/core/cmd/worker` (`package main`) **chưa có job nào**: log JSON (slog, cùng dạng `time`/`level`/`msg` như core) một dòng khởi động, chờ SIGTERM/SIGINT, rồi log theo quy ước `shutdown_step` của P0-T09 (ít nhất `signal` rồi `done`) và thoát `0`. Worker **không** mở cổng, **không** kết nối PG/Redis, **không** bắt buộc biến môi trường nào (config/pool/OTel thêm khi có job thật ở P4-T08/P6-T01). Có unit test cho phần logic (ví dụ hàm `run(ctx, …)` trả về khi `ctx` huỷ và log đúng mốc). Phương án bị loại: chỉ làm `server`, để `cmd/worker` ra ngoài phạm vi — bỏ một nửa dòng task, và `local-setup.md` tiếp tục ghi lệnh chạy binary không tồn tại.
> - A3. **Không thêm service core vào compose, không push**. Image core chạy bằng `docker run --network snaptix_default` (không sửa `deploy/**`). Không truyền `repository` → không có target `_push`. Không viết `Dockerfile`.
> - A4. **Kích thước**: docs không nêu ngưỡng. Image phải chỉ gồm base distroless static (≈ 2 MiB, 12 layer) và **một** layer chứa đúng binary `/app/<name>`, nên tiêu chí là `layers = 13` và tổng byte các layer ≤ kích thước binary + 5 MiB (không có tệp thừa). Con số cụ thể được ghi nhận, không đặt ngưỡng tuyệt đối. **Lỗi đã biết của macro P0-T01a, task này phải sửa**: `tar` trong `com_tm_go_image` hiện đóng gói cả runfiles (`app/<name>.runfiles/_main/.../<name>` — bản sao thứ hai của binary; smoke: layer 3152896 byte với binary 1573024 byte), nên với binary core cỡ chục MiB tiêu chí trên sẽ fail. Task tắt đóng gói runfiles (ví dụ `tar(..., include_runfiles = False)`, tar.bzl 0.10.9 hỗ trợ) — thuộc `tools/rules/**` (TC16 cho phép), hồi quy smoke kiểm ở TC14. Số layer: reviewer đo smoke hiện `layers=13` (base 12 + 1); nếu base/macro đổi số layer thì TC07 so với `ociinfo` của smoke build cùng lúc.
> - A5. **Env trong container**: dùng lại env của P0-T07/P0-T10 (không thêm biến mới). `CORE_HTTP_ADDR` mặc định `:8080` nghe mọi interface nên publish được cổng. Test publish `127.0.0.1:18080→8080` để không đụng core chạy trên host (8080).
> - A6. **Graceful shutdown trong container** là hành vi của P0-T09/P0-T10 (binary là entrypoint, chạy PID 1, nhận SIGTERM từ `docker stop`). Task này chỉ kiểm nó vẫn đúng khi chạy trong image, không đổi code shutdown của server. `docker stop -t` phải > `CORE_SHUTDOWN_TIMEOUT` + 1 s (ghi trong docs).
>
> **Không thuộc task này**: job thật của worker (P4-T08 hết hạn hold, P6-T01 relay outbox), stats-worker và image của nó, service core/worker trong `deploy/docker-compose.yml`, push registry / `repository`, CI build image, chạy CI thật trên GitHub (kiểm khi đóng phase), BFF (P0-T11+).
>
> **Môi trường khi viết**: macOS arm64 (Apple Silicon), Go 1.27.1, Bazel 8.7.0 (Bazelisk), Docker Desktop 29.8 (containerd image store), stack P0-T02, curl, jq, `/usr/bin/python3` 3.9, `file`.
>
> **Chuẩn bị**: lưu khối dưới thành `<scratchpad>/p0t10a/prep.sh` một lần (tạo bằng Write tool, thay `<scratchpad>` bằng đường dẫn thật). Mỗi lần gọi Bash bắt đầu bằng `source <scratchpad>/p0t10a/prep.sh`.
> - Chạy được trên cả bash 3.2 lẫn zsh. Không dùng biến tên `status`/`path`. Compose gọi qua `dc`, Bazel qua `bz` (chạy trong `com/tm/server`). `nap` dùng python (harness có thể chặn `sleep`).
> - Phần chuẩn bị và các TC **không** có lệnh ghi lịch sử git (không `git commit`/`add`/`stash`).
> - Trước TC08: `dc up -d --wait` (toàn stack) đã chạy, TC04 đã nạp image. Core **không** cần migration.
> ```bash
> REPO=/Users/maiductoan/Documents/code/jarvis; cd $REPO
> SRV=$REPO/com/tm/server; CORE=$SRV/services/core
> S=<scratchpad>/p0t10a; mkdir -p $S
> dc() { docker compose -f "$REPO/deploy/docker-compose.yml" "$@"; }
> bz() { (cd $SRV && bazel "$@"); }
> # files [FLAGS] LABEL — đường dẫn TUYỆT ĐỐI output của target (cquery in đường dẫn tương đối từ $SRV).
> # Lưu ý: host, --config=linux-arm64 và --config=linux-amd64 cho CÙNG đường dẫn bazel-out/darwin_arm64-fastbuild/...
> # → file ở đó là của lần build gần nhất; muốn đọc theo arch thì dùng snap rồi đọc $S/<arch>/.
> files() { bz cquery --output=files "$@" 2>/dev/null | sed "s|^|$SRV/|"; }
> # snap ARCH — build đúng --config=linux-ARCH rồi copy binary + OCI layout của server/worker ra $S/ARCH/{server,worker,server_image,worker_image}
> snap() { local a=$1 b; bz build --config=linux-$a $SL:server $SL:server_image $WL:worker $WL:worker_image || return 1
>          chmod -R u+w "$S/$a" 2>/dev/null; rm -rf "$S/$a"; mkdir -p "$S/$a"
>          for b in server worker; do cp -L "$(files --config=linux-$a //services/core/cmd/$b:$b)" "$S/$a/$b" || return 1
>            cp -RL "$(files --config=linux-$a //services/core/cmd/$b:${b}_image)" "$S/$a/${b}_image" || return 1; done; chmod -R u+w "$S/$a"; }  # output Bazel read-only → mở quyền ghi để xoá được
> NET=snaptix_default
> IMG=com.tm.go.core-server:v1.0.0; WIMG=com.tm.go.core-worker:v1.0.0
> SL=//services/core/cmd/server; WL=//services/core/cmd/worker
> CN=p0t10a-core; WN=p0t10a-worker; HP=18080; C=http://127.0.0.1:$HP; TEMPO=http://localhost:3200
> DSN_NET='postgres://snaptix:snaptix@postgres-core:5432/core?sslmode=disable'
> DSN_HOST='postgres://snaptix:snaptix@host.docker.internal:5432/core?sslmode=disable'
> ENV_NET=(-e "CORE_DATABASE_URL=$DSN_NET" -e OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318 -e OTEL_BSP_SCHEDULE_DELAY=500)
> nap()   { python3 -c 'import time,sys;time.sleep(float(sys.argv[1]))' "$1"; }
> now()   { python3 -c 'import time;print("%.3f"%time.time())'; }
> since() { python3 -c "import time;print('%.2f'%(time.time()-$1))"; }
> hex()   { python3 -c 'import secrets,sys;print(secrets.token_hex(int(sys.argv[1])))' "$1"; }
> ctrm()  { docker rm -f $CN $WN >/dev/null 2>&1; true; }
> # crun [docker-run-args...] — chạy image core nền, publish 127.0.0.1:18080→8080, mạng compose
> crun()  { ctrm; docker run -d --name $CN --network $NET -p 127.0.0.1:$HP:8080 "$@" $IMG; }
> cstate() { docker inspect -f '{{.State.Status}} exit={{.State.ExitCode}} oom={{.State.OOMKilled}}' "$1"; }
> clog()  { docker logs "$1" >"$S/$1.log" 2>&1; echo "$S/$1.log"; }
> code()  { curl -s -m 10 -o /dev/null -w '%{http_code}\n' "$@"; }
> waitcode() { local s=$(date +%s) c; while :; do c=$(curl -s -m 5 -o /dev/null -w '%{http_code}' "$1")
>              [ "$c" = "$2" ] && { echo "$c after $(( $(date +%s)-s ))s"; return 0; }
>              [ $(( $(date +%s)-s )) -ge "$3" ] && { echo "$c TIMEOUT"; return 1; }; nap 0.3; done; }
> tp()    { echo "traceparent: 00-$1-$2-01"; }
> # steps FILE — chuỗi shutdown_step; tflush FILE — dòng telemetry_flush
> steps() { jq -rR 'fromjson? | select(type=="object" and has("shutdown_step")) | .shutdown_step' "$1" | tr '\n' ' '; echo; }
> tflush() { jq -cR 'fromjson? | select(type=="object" and has("telemetry_flush")) | {level,telemetry_flush}' "$1"; }
> jsonlines() { python3 -c 'import json,sys
> n=bad=0
> for l in open(sys.argv[1]):
>     l=l.strip()
>     if not l: continue
>     n+=1
>     try: o=json.loads(l); assert isinstance(o,dict) and {"time","level","msg"}<=o.keys()
>     except Exception: bad+=1; print("BAD:",l[:200])
> print("lines=%d bad=%d"%(n,bad))' "$1"; }
> # ociinfo DIR — đọc OCI layout do rules_oci sinh: os/arch, user, entrypoint, tổng byte các layer
> ociinfo() { python3 -c 'import json,os,sys
> d=sys.argv[1]; b=lambda h: json.load(open(os.path.join(d,"blobs",*h.split(":"))))
> m=b(json.load(open(os.path.join(d,"index.json")))["manifests"][0]["digest"])
> c=b(m["config"]["digest"]); cc=c.get("config",{})
> print("%s/%s user=%s entrypoint=%s layers=%d layer_bytes=%d"%(c["os"],c["architecture"],cc.get("User"),json.dumps(cc.get("Entrypoint")),len(m["layers"]),sum(l["size"] for l in m["layers"])))' "$1"; }
> # lastlayer DIR — liệt kê entry của layer cuối (layer binary) trong OCI layout
> lastlayer() { python3 -c 'import json,os,sys,tarfile
> d=sys.argv[1]; p=lambda h: os.path.join(d,"blobs",*h.split(":"))
> m=json.load(open(p(json.load(open(os.path.join(d,"index.json")))["manifests"][0]["digest"])))
> for t in tarfile.open(p(m["layers"][-1]["digest"]),"r:*"): print(t.name)' "$1"; }
> tcode() { curl -s -m 5 -o /dev/null -w '%{http_code}' -H 'Accept: application/json' "$TEMPO/api/traces/$1"; }
> waittrace() { local s=$(date +%s) c; while :; do c=$(tcode "$1"); [ "$c" = 200 ] && { echo "trace 200 after $(( $(date +%s)-s ))s"; return 0; }
>              [ $(( $(date +%s)-s )) -ge "$2" ] && { echo "trace $c TIMEOUT"; return 1; }; nap 1; done; }
> # tspans TID — mỗi span một dòng: name, kind, id, parent, db, service.name, service.version
> tspans() { python3 -c 'import base64,json,sys,urllib.request
> r=urllib.request.Request("http://localhost:3200/api/traces/"+sys.argv[1],headers={"Accept":"application/json"})
> d=json.load(urllib.request.urlopen(r,timeout=10)); d=d.get("trace",d)
> h=lambda v: "" if not v else (v.lower() if len(v) in (16,32) and all(c in "0123456789abcdefABCDEF" for c in v) else base64.b64decode(v).hex())
> val=lambda v: next((v[k] for k in ("stringValue","intValue","boolValue") if k in v), v)
> at=lambda a: dict((x["key"],val(x.get("value",{}))) for x in (a or []))
> K={1:"INTERNAL",2:"SERVER",3:"CLIENT"}
> for b in d.get("batches") or d.get("resourceSpans") or []:
>     ra=at((b.get("resource") or {}).get("attributes"))
>     for ss in b.get("scopeSpans") or []:
>         for s in ss.get("spans",[]):
>             a=at(s.get("attributes")); k=s.get("kind","")
>             print(json.dumps({"name":s.get("name"),"kind":K.get(k,k) if isinstance(k,int) else str(k).replace("SPAN_KIND_",""),"id":h(s.get("spanId")),"parent":h(s.get("parentSpanId")),
>                 "db":a.get("db.system.name") or a.get("db.system"),"svc":ra.get("service.name"),"ver":ra.get("service.version")},sort_keys=True))' "$1"; }
> ```

| ID | Loại | Kịch bản | Kết quả mong đợi | Trạng thái |
|---|---|---|---|---|
| P0-T10a-TC01 | Cấu trúc | BUILD dùng macro qua Gazelle `map_kind` (A2, G14): `ls $CORE/cmd $CORE/cmd/worker`; `grep -n 'com_tm_go_image\|go_binary(' $CORE/cmd/server/BUILD.bazel $CORE/cmd/worker/BUILD.bazel`; `bz run //:gazelle -- -mode=diff; echo $?`; `bz query "attr(generator_function, com_tm_go_image, $SL:* + $WL:*)"`; `bz query "$SL:all + $WL:all" \| grep -c '_push$'`; `bz query "kind(go_test, $WL:all)"`; `find $SRV -name 'Dockerfile*' -not -path '*/bazel-*'` | `cmd/` có đúng `server` và `worker`; `cmd/worker` có `main.go` (+ file khác nếu cần), file `_test.go` và `BUILD.bazel`. Cả hai BUILD `load` macro từ `//tools/rules:com_tm_container.bzl` và gọi `com_tm_go_image(name = "server"…)` / `com_tm_go_image(name = "worker"…)`, không có `go_binary(` trực tiếp. Gazelle `-mode=diff` không in diff, exit `0` (thuộc tính đặt tên image, nếu có, được Gazelle giữ). `bazel query` có `server`, `server_layer`, `server_image`, `server_docker`, `worker`, `worker_layer`, `worker_image`, `worker_docker`. Số target `_push` = `0` (A3). Có `go_test` cho `//services/core/cmd/worker`. Không có `Dockerfile` | ✅ |
| P0-T10a-TC02 | Macro | Quyết định tên image (A1) và không làm hỏng lời gọi cũ: `bz test //tools/rules/...; echo $?`; `grep -n 'image_names\|image_tag\|def ' $SRV/tools/rules/com_tm_container.bzl`; `grep -rn 'com.tm.go.core-server\|com.tm.go.core-worker' $CORE/cmd $SRV/tools/rules`; `grep -n 'com.tm.go' com/tm/docs/technical/project-structure.md` | Test skylib của macro exit `0`, gồm case cũ (`image_tag("smoke")` = `com.tm.go.smoke:v1.0.0`) và case mới cho tên image của core nếu thêm hàm/tham số (ví dụ `com.tm.go.core-server:v1.0.0`). Tag `com.tm.go.core-server:v1.0.0` và `com.tm.go.core-worker:v1.0.0` được khai báo (trong BUILD của core hoặc suy ra được bằng hàm có test). Bảng "Thiết lập Bazel" của `project-structure.md` ghi quy tắc tag mới (`com.tm.go.<service>-<binary>` cho binary của service, `com.tm.go.<name>` khi không đặt) và ví dụ `core-server` | ✅ |
| P0-T10a-TC03 | Build | Build bằng `go` và Bazel trên darwin, image bị bỏ qua (G14): `(cd "$SRV" && go vet ./... && go build ./... && go test -race -count=1 ./... && echo OK)`; `gofmt -l $CORE`; `(cd "$SRV" && golangci-lint run ./...); echo $?`; `bz build //...; echo $?`; `bz test //...; echo $?`; `bz build $SL:server_image 2>&1 \| grep -ciE 'incompatible\|skipp'`; `git -C $REPO status --porcelain --untracked-files=all com/tm/server \| grep -vE '\.go$\|BUILD\.bazel$\|\.bzl$'` | In `OK` (gồm test worker). `gofmt -l` rỗng. golangci-lint `0 issues`, exit `0`. `bazel build //...` và `bazel test //...` exit `0` trên macOS (target image SKIPPED, không lỗi thiếu base darwin). Build riêng `server_image` không có `--config` báo target không tương thích (≥ `1`). Lệnh `git status` cuối không in dòng nào (không có binary/file rác) | ✅ |
| P0-T10a-TC04 | Image | Build arm64 và nạp vào Docker (A1): `docker image rm $IMG $WIMG >/dev/null 2>&1; true`; `bz run --config=linux-arm64 $SL:server_docker; echo $?`; `bz run --config=linux-arm64 $WL:worker_docker; echo $?`; `docker image ls --format '{{.Repository}}:{{.Tag}}' \| grep -E '^com\.tm\.go\.core-(server\|worker):'`; `for i in $IMG $WIMG; do docker image inspect $i -f '{{.Os}}/{{.Architecture}} user={{.Config.User}} ep={{json .Config.Entrypoint}}'; done`; `snap arm64; echo $?`; `ociinfo $S/arm64/server_image; ociinfo $S/arm64/worker_image` | Hai `bazel run` exit `0` **và** `docker image ls` có đúng `com.tm.go.core-server:v1.0.0` và `com.tm.go.core-worker:v1.0.0` (không tin riêng exit code, handbook P0-T01a). Inspect: `linux/arm64 user=65532 ep=["/app/server"]` và `linux/arm64 user=65532 ep=["/app/worker"]` (distroless `nonroot`, user có thể hiển thị `65532` hoặc `nonroot`, không phải rỗng/`root`/`0`). `snap arm64` exit `0`; `ociinfo` (bản copy arm64) in `linux/arm64 user=65532`, entrypoint `["/app/server"]` / `["/app/worker"]` | ✅ |
| P0-T10a-TC05 | Image | Cross-build amd64, binary tĩnh. Host/arm64/amd64 dùng chung `bazel-out/darwin_arm64-fastbuild`, nên mỗi arch build rồi copy riêng ra `$S/<arch>/` (`snap`) và chỉ đọc bản copy: `snap amd64; echo $?`; `snap arm64; echo $?`; `for a in amd64 arm64; do for p in server worker; do file -L $S/$a/$p; done; done`; `ociinfo $S/amd64/server_image`; `ociinfo $S/amd64/worker_image`; `ociinfo $S/arm64/server_image` | Hai `snap` exit `0`. `file`: bản amd64 là `ELF 64-bit LSB executable, x86-64`, bản arm64 là `ELF 64-bit LSB executable, ARM aarch64`, cả bốn đều `statically linked` (CGO off, chạy được trên distroless static). `ociinfo` bản amd64 in `linux/amd64 user=65532 entrypoint=["/app/server"]` và `…["/app/worker"]`; bản arm64 của server vẫn `linux/arm64` (hai bản copy không ghi đè nhau) | ✅ |
| P0-T10a-TC06 | Image (negative) | Distroless, không shell, chạy non-root: `for i in $IMG $WIMG; do docker run --rm --entrypoint /bin/sh $i -c 'echo hi'; echo "sh=$?"; docker run --rm --entrypoint /busybox/sh $i -c 'echo hi'; echo "busybox=$?"; done`; `crun "${ENV_NET[@]}"; waitcode $C/healthz 200 15`; `docker top $CN -eo uid,pid,args`; `docker exec $CN /bin/sh -c id; echo "exec=$?"`; `ctrm` | Mọi lệnh có shell đều lỗi, exit ≠ `0` (thường `127`), không in `hi`. Core chạy: `200`. `docker top` cho process `/app/server` với uid `65532` (không phải `0`). `docker exec … /bin/sh` lỗi (exit ≠ `0`) | ✅ |
| P0-T10a-TC07 | Image | Kích thước (A4, chỉ ghi nhận + tiêu chí không có tệp thừa), đo đúng artifact arm64 (build lại rồi copy, không đọc `bazel-out` có thể là bản amd64 của TC05): `snap arm64; echo $?`; `for p in server worker; do B=$(stat -f %z $S/arm64/$p); echo "$p binary=$B"; ociinfo $S/arm64/${p}_image; lastlayer $S/arm64/${p}_image; lastlayer $S/arm64/${p}_image \| grep -c runfiles; done`; `bz build --config=linux-arm64 //tools/smoke:smoke_image && ociinfo "$(files --config=linux-arm64 //tools/smoke:smoke_image)"`; `docker image ls --format '{{.Repository}}:{{.Tag}} {{.Size}}' \| grep com.tm.go.core-` | `snap` exit `0`. Mỗi image có `layers=13` (base distroless 12 + 1 layer binary; bằng số layer của smoke build cùng lúc). Layer cuối chỉ chứa thư mục `app` và `app/server` (hoặc `app/worker`), **không** có `.runfiles` (`grep -c runfiles` = `0`) — macro đã tắt đóng gói runfiles (A4). `layer_bytes` ≤ `binary` + 5 MiB (5242880) với cả hai image. Ghi nhận kích thước binary, `layer_bytes` và cột SIZE của Docker vào mục kết quả (không có ngưỡng tuyệt đối) | ✅ |
| P0-T10a-TC08 | E2E | Chạy image core trong mạng compose (A5), P0-FR3, log JSON: `crun "${ENV_NET[@]}"`; `waitcode $C/healthz 200 15`; `waitcode $C/readyz 200 15`; `code $C/metrics`; `code $C/khong-co`; `code -H 'X-Request-ID: tc08-a' $C/healthz`; `nap 1; L=$(clog $CN)`; `jq -cR 'fromjson? \| select(type=="object" and .request_id=="tc08-a") \| {request_id,status,latency_ms,route,trace_id}' $L`; `jsonlines $L`; `cstate $CN`; `ctrm` | `/healthz` `200` trong ≤ 15 s, `/readyz` `200` (ping được `postgres-core` qua mạng compose), `/metrics` `200`, `/khong-co` `404`. Access log `tc08-a`: `status` = `200`, `latency_ms` là số, `route` = `/healthz`, `trace_id` 32 hex. `bad=0` (mọi dòng log là JSON có `time`/`level`/`msg`). `cstate` = `running` | ✅ |
| P0-T10a-TC09 | E2E | Trace từ container lên Tempo qua `otel-collector:4318` (ghi chú P0-T10): `crun "${ENV_NET[@]}"`; `waitcode $C/healthz 200 15`; `TID=$(hex 16); code -H "$(tp $TID $(hex 8))" $C/readyz`; `waittrace $TID 30; nap 3`; `tspans $TID`; `L=$(clog $CN); jq -cR 'fromjson? \| select(type=="object" and (.level=="WARN" or .level=="ERROR"))' $L \| grep -ciE 'otel\|otlp\|export'`; `ctrm` | `/readyz` `200`. Trace lên Tempo trong ≤ 30 s. Có span `GET /readyz` `SERVER` với `svc` = `core`, `ver` khác rỗng/null (binary do Bazel build vẫn có `service.version`, ví dụ `dev`/`(devel)`), và ≥ 1 span `db` = `postgresql` có `parent` dẫn tới span server. Không có WARN/ERROR về export OTel (`0`) | ✅ |
| P0-T10a-TC10 | E2E | Biến thể `host.docker.internal` và lỗi cấu hình trong container: (a) `crun -e "CORE_DATABASE_URL=$DSN_HOST" -e OTEL_EXPORTER_OTLP_ENDPOINT=http://host.docker.internal:4318`; `waitcode $C/readyz 200 15`; `ctrm`. (b) Thiếu DSN: `docker run -d --name $CN $IMG`; `nap 5`; `cstate $CN`; `L=$(clog $CN); jq -cR 'fromjson? \| select(type=="object" and .level=="ERROR")' $L \| grep -c CORE_DATABASE_URL`; `ctrm` | (a) `/readyz` `200` (core trong container nối PG qua cổng host 5432). (b) Container đã dừng: `exited exit=1`, có ≥ `1` dòng log `ERROR` nêu `CORE_DATABASE_URL` (hành vi P0-T07 giữ nguyên trong image) | ✅ |
| P0-T10a-TC11 | E2E | `docker stop` → graceful shutdown (A6, P0-T09/P0-T10): `crun "${ENV_NET[@]}" -e CORE_SHUTDOWN_TIMEOUT=3s`; `waitcode $C/readyz 200 15`; `for i in 1 2 3 4 5; do code $C/healthz; done >/dev/null`; `T0=$(now); docker stop -t 10 $CN; echo "after=$(since $T0)s"`; `cstate $CN`; `L=$(clog $CN)`; `steps $L`; `tflush $L`; `jq -cR 'fromjson? \| select(type=="object" and .shutdown_step=="done") \| {level,exit_code}' $L`; `jsonlines $L`; `ctrm` | `after` < `5`s (dừng bằng SIGTERM, không phải SIGKILL sau 10 s). `cstate` = `exited exit=0 oom=false` (không phải `137`/`143`). `steps` = `signal http_stopped pool_closed done` (y hệt P0-T09). `tflush` = `{"level":"INFO","telemetry_flush":"ok"}`. Dòng `done`: `INFO`, `exit_code` = `0`. `bad=0` | ✅ |
| P0-T10a-TC12 | E2E + Unit | Worker skeleton (A2): `(cd $SRV && go test -race -count=1 -v ./services/core/cmd/worker/ 2>&1 \| grep -E '^\s*--- (PASS\|FAIL\|SKIP)')`; `grep -rn 't\.Skip' $CORE/cmd/worker`; `(cd $SRV && go list -deps ./services/core/cmd/worker \| grep -cE 'jackc/pgx\|net/http$\|redis')`; `docker run -d --name $WN $WIMG`; `nap 2; cstate $WN; docker port $WN \| wc -l \| tr -d ' '`; `T0=$(now); docker stop -t 10 $WN; echo "after=$(since $T0)s"`; `cstate $WN`; `L=$(clog $WN); jsonlines $L; steps $L`; `ctrm` | Có ≥ 1 subtest `--- PASS`, không `FAIL`/`SKIP`, không `t.Skip`. Worker không phụ thuộc pgx, `net/http`, redis (`0`) — chưa có job, không mở cổng, không nối DB. Container chạy không cần env nào: `running`, không có cổng (`0`). Sau `docker stop`: `after` < `3`s, `exited exit=0`. Log `bad=0`, có ≥ 1 dòng trước tín hiệu (khởi động), `steps` bắt đầu bằng `signal` và kết thúc bằng `done` | ✅ |
| P0-T10a-TC13 | CI | Chỉ test target bị ảnh hưởng (G14; CI thật chạy khi đóng phase): `bz query "tests(rdeps(//..., set($SL:all)))"`; `bz query "tests(rdeps(//..., set($WL:all)))"`; `bz query "tests(rdeps(//..., set(//tools/smoke:all)))"`; `bash scripts/ci-affected_test.sh; echo $?`; `bash scripts/ci-server_test.sh; echo $?` | Đổi `cmd/server` → chỉ `//services/core/cmd/server:server_test` (không có `smoke_test`, `otelx_test`, test của worker). Đổi `cmd/worker` → chỉ test của `//services/core/cmd/worker`. Đổi `tools/smoke` → chỉ `//tools/smoke:smoke_test`. Test của `ci-affected.sh` (gồm luật `*.bzl` là file toàn cục → `//...`, áp dụng khi sửa `com_tm_container.bzl`) và `ci-server.sh` pass, exit `0` | ✅ |
| P0-T10a-TC14 | Hồi quy | P0-T01a/P0-T05 không vỡ: `docker image rm com.tm.go.smoke:v1.0.0 >/dev/null 2>&1; bz run --config=linux-arm64 //tools/smoke:smoke_docker; docker run --rm com.tm.go.smoke:v1.0.0; echo $?`; `bz run //:gazelle -- -mode=diff; echo $?`; `git -C $REPO diff --exit-code -- com/tm/server/go.mod com/tm/server/go.sum com/tm/server/MODULE.bazel com/tm/server/MODULE.bazel.lock com/tm/server/.bazelrc; echo $?`; `bash scripts/check-structure.sh; echo $?`; `bash scripts/test-all.sh server; echo $?` | Smoke in `snaptix smoke ok`, exit `0`, tag vẫn `com.tm.go.smoke:v1.0.0` (A1: lời gọi không đặt tên giữ hành vi cũ). Gazelle `0`. `go.mod`/`go.sum`/`MODULE.bazel`/lockfile/`.bazelrc` không đổi (`0`): task không cần dependency hay config Bazel mới. `check-structure.sh` `0`. `test-all.sh server` `0` | ✅ |
| P0-T10a-TC15 | Tài liệu | Docs khớp code: `grep -nE 'server_docker\|worker_docker\|core-server\|core-worker\|snaptix_default\|otel-collector:4318\|postgres-core:5432\|docker stop' com/tm/docs/technical/local-setup.md`; `grep -n 'cmd/worker' com/tm/docs/technical/local-setup.md`; `grep -n 'CORE_SHUTDOWN_TIMEOUT' com/tm/docs/technical/local-setup.md \| grep -cE '\+ *1 *s'`; `grep -nE 'docker stop -t [0-9]+' com/tm/docs/technical/local-setup.md`; `grep -n 'G14' com/tm/docs/technical/planning/phase-0-foundation/README.md \| grep -c '🟨'`; `ls com/tm/docs/technical/handbook/phase-0/P0-T10a.md && grep -c 'P0-T10a' com/tm/docs/technical/handbook/README.md` | `local-setup.md` mục "Build & kiểm thử" có lệnh `bazel run --config=linux-arm64 //services/core/cmd/server:server_docker` (và `worker_docker`), tag `com.tm.go.core-server:v1.0.0`, ví dụ `docker run` với `--network snaptix_default`, `CORE_DATABASE_URL=…@postgres-core:5432/…`, `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318`, và `docker stop -t` lớn hơn `CORE_SHUTDOWN_TIMEOUT` + 1 s. Dòng `go run ./services/core/cmd/worker` còn và nay chạy được. Riêng trong `local-setup.md` (không tính `architecture.md:92` có sẵn): ≥ `1` dòng `CORE_SHUTDOWN_TIMEOUT` nêu quy tắc "+ 1 s", và ≥ 1 lệnh `docker stop -t N` với N > `CORE_SHUTDOWN_TIMEOUT` của ví dụ + 1 (mặc định `10s` → N ≥ `12`). G14 vẫn 🟨 (`1`, còn CI thật). Có handbook `P0-T10a.md` và dòng trong `handbook/README.md` (≥ `1`) | ✅ |
| P0-T10a-TC16 | Phạm vi | Không làm việc của task khác (A3): `git -C $REPO status --porcelain --untracked-files=all`; `git -C $REPO status --porcelain deploy com/tm/app`; `grep -nE '^  (core\|worker\|core-server):' deploy/docker-compose.yml`; `ls $SRV/services`; `grep -rn 'repository *=' $CORE --include=BUILD.bazel` | Thay đổi chỉ nằm trong: `com/tm/server/services/core/cmd/**`, `com/tm/server/tools/rules/**` (nếu đổi macro/test macro), `com/tm/docs/technical/{local-setup.md,project-structure.md,architecture.md}`, handbook (`handbook/phase-0/P0-T10a.md`, `handbook/README.md`), planning (`planning/README.md`, `phase-0-foundation/README.md`, `tasks/P0-T10a/**`, `acceptance-tests.md`), `planning/execute-all-notes.md`. `deploy` và `com/tm/app` không đổi (rỗng). Compose không có service core/worker (rỗng). `services/` chỉ có `core` (chưa có stats-worker). Không có `repository =` (không push) | ✅ |
| P0-T10a-TC17 | Dọn dẹp | `ctrm`; `docker ps -a --filter name=p0t10a -q \| wc -l \| tr -d ' '`; `docker image rm $IMG $WIMG com.tm.go.smoke:v1.0.0 >/dev/null 2>&1; true`; `docker image ls --format '{{.Repository}}' \| grep -c '^com\.tm\.go\.core-'`; `dc down`; `docker ps -q --filter label=com.docker.compose.project=snaptix \| wc -l \| tr -d ' '`; `lsof -nP -iTCP:$HP -sTCP:LISTEN \| wc -l \| tr -d ' '`; `chmod -R u+w $S 2>/dev/null; rm -rf $S; ls -d $S 2>/dev/null \| wc -l \| tr -d ' '` | Không còn container `p0t10a-*` (`0`), không còn image `com.tm.go.core-*` (`0`), không còn container của project (`0`), cổng 18080 trống (`0`). Volume giữ nguyên (`down` không `-v`). Scratchpad đã xoá (`0`, bản copy read-only từ `snap` cũng xoá được) | ✅ |

Test nghiệm thu liên quan:
- Không có test nghiệm thu nào của phase 0 nói trực tiếp về image. Task góp phần cho **G14** ("build được bằng cả `go` và Bazel": TC03–TC05, TC14) và cho **P0-AT11** (CI không chạy job Bazel khi chỉ sửa `com/tm/app/**`) ở mức chọn target (TC13). P0-AT11 cần CI thật trên GitHub khi đóng phase → để ⬜.
- **P0-AT02** (`core /readyz` 200 khi PG chạy): TC08 kiểm thêm trong container, trạng thái đã ✅ từ trước, không đổi.

## Review

### Review lần 1 — CHANGES_REQUESTED

Reviewer độc lập (execute-all). Phạm vi, giả định và định dạng nhìn chung ổn. Có hai lỗi làm lệnh không chạy đúng, nên chưa duyệt. Kết quả đã kiểm chứng trên máy dev (không sửa file, đã dọn image tạm):

- `bazel cquery --output=files //tools/smoke:smoke_image --config=linux-arm64` → `bazel-out/darwin_arm64-fastbuild/bin/tools/smoke/smoke_image`, là OCI layout (`index.json`, `oci-layout`, `blobs/sha256`). Cấu trúc mà `ociinfo` đọc là đúng: trên smoke nó in `linux/arm64 user=65532 entrypoint=["/app/smoke"] layers=13 layer_bytes=3868624`.
- `bazel run --config=linux-arm64 //tools/smoke:smoke_docker` rồi `docker image inspect` → `Config.User` = `65532`, nên TC04/TC05 kỳ vọng đúng.
- Build `//services/core/cmd/server:server` bằng Bazel, `go version -m` không có dòng `mod` → `bi.Main.Version` rỗng → `buildVersion()` trả `dev`. TC09 "`ver` khác rỗng/null" là đạt được.
- `cquery --output=files` của `//tools/smoke:smoke` với host, `--config=linux-arm64` và `--config=linux-amd64` đều ra **cùng một đường dẫn** `bazel-out/darwin_arm64-fastbuild/...`, vì Bazel 8 không tách thư mục output theo `--platforms`.
- Layer binary của smoke là 3152896 byte, gấp đôi binary (1573024). Chạy `tar -tvf` thấy layer chứa cả `app/smoke` lẫn `app/smoke.runfiles/_main/tools/smoke/smoke_/smoke`, tức macro hiện tại đóng gói luôn runfiles (bản sao thứ hai của binary).

Điểm phải sửa (chặn):

1. **P0-T10a-TC04, TC05, TC07: đường dẫn tương đối.** `files()` chạy `bz` trong subshell `cd $SRV`, nên in ra đường dẫn tương đối `bazel-out/...`. Nhưng `ociinfo`, `file -L` và `stat -f %z` lại chạy ở `$REPO` (prep đã `cd $REPO`), nên báo không tìm thấy file. Sửa helper cho ra đường dẫn tuyệt đối, ví dụ `files() { bz cquery --output=files "$@" 2>/dev/null | sed "s|^|$SRV/|"; }`.
2. **P0-T10a-TC05 (và TC07): amd64/arm64 dùng chung thư mục output.** Sau lệnh `bz build --config=linux-amd64 …`, `files --config=linux-arm64 $t` trỏ tới **đúng file vừa build amd64**. Vì vậy `file -L` không thể cùng lúc ra "x86-64" và "ARM aarch64" như kỳ vọng, và TC05 luôn fail. TC07 chạy sau TC05 cũng đo nhầm binary/image amd64. Cần build lại đúng config ngay trước khi đọc file, hoặc copy output ra `$S` sau mỗi lần build. Ví dụ: `for a in amd64 arm64; do bz build --config=linux-$a $SL:server $WL:worker $SL:server_image $WL:worker_image; for t in $SL:server $WL:worker; do file -L "$(files --config=linux-$a $t)"; done; …; done`. TC07 phải `bz build --config=linux-arm64 …` trước khi `stat`/`ociinfo`.

Góp ý (không chặn, nên sửa luôn khi sửa hai điểm trên):

- TC07/A4: kiểm chứng ở trên cho thấy macro hiện tại đóng gói runfiles, nên `layer_bytes ≈ 2×binary + ~0.7 MiB`. Với core (binary cỡ chục MiB), tiêu chí `≤ binary + 5 MiB` **sẽ fail** nếu không sửa macro. Tiêu chí này hợp lý vì nó bắt đúng "tệp thừa", và sửa được trong phạm vi task: `tar(..., include_runfiles = False)` có sẵn trong tar.bzl 0.10.9, thuộc `tools/rules/**` mà TC16 cho phép, và TC14 kiểm hồi quy smoke. Tuy vậy nên ghi rõ trong A4 rằng đây là lỗi đã biết của macro P0-T01a phải sửa, và câu "Image chỉ gồm base … và một layer chứa binary" hiện chưa đúng. "Số layer base + 1" nên ghi cụ thể: base distroless hiện có 12 layer, tức `layers = 13`, hoặc so với `ociinfo` của smoke.
- A1: khả thi. Gazelle chỉ merge các attr mà nó sinh ra (`embed`, `srcs`, `deps`…), còn attr lạ trên rule đã có (ví dụ `image = "core-server"`) được giữ nguyên, nên `-mode=diff` vẫn exit 0. Căn cứ trùng tag với `services/stats-worker/cmd/worker` là xác đáng. Macro phải tự tiêu thụ tham số mới, không được đẩy vào `**kwargs` của `go_binary`.
- A2: hợp lý. Dòng task nêu rõ `cmd/worker`, `project-structure.md`/`local-setup.md` đã mô tả binary này, còn P4-T08 (G4) và P6-T01 mới thêm job thật. Skeleton không DB, không cổng không mâu thuẫn docs. A3, A5, A6: đúng phạm vi.
- TC03: `cd $SRV && …` rồi `golangci-lint run ./...` phụ thuộc cwd từ lệnh trước. Nên ghi rõ `(cd $SRV && golangci-lint run ./...)`.
- TC15: dòng `CORE_SHUTDOWN_TIMEOUT … + 1 s` đã có sẵn trong `architecture.md:92`, nên điều kiện "≥ 1" đạt kể cả khi `local-setup.md` không thêm gì. Tiêu chí `docker stop -t` trong `local-setup.md` nên kiểm riêng.

### Sửa lần 1

Agent sinh test case (execute-all) sửa theo review lần 1. Giữ nguyên ID `P0-T10a-TC01..TC17` (17 test case), trạng thái ⬜, 5 cột. Khối chuẩn bị và các lệnh đã sửa (TC03, TC04, TC05, TC07, TC15, TC17) đều qua `bash -n` và `zsh -n`. Helper `files`, `ociinfo`, `lastlayer` và bước copy/xoá của `snap` đã chạy thử trên `//tools/smoke`: đường dẫn tuyệt đối đúng, `layers=13`, và layer cuối của smoke hiện có `app/smoke.runfiles/...` (lỗi A4 được xác nhận).

- **Chặn 1, đường dẫn tương đối (TC04, TC05, TC07)**: `files()` giờ thêm tiền tố `$SRV/` vào output của `cquery`, nên cho ra đường dẫn tuyệt đối.
- **Chặn 2, host/arm64/amd64 dùng chung `bazel-out` (TC04, TC05, TC07)**: thêm helper `snap ARCH`. Helper này build đúng `--config=linux-ARCH` cho server, worker và hai image, copy (`cp -L`/`cp -RL`) sang `$S/ARCH/`, rồi `chmod -R u+w` vì output Bazel là read-only.
  - TC04 đọc `ociinfo` trên `$S/arm64/*_image`.
  - TC05 chạy `snap amd64` và `snap arm64` rồi so `file -L` trên hai bản copy riêng. Thêm `ociinfo $S/arm64/server_image` để kiểm hai bản copy không ghi đè nhau.
  - TC07 chạy `snap arm64` trước khi `stat`/`ociinfo`, nên đo đúng artifact arm64.
- **A4 và TC07**:
  - A4 ghi rõ lỗi đã biết của macro P0-T01a: `tar` đóng gói cả runfiles (bản sao thứ hai của binary). Task này phải tắt nó, ví dụ `include_runfiles = False` (tar.bzl 0.10.9), trong phạm vi `tools/rules/**`, và TC14 kiểm hồi quy smoke.
  - Tiêu chí TC07 đổi thành `layers=13` (base 12 + 1, so với `ociinfo` của smoke build cùng lúc).
  - TC07 thêm helper `lastlayer` và kiểm layer cuối chỉ có `app`, `app/<name>`, với `grep -c runfiles` = `0`.
- **TC03**: `(cd "$SRV" && golangci-lint run ./...)` và `(cd "$SRV" && go vet … && echo OK)` chạy trong subshell, nên không phụ thuộc cwd của lệnh trước.
- **TC15**: điều kiện "+ 1 s" chỉ grep `local-setup.md` (`grep -cE '\+ *1 *s'`), không gộp với `architecture.md`. Thêm lệnh kiểm riêng `docker stop -t N` trong `local-setup.md`, với N > `CORE_SHUTDOWN_TIMEOUT` của ví dụ + 1 (mặc định `10s` → N ≥ `12`).
- **TC17**: thêm `chmod -R u+w $S` trước `rm -rf $S`, vì bản copy read-only từ `snap` làm `rm` lỗi `Permission denied` (gặp khi chạy thử). Thêm kiểm scratchpad đã xoá (`0`).

### Review lần 2 — APPROVED

Reviewer độc lập (execute-all), lần 2. Đã tự kiểm chứng trên máy dev. Không sửa file repo; scratchpad, image và container tạm đã dọn (`rm` sạch, `0` image `com.tm.go.*`, `0` container `p0t10a-*`).

- **Chặn 1 (TC04, TC05, TC07) — đã xử lý.** Lưu khối Chuẩn bị ra scratchpad, `bash -n` và `zsh -n` đều OK, `source` chạy được trên cả `/bin/bash` 3.2.57 lẫn zsh. Sau `source`, cwd là `$REPO`, và `files --config=linux-arm64 //tools/smoke:smoke_image` in đường dẫn tuyệt đối `…/com/tm/server/bazel-out/darwin_arm64-fastbuild/bin/tools/smoke/smoke_image`.
- **Chặn 2 (TC05, TC07) — đã xử lý.** Chạy thử `snap` trên bản sao helper, trong đó thay target server/worker bằng `//tools/smoke:smoke` và `smoke_image`. Chạy `snap amd64` rồi `snap arm64`, cả hai exit `0` trên bash lẫn zsh:
  - `file -L $S/amd64/smoke` cho `ELF 64-bit LSB executable, x86-64 … statically linked`.
  - `file -L $S/arm64/smoke` cho `ARM aarch64 … statically linked`. Bản amd64 không bị lần build arm64 ghi đè.
  - `ociinfo` bản amd64 cho `linux/amd64 user=65532 entrypoint=["/app/smoke"] layers=13 layer_bytes=3753935`.
  - `ociinfo` bản arm64 cho `linux/arm64 … layers=13 layer_bytes=3868624`.
  - `lastlayer` liệt kê đúng các entry của layer binary: `app`, `app/smoke`, `app/smoke.runfiles/...` (7 dòng có `runfiles`). Kết quả này xác nhận lỗi A4.
  - `stat -f %z` cho `1573024`.
  - Bước dọn `chmod -R u+w` + `rm -rf` xoá sạch (`0`).
  - `nap`, `now` và mảng `ENV_NET` chạy đúng trên zsh.
- **Tính khả thi của A4 — xác nhận.** Trong `external/tar.bzl+/tar/tar.bzl` (0.10.9), macro `tar(name, mtree = "auto", mutate = None, include_runfiles = None, …)` mặc định `include_runfiles = True` khi `mtree == "auto"`. `com_tm_container.bzl` load `tar` từ `@tar.bzl` và dùng `mtree` mặc định (`auto`) cùng `mutate`, nên truyền `include_runfiles = False` là hợp lệ. Tiêu chí `layers=13` và `grep -c runfiles` = `0` ở TC07 vì vậy đạt được.
- **Góp ý lần 1 — đã xử lý.** Các góp ý cho TC03 (subshell cho lint), TC15 (chỉ grep `local-setup.md`, kiểm riêng `docker stop -t N`, N ≥ 12) và TC17 (`chmod -R u+w` trước `rm -rf`) đều đã sửa đúng.
- **Định dạng — đạt.** Bảng 5 cột, ID `P0-T10a-TC01..TC17` liên tục, trạng thái đều ⬜, có dòng "Test nghiệm thu liên quan". Không có lệnh ghi lịch sử git (`commit`/`add`/`stash`).

Góp ý nhỏ (không chặn):

- Các lệnh `grep -c` (TC07 `runfiles`, TC10, TC13…) trả exit `1` khi đếm được `0`. Đây là điều bình thường, người chạy cần đọc số in ra chứ không đọc exit code.
- TC05/TC07 gọi `snap` nhiều lần, mỗi lần đổi `--platforms` làm Bazel bỏ analysis cache (WARNING), nên chạy chậm hơn nhưng vẫn đúng.

## Ghi chú thực thi

Agent triển khai (execute-all), 2026-09-27, macOS arm64, Bazel 8.7.0, Docker Desktop 29.8. Mọi TC01..TC17 ✅.

- **Shell**: TC01–TC03 chạy trong zsh; TC04–TC17 chạy bằng `/bin/bash` (prep hỗ trợ bash 3.2). Lý do: trong zsh, `$SL:server_docker` bị hiểu là modifier `:s` của zsh → nhãn Bazel hỏng (`//ser_dockices/...`), `bazel run` exit 1. Không phải lỗi code; lệnh TC giữ nguyên.
- **Kích thước (TC07, arm64)**: server binary `20185248`, `layers=13 layer_bytes=20903376` (≤ binary + 5242880), Docker SIZE `46.6MB`; worker binary `2687136`, `layers=13 layer_bytes=3405264`, SIZE `11.6MB`. Layer cuối: chỉ `app`, `app/server` / `app/worker`, `grep -c runfiles` = `0`. Smoke build cùng lúc `layers=13 layer_bytes=2291152` (trước khi tắt runfiles: `3868624`).
- **amd64 (TC05)**: server binary `21467296`, `linux/amd64 user=65532 entrypoint=["/app/server"] layers=13 layer_bytes=22185423`; worker binary `2715808`, `linux/amd64 … ["/app/worker"] layers=13 layer_bytes=3433935`. `file`: 4 binary `statically linked`, x86-64 / ARM aarch64 đúng arch.
- **E2E**: TC08 access log `tc08-a` `status=200 latency_ms=0.016 route=/healthz`, `bad=0`. TC09 trace lên Tempo sau 1 s, span `GET /readyz` SERVER `svc=core ver=dev`, `pool.acquire` (parent = span server) → `connect`, `db=postgresql`; WARN/ERROR OTel `0`. TC10b `exited exit=1`, 1 dòng ERROR `CORE_DATABASE_URL`. TC11 `after=0.15s`, `exited exit=0 oom=false`, `signal http_stopped pool_closed done`, `telemetry_flush=ok`. TC12 worker `after=0.15s`, `exited exit=0`, log 3 dòng (khởi động, `signal`, `done`), deps pgx/net/http/redis `0`, cổng `0`.
- **Dọn (TC17)**: container `p0t10a-*` `0`, image `com.tm.go.core-*` `0` (cả smoke đã xoá), `dc down` (không `-v`, volume giữ), container project `0`, cổng 18080 `0`, scratchpad `0`.
