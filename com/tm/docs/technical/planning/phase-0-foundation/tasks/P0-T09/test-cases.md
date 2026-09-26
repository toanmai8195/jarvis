# Test cases — P0-T09: Graceful shutdown: bắt SIGTERM, `http.Server.Shutdown` có timeout, đóng pool sau cùng

> **Phạm vi**: core dừng an toàn khi nhận SIGTERM/SIGINT. Ngừng nhận kết nối mới, chờ request đang xử lý xong trong một khoảng timeout, hết hạn thì đóng cưỡng bức, đóng pool PG **sau** khi HTTP server đã dừng, log JSON từng mốc, exit code theo quy ước. Căn cứ:
> - Phase 0 README: dòng P0-T09 "bắt SIGTERM, `http.Server.Shutdown` có timeout, đóng pool sau cùng". **P0-NFR3** "Service dừng an toàn khi nhận SIGTERM, không cắt ngang request". Challenge **G3** "Bắt SIGTERM, ngừng nhận request, chờ in-flight, đóng pool theo thứ tự". Hoàn thành khi "Rolling deploy dưới tải không mất/không lỗi request". DoD "Gửi SIGTERM khi đang có request chậm → request hoàn thành, service thoát sạch".
> - [acceptance-tests](../../acceptance-tests.md): **P0-AT08** (handler sleep 3s, SIGTERM sau 1s → 200, exit 0), **P0-AT09** (SIGTERM rồi gửi request mới → connection refused / 503), **P0-AT10** (handler chạy quá shutdown timeout → vẫn thoát sau timeout, log cảnh báo).
> - [execute-all-notes](../../../execute-all-notes.md) mục P0-T07: "`defer pool.Close()` không chạy khi process bị SIGTERM. Đúng phạm vi P0-T09".
> - Code hiện có: `services/core/cmd/server/main.go` gọi `srv.ListenAndServe()` trực tiếp, chưa bắt tín hiệu. `http.Server` có `ReadHeaderTimeout: 5s`. `httpx.ReadyTimeout` = 2s. `httpx.withRoutes` (không export) cho test gắn route.
> - [project-structure](../../../../project-structure.md): wiring tay trong `main.go`, interface ở phía dùng, quy tắc 3 tầng (chỉ đưa vào `pkg/` khi service thứ hai cần).
> - Docs **không** nói về: khoảng chờ drain cho load balancer, `/readyz` trả 503 khi đang dừng, giá trị timeout, tín hiệu thứ hai, exit code khi hết timeout. Các điểm này là giả định bên dưới.
>
> **Giả định** (reviewer xem kỹ):
> - A1. **Vị trí**. Logic shutdown nằm trong `services/core/` — `cmd/server/` (ví dụ hàm `run` tách khỏi `main` để test được) hoặc một package dưới `services/core/internal/`. **Không** đặt vào `pkg/`: hiện chỉ core có HTTP server Go, stats-worker chưa có (quy tắc 3 tầng). Không DI framework, không `IFoo`/`FooImpl`. Pool được truyền vào qua interface nhỏ ở phía dùng (ví dụ `interface{ Close() }`) để unit test dùng fake.
> - A2. **Tín hiệu**: `SIGTERM` và `SIGINT` (Ctrl+C khi chạy local) đều kích hoạt shutdown, qua `signal.NotifyContext` hoặc `signal.Notify`.
> - A3. **Timeout**: biến môi trường mới `CORE_SHUTDOWN_TIMEOUT`, định dạng `time.ParseDuration` (`10s`, `1500ms`), **mặc định `10s`**. Rỗng/chỉ khoảng trắng → mặc định. Không parse được, thiếu đơn vị (`10`), `0` hoặc âm → lỗi cấu hình nêu tên biến, core thoát `1` trước khi mở cổng (cùng kiểu với `config.Load` hiện có). Khi chạy trong container, `stop_grace_period`/`terminationGracePeriodSeconds` phải lớn hơn giá trị này (ghi vào handbook, không cấu hình ở task này).
> - A4. **Trình tự** khi nhận tín hiệu:
>   1. log mốc `signal`;
>   2. `srv.Shutdown(ctx có timeout)`: đóng listener ngay (kết nối mới bị từ chối), đóng kết nối keep-alive rảnh, chờ request đang chạy;
>   3. hết timeout → log WARN mốc `timeout`, gọi `srv.Close()` để cắt kết nối còn lại, **không** chờ handler;
>   4. đóng pool PG (`pool.Close()`) có giới hạn thời gian theo A12, log mốc `pool_closed` (xong trong hạn) hoặc `pool_close_timeout` (quá hạn) — luôn **sau** bước 2/3;
>   5. log mốc `done` rồi thoát.
>
>   Không bỏ bước 4 ở nhánh lỗi. Không dùng `defer` + `os.Exit` (defer không chạy).
> - A5. **Không có khoảng chờ drain, không đổi `/readyz`**. Docs không có load balancer ở máy dev và không yêu cầu. Listener đóng ngay khi bắt đầu shutdown, nên request mới nhận connection refused (khớp P0-AT09). Nếu phase triển khai có load balancer, việc cho `/readyz` trả 503 và chờ trước `Shutdown` thuộc phase đó.
> - A6. **Log mốc shutdown**: mỗi mốc là một dòng log JSON slog có key `shutdown_step` với giá trị lần lượt `signal`, `http_stopped` (Shutdown xong trong hạn) hoặc `timeout` (hết hạn), `pool_closed` (đóng pool xong trong hạn) hoặc `pool_close_timeout` (quá hạn, A12), `done`. Dòng `signal` có key `signal` = `os.Signal.String()` (`terminated` cho SIGTERM, `interrupt` cho SIGINT) và key `timeout` (giá trị cấu hình). Dòng `timeout` và `pool_close_timeout` mức `WARN`, các dòng khác mức `INFO`. Dòng `done` có key `exit_code`. `msg` do người làm chọn.
> - A7. **Exit code**: `0` khi shutdown sạch (Shutdown xong trong hạn, kể cả khi có request đang chạy). `1` khi hết timeout phải đóng cưỡng bức, khi đóng pool quá hạn (`pool_close_timeout`, A12), khi lỗi listen (cổng bận), khi lỗi cấu hình. P0-AT10 chỉ đòi "vẫn thoát sau timeout, log cảnh báo". Chọn `1` để orchestrator phân biệt được lần dừng không sạch.
> - A8. **Tín hiệu thứ hai** trong lúc đang shutdown: thoát ngay, không chờ hết timeout, exit code ≠ `0` (`1`, hoặc 143/130 nếu để handler mặc định của Go giết process sau `stop()` của `NotifyContext`). Docs không nói; đây là hành vi phổ biến để người vận hành bấm Ctrl+C lần hai.
> - A9. **Lỗi listen** (cổng bận): thoát `1` ngay, không chờ tín hiệu, log `ERROR` có lỗi `address already in use`, pool vẫn được đóng.
> - A10. **"Request đang chạy" ở E2E**: binary thật không có route chậm và không được thêm route debug.
>   - Bằng chứng chính cho P0-AT08 ("request đang chạy được phục vụ xong") là **TC18**: Go test gắn route ngủ 3 s, trigger shutdown sau 1 s, dùng cùng luồng shutdown với `main`. Trên binary thật là **TC11**: `/readyz` khi PG bị `docker pause`, handler chạy thật tới `ReadyTimeout` 2 s rồi trả 503 đầy đủ.
>   - **Client chậm** (`slowreq`: mở TCP, gửi nửa header, chờ rồi gửi nốt) **không** chứng minh được request được phục vụ. `net/http` (Go 1.27.1, `conn.serve`) có kiểm tra `shuttingDown()` sau `readRequest`: request đọc xong khi server đang shutdown bị đóng kết nối, **không** trả response. `Shutdown` vẫn chờ kết nối `StateNew` chưa quá 5 s. Vì vậy `slowreq` chỉ dùng để **giữ process sống** trong lúc shutdown (TC09, TC10, TC12, TC14, TC15). Kết quả mong đợi phía client là `EOF` hoặc kết nối bị đóng/reset, không phải `200`. Giữ thời gian chờ < `ReadHeaderTimeout` 5 s.
> - A11. **G3 ở máy dev**: TC17 đo phần đo được. Tải nhẹ, mỗi request một kết nối mới, SIGTERM giữa chừng. Mọi request server đã xử lý (có dòng access log) đều về client với `200`. Phần còn lại phần lớn là connection refused (`000 7`). Một số rất nhỏ `000 52` (empty reply) / `000 56` (reset) được chấp nhận (≤ 2/1500). Đó là kết nối đã vào backlog hoặc đã `Accept` nhưng request đọc xong sau `shuttingDown()` (A10), server không tránh được nếu không có drain + load balancer (A5). Đây là lý do G3 cần drain/LB ở phase triển khai, ghi vào handbook. Không có 5xx, không timeout. Rolling deploy thật (nhiều instance, load balancer) chưa có ở phase 0, nên G3 chỉ chuyển 🟨.
> - A12. **Đóng pool có giới hạn thời gian**. `pgxpool.Pool.Close()` có thể treo tới ~15 s khi một kết nối vừa bị huỷ giữa chừng (ví dụ ping `/readyz` hết hạn lúc PG bị `docker pause`: pgxpool chờ việc huỷ kết nối bất đồng bộ; reviewer đo được 15.0 s). Vì vậy `Close()` chạy trong goroutine, hàm shutdown chờ tối đa **phần còn lại** của `CORE_SHUTDOWN_TIMEOUT` tính từ lúc nhận tín hiệu, nhưng **ít nhất 1 s** (hằng không export, ví dụ `minPoolCloseWait = 1 * time.Second`, để pool vẫn có thời gian đóng khi HTTP đã dùng hết hạn ở A4 bước 3). Quá hạn thì log `WARN` mốc `pool_close_timeout`, không chờ tiếp, sang `done` và thoát `1` (A7). Goroutine `Close()` bị bỏ lại chết theo process. Tổng thời gian từ tín hiệu tới khi thoát ≤ `CORE_SHUTDOWN_TIMEOUT` + 1 s (+ sai số poll của `Shutdown` ~0.5 s). Không thêm biến môi trường mới.
>
> **Không thuộc task này**: `TracerProvider`/`MeterProvider` và việc flush/`Shutdown` provider (P0-T10 tạo provider, P0-T10 thêm bước shutdown provider vào trình tự A4). Image OCI, `stop_grace_period` trong compose (P0-T10a/phase triển khai). Graceful shutdown của BFF (P0-T11, `server.ts`). `cmd/worker`, stats-worker (task sau). Khoảng chờ drain / `/readyz` 503 cho load balancer (A5). Route debug/sleep trong code chạy thật.
>
> **Môi trường khi viết**: macOS arm64, Go 1.27.1, Bazel 8.7.0 (Bazelisk), golangci-lint v2.14.0, Docker Desktop, stack P0-T02 (`postgres-core` 5432), curl, jq, python3, lsof.
>
> **Chuẩn bị**: lưu khối dưới thành `<scratchpad>/p0t09/prep.sh` một lần, tạo bằng Write tool, thay `<scratchpad>` bằng đường dẫn thật. Mỗi lần gọi Bash bắt đầu bằng `source <scratchpad>/p0t09/prep.sh`.
> - Khối này chạy được trên cả bash 3.2 lẫn zsh (không dùng biến tên `status`/`path`). Compose gọi qua hàm `dc`.
> - Phần chuẩn bị và các TC **không** có lệnh ghi lịch sử git.
> - Mỗi TC E2E (TC06–TC17, TC19) chạy trong **một** lần gọi Bash, từ `corerun` tới `waitexit`.
> - Trước TC06: `lsof -nP -iTCP:8080 -sTCP:LISTEN` phải rỗng. `dc up -d --wait postgres-core` đã chạy.
> - `corerun` chạy binary trong subshell nền và ghi exit code thật vào `$S/core.exit` (`wait` trên PID của core). `sig` gửi tín hiệu và ghi mốc `T0`; `waitexit N` chờ tối đa N giây (số nguyên), in exit code và thời gian từ `T0`.
> - Nếu harness chặn `sleep` thì thay thân `nap` bằng `python3 -c 'import time,sys;time.sleep(float(sys.argv[1]))' "$1"`.
> ```bash
> REPO=/Users/maiductoan/Documents/code/jarvis; cd $REPO
> SRV=$REPO/com/tm/server; CORE=$SRV/services/core
> S=<scratchpad>/p0t09; mkdir -p $S
> dc() { docker compose -f "$REPO/deploy/docker-compose.yml" "$@"; }
> DSN='postgres://snaptix:snaptix@localhost:5432/core?sslmode=disable'
> C=http://localhost:8080
> BIN=$S/core
> nap() { sleep "$1"; }
> now()   { python3 -c 'import time;print("%.3f"%time.time())'; }
> since() { python3 -c "import time;print('%.2f'%(time.time()-$1))"; }
> corebuild() { (cd $SRV && go build -o $BIN ./services/core/cmd/server); }
> # corerun VAR=val... — chạy core nền, PID ở core.pid, exit code thật ở core.exit khi process thoát
> corerun() { corestop; rm -f "$S/core.exit" "$S/core.pid"
>   ( env -i PATH="$PATH" HOME="$HOME" "$@" "$BIN" >"$S/core.log" 2>&1 & echo $! >"$S/core.pid"; wait $!; echo $? >"$S/core.exit" ) &
>   local i; for i in $(seq 40); do [ -s "$S/core.pid" ] && break; nap 0.05; done; }
> corestop() { [ -f $S/core.pid ] || return 0; local p i; p=$(cat $S/core.pid); kill -9 "$p" 2>/dev/null
>              for i in $(seq 40); do kill -0 "$p" 2>/dev/null || break; nap 0.05; done; rm -f $S/core.pid; true; }
> alive()    { kill -0 "$(cat $S/core.pid)" 2>/dev/null && echo ALIVE || echo DEAD; }
> sig()      { T0=$(now); kill -"$1" "$(cat $S/core.pid)"; }
> waitexit() { local i; for i in $(seq $(( $1 * 20 ))); do [ -s "$S/core.exit" ] && break; nap 0.05; done
>              if [ -s "$S/core.exit" ]; then echo "exit=$(cat $S/core.exit) after=$(since $T0)s"; else echo "NOT_EXITED after ${1}s"; fi; }
> code()     { curl -s -m 10 -o /dev/null -w '%{http_code}\n' "$@"; }
> # newconn URL — 1 request kết nối mới, in "<http_code> rc=<curl exit>" (000 rc=7 = connection refused)
> newconn()  { local c; c=$(curl -s -m 3 -o /dev/null -w '%{http_code}' "$1"); echo "$c rc=$?"; }
> waitcode() { local s=$(date +%s) c; while :; do c=$(curl -s -m 5 -o /dev/null -w '%{http_code}' "$1")
>              [ "$c" = "$2" ] && { echo "$c after $(( $(date +%s)-s ))s"; return 0; }
>              [ $(( $(date +%s)-s )) -ge "$3" ] && { echo "$c TIMEOUT"; return 1; }; nap 0.2; done; }
> # steps — chuỗi shutdown_step theo thứ tự trong log
> steps()    { jq -rR 'fromjson? | select(type=="object" and has("shutdown_step")) | .shutdown_step' "$S/core.log" | tr '\n' ' '; echo; }
> # stepline NAME — dòng log của mốc NAME
> stepline() { jq -cR --arg s "$1" 'fromjson? | select(type=="object" and .shutdown_step==$s)' "$S/core.log"; }
> # slowreq HOLD OUT — client chậm (A10), CHỈ để giữ process sống lúc shutdown: gửi nửa header GET /healthz, chờ HOLD giây, gửi nốt; ghi "<status line hoặc EOF/ERR:...>|<giây>" vào OUT (khi đang shutdown: mong đợi EOF, không phải 200)
> slowreq() { python3 -c '
> import socket,sys,time
> h=float(sys.argv[1]); t=time.time()
> try:
>     s=socket.create_connection(("127.0.0.1",8080),timeout=h+15)
>     s.sendall(b"GET /healthz HTTP/1.1\r\nHost: localhost\r\n"); time.sleep(h)
>     s.sendall(b"User-Agent: slowreq\r\n\r\n"); d=b""
>     while True:
>         c=s.recv(4096)
>         if not c: break
>         d+=c
>     r=d.split(b"\r\n",1)[0].decode() if d else "EOF"
> except Exception as e: r="ERR:"+type(e).__name__
> print("%s|%.2f"%(r,time.time()-t))' "$1" >"$2" 2>&1 & }
> # idleconn HOLD OUT — 1 request keep-alive rồi giữ kết nối rảnh HOLD giây; ghi "<status line>|CLOSED|RESET|STILL_OPEN|<giây>"
> idleconn() { python3 -c '
> import socket,sys,time
> h=float(sys.argv[1]); t=time.time()
> s=socket.create_connection(("127.0.0.1",8080),timeout=3)
> s.sendall(b"GET /healthz HTTP/1.1\r\nHost: localhost\r\n\r\n"); first=s.recv(4096).split(b"\r\n",1)[0].decode()
> s.settimeout(h)
> try: r="CLOSED" if s.recv(1)==b"" else "DATA"
> except socket.timeout: r="STILL_OPEN"
> except ConnectionResetError: r="RESET"
> print("%s|%s|%.2f"%(first,r,time.time()-t))' "$1" >"$2" 2>&1 & }
> # loadgen N P OUT — N request /healthz, P luồng, mỗi request 1 kết nối mới; mỗi dòng "<http_code> <curl exit>"
> loadgen() { seq "$1" | xargs -P "$2" -I{} sh -c 'c=$(curl -s -m 5 -o /dev/null -w "%{http_code}" http://localhost:8080/healthz); echo "$c $?"' >"$3" 2>&1; }
> jsonlines() { python3 -c 'import json,sys
> n=bad=0
> for l in open(sys.argv[1]):
>     l=l.strip()
>     if not l: continue
>     n+=1
>     try: o=json.loads(l); assert isinstance(o,dict) and {"time","level","msg"}<=o.keys()
>     except Exception: bad+=1; print("BAD:",l[:200])
> print(f"lines={n} bad={bad}")' "$1"; }
> ```

| ID | Loại | Kịch bản | Kết quả mong đợi | Trạng thái |
|---|---|---|---|---|
| P0-T09-TC01 | Cấu trúc | Vị trí và quy ước Go (A1): `git -C $REPO status --porcelain --untracked-files=all com/tm/server`; `grep -rlnE 'signal\.Notify(Context)?\(' $SRV --include='*.go' \| grep -v _test.go`; `grep -rlnE '\.Shutdown\(' $SRV --include='*.go' \| grep -v _test.go`; `ls $SRV/pkg/otelx 2>&1`; `grep -rnE 'type I[A-Z][A-Za-z]* interface\|[A-Za-z]Impl\b\|go.uber.org/(fx\|dig)\|google/wire' $CORE`; `grep -rnE 'defer .*\.Close\(\)' $CORE/cmd/server/main.go`; `bash scripts/check-structure.sh; echo $?` | File thay đổi/mới chỉ trong `services/core/**` (và `go.mod`/`go.sum`/`MODULE.bazel*` nếu thêm thư viện test). `signal.Notify*` và `Shutdown(` chỉ xuất hiện trong file dưới `services/core/` (không trong `pkg/`). `pkg/otelx` không tồn tại. Lệnh grep `IFoo`/`Impl`/DI rỗng. Nếu còn `defer pool.Close()` trong `main.go` thì `main` không gọi `os.Exit` sau nó ở nhánh thường (pool đóng tường minh theo A4). `check-structure.sh` exit `0` | ✅ |
| P0-T09-TC02 | Build | Build bằng `go` (G14): `cd $SRV && go vet ./... && go build ./... && echo OK`; `go mod tidy -diff; echo $?`; `corebuild; echo $?`; `gofmt -l $CORE`; `git -C $REPO status --porcelain --untracked-files=all com/tm/server \| grep -vE '\.go$\|BUILD\.bazel$\|go\.(mod\|sum)$\|MODULE\.bazel(\.lock)?$\|\.env\.example$'` | In `OK`. `go mod tidy -diff` exit `0`. `corebuild` exit `0`. `gofmt -l` rỗng. Lệnh `git status` cuối không in dòng nào (không có binary/file rác) | ✅ |
| P0-T09-TC03 | Build | Bazel và Gazelle (G14): `cd $SRV && bazel run //:gazelle -- -mode=diff; echo $?`; `bazel build //...; echo $?`; `bazel test //services/core/...; echo $?`; `bazel query 'kind(go_test, //services/core/...)'` | Gazelle `-mode=diff` exit `0`. `bazel build //...` và `bazel test //services/core/...` exit `0`. Có `go_test` cho package chứa logic shutdown (ví dụ `//services/core/cmd/server:server_test`) và `//services/core/internal/config:config_test` | ✅ |
| P0-T09-TC04 | Lint | `cd $SRV && golangci-lint --version && golangci-lint run ./...; echo $?`; `grep -rn 'nolint' $CORE` | Phiên bản `2.14.0`, `0 issues`, exit `0`. Không có `//nolint`, hoặc mỗi cái nêu linter kèm lý do | ✅ |
| P0-T09-TC05 | Unit | Unit test table-driven, `-race`, không cần Docker, ổn định khi chạy lặp: `dc stop postgres-core`; `cd $SRV && go test -race -count=1 ./...; echo $?`; `go test -race -count=5 ./services/core/...; echo $?`; `go test -race -count=1 -v ./services/core/... 2>&1 \| grep -E '^\s*--- (PASS\|FAIL\|SKIP)'`; `grep -rnE 'range tests\|range tt\|range cases\|t\.Run\(' $CORE/cmd $CORE/internal/config --include='*_test.go' \| wc -l \| tr -d ' '`; `grep -rn 't\.Skip' $CORE --include='*_test.go'`; `dc start postgres-core` | Hai lệnh `go test` exit `0` khi PG dừng; `-count=5` không flaky. Có subtest `--- PASS` phủ đủ các nhóm (fake closer thay pool, trigger bằng huỷ `ctx` hoặc tín hiệu thật): **U1** không có request → trả `nil`, closer gọi đúng 1 lần; **U2** request đang chạy (handler chậm) được trả `200` trước khi hàm trả về, trả `nil`; **U3** sau khi bắt đầu shutdown, dial kết nối mới bị từ chối; **U4** handler chậm hơn timeout → hàm trả lỗi (`errors.Is(err, context.DeadlineExceeded)` hoặc lỗi bọc tương đương) trong < timeout + 1 s, **không** chờ handler xong, closer vẫn được gọi; **U5** cổng bận → trả lỗi ngay (< 1 s) không cần tín hiệu, closer vẫn được gọi; **U6** closer gọi **sau** khi `Shutdown`/`Close` của server trả về và sau khi handler ở U2 xong (fake ghi thứ tự sự kiện); **U7** không rò goroutine (`goleak.VerifyNone` hoặc so `runtime.NumGoroutine()` trước/sau có chờ ổn định); **U8** bảng `CORE_SHUTDOWN_TIMEOUT` (TC06); **U9** exit code: sạch → `0`, timeout/đóng pool quá hạn/listen lỗi → `1` (A7) nếu có hàm ánh xạ; **U10** đóng pool có hạn (A12): fake closer treo (chặn trên channel, test mở khoá ở `t.Cleanup` để U7 không báo rò) → hàm trả về trong ≤ phần hạn còn lại + 0.5 s, với trường hợp HTTP đã hết hạn thì ≤ 1 s (mức tối thiểu) + 0.5 s; trả lỗi nhận diện được (sentinel hoặc bọc tương đương) ↔ exit `1`; log mốc `pool_close_timeout` mức `WARN` thay cho `pool_closed`, vẫn có `done`. Fake closer trả ngay → `pool_closed`, không có `pool_close_timeout`. Không `FAIL`/`SKIP`, không `t.Skip`. Có test dạng bảng. Test cũ P0-T07/P0-T08 vẫn pass | ✅ |
| P0-T09-TC06 | Unit + E2E | Cấu hình `CORE_SHUTDOWN_TIMEOUT` (A3): `cd $SRV && go test -race -count=1 -v -run 'Shutdown\|Load' ./services/core/internal/config/ 2>&1 \| grep -E -- '--- (PASS\|FAIL)'`; `corebuild`; `for v in abc 10 0 -1s; do corerun CORE_DATABASE_URL="$DSN" CORE_SHUTDOWN_TIMEOUT=$v; T0=$(now); r=$(waitexit 3); echo "$v: $r"; grep -c 'CORE_SHUTDOWN_TIMEOUT' $S/core.log; lsof -nP -iTCP:8080 -sTCP:LISTEN \| wc -l \| tr -d ' '; done`; `corerun CORE_DATABASE_URL="$DSN" CORE_SHUTDOWN_TIMEOUT=' '`; `waitcode $C/healthz 200 10`; `sig TERM; waitexit 3`; `stepline signal \| jq -r .timeout` | Unit test có subtest cho: không đặt → `10s`; rỗng/khoảng trắng → `10s`; `30s`, `1500ms` → đúng giá trị; `abc`, `10`, `0`, `-1s` → lỗi, thông báo bắt đầu bằng `CORE_SHUTDOWN_TIMEOUT`; lỗi này gom chung với lỗi biến khác (`errors.Join`). E2E: cả 4 giá trị sai → `exit=1` trong < 3 s, log có `CORE_SHUTDOWN_TIMEOUT` (≥ `1`), cổng 8080 không mở (`0`). Giá trị `' '` → chạy bình thường, thoát `exit=0`, dòng `signal` có `timeout` là `10s` | ✅ |
| P0-T09-TC07 | E2E | SIGTERM khi rảnh (A2, A4, A6, A7): `corerun CORE_DATABASE_URL="$DSN"`; `waitcode $C/healthz 200 10`; `code $C/readyz`; `sig TERM; waitexit 5`; `steps`; `stepline signal \| jq -c '{level,signal,timeout}'`; `stepline done \| jq -c '{level,exit_code}'`; `lsof -nP -iTCP:8080 -sTCP:LISTEN \| wc -l \| tr -d ' '`; `pgrep -f "$BIN" \| wc -l \| tr -d ' '` | `exit=0`, `after` < `1.0`s. `steps` in đúng `signal http_stopped pool_closed done`. Dòng `signal`: `INFO`, `signal` = `terminated`, `timeout` = `10s`. Dòng `done`: `INFO`, `exit_code` = `0`. Cổng 8080 đã giải phóng (`0`), không còn process (`0`) | ✅ |
| P0-T09-TC08 | E2E | SIGINT cũng kích hoạt shutdown (A2): `corerun CORE_DATABASE_URL="$DSN"`; `waitcode $C/healthz 200 10`; `sig INT; waitexit 5`; `steps`; `stepline signal \| jq -r .signal` | `exit=0`, `after` < `1.0`s (process nền của shell không interactive vẫn nhận SIGINT vì Go cài lại handler). `steps` = `signal http_stopped pool_closed done`. `signal` = `interrupt` | ✅ |
| P0-T09-TC09 | E2E | Kết nối đang mở giữ shutdown chờ nhưng không kéo tới timeout; request đọc xong sau khi bắt đầu shutdown bị đóng không trả response (hành vi `net/http`, A10): `corerun CORE_DATABASE_URL="$DSN"`; `waitcode $C/healthz 200 10`; `slowreq 3 $S/slow.txt; nap 1`; `sig TERM`; `nap 0.3; alive`; `waitexit 6`; `cat $S/slow.txt`; `steps`; `jq -rR 'fromjson? \| select(type=="object" and (has("shutdown_step") or has("latency_ms"))) \| (.shutdown_step // "access")' $S/core.log \| tr '\n' ' '` | Ngay sau SIGTERM process vẫn `ALIVE` (`Shutdown` chờ kết nối `StateNew`). `slow.txt` = `EOF\|<≈3.0>`: server đóng kết nối, không trả response (không phải `HTTP/1.1 200`, không `ERR:timeout`). `exit=0`, `after` ≥ `1.5`s (chờ kết nối) và < `3.0`s (thoát ngay khi kết nối đóng, không chờ tới hết timeout 10s). `steps` = `signal http_stopped pool_closed done`. Chuỗi cuối: sau `signal` không có `access` nào (request của `slowreq` không vào handler) | ✅ |
| P0-T09-TC10 | E2E | Kết nối mới bị từ chối ngay khi bắt đầu shutdown (P0-AT09, A5): `corerun CORE_DATABASE_URL="$DSN"`; `waitcode $C/healthz 200 10`; `slowreq 3 $S/slow.txt; nap 0.5`; `sig TERM; nap 0.3`; `alive`; `newconn $C/healthz`; `newconn $C/readyz`; `lsof -nP -iTCP:8080 -sTCP:LISTEN \| wc -l \| tr -d ' '`; `waitexit 6`; `cat $S/slow.txt` | Trong lúc process còn `ALIVE` (`slowreq` giữ process sống): cả hai request mới in `000 rc=7` (connection refused), không có `200`; không còn socket LISTEN trên 8080 (`0`). Sau đó `exit=0`; `slow.txt` bắt đầu bằng `EOF` (kết nối bị đóng, không phải `HTTP/1.1 200`, A10) | ✅ |
| P0-T09-TC11 | E2E | Handler thật đang chạy khi SIGTERM được phục vụ xong, đóng pool có hạn (P0-AT08 trên binary thật, A10, A12): `corerun CORE_DATABASE_URL="$DSN" CORE_SHUTDOWN_TIMEOUT=4s`; `waitcode $C/readyz 200 15`; `dc pause postgres-core`; `(curl -s -m 10 -o /dev/null -w '%{http_code} %{time_total}' $C/readyz; echo " rc=$?") >$S/ready.txt 2>&1 &`; `nap 0.5; sig TERM`; `nap 0.3; alive`; `waitexit 8`; `cat $S/ready.txt`; `steps`; `stepline pool_close_timeout \| jq -c '{level}'`; `stepline done \| jq -r .exit_code`; `dc unpause postgres-core`; `dc up -d --wait postgres-core` | Ngay sau SIGTERM process `ALIVE`. `ready.txt` = `503 <≈2.0> rc=0`: handler `/readyz` chạy tới `ReadyTimeout` rồi trả 503 đầy đủ, client không bị cắt (`rc=0`, không phải `000`). `after` ≤ `5.0`s (= timeout 4s + 1s, A12), không treo ~15s theo `pool.Close()`. Kết quả mong đợi (pool treo vì ping vừa bị huỷ, đã được reviewer kiểm chứng): `steps` = `signal http_stopped pool_close_timeout done`, dòng `pool_close_timeout` mức `WARN`, `done.exit_code` = `1`, `exit=1`, `after` ≥ `3.5`s. Nếu lần chạy đó `pool.Close()` xong trong hạn thì chấp nhận `steps` = `signal http_stopped pool_closed done` và `exit=0`, nhưng mốc log và exit code phải khớp nhau như trên | ✅ |
| P0-T09-TC12 | E2E | Hết timeout → đóng cưỡng bức, log cảnh báo (P0-AT10, A4, A7): `corerun CORE_DATABASE_URL="$DSN" CORE_SHUTDOWN_TIMEOUT=1s`; `waitcode $C/healthz 200 10`; `slowreq 4 $S/slow.txt; nap 0.5`; `sig TERM; waitexit 5`; `steps`; `stepline timeout \| jq -c '{level}'`; `stepline done \| jq -r .exit_code`; `nap 4; cat $S/slow.txt`; `lsof -nP -iTCP:8080 \| wc -l \| tr -d ' '` | `exit=1`, `after` trong [`0.9`, `2.0`]s — thoát sau timeout 1s, không chờ client 4s. `steps` = `signal timeout pool_closed done` (không có `http_stopped`). Dòng `timeout` mức `WARN`. `done.exit_code` = `1`. `slow.txt` **không** phải `HTTP/1.1 200` (là `EOF`, `ERR:ConnectionResetError` hoặc `ERR:BrokenPipeError`). Không còn socket nào trên 8080 (`0`) | ✅ |
| P0-T09-TC13 | E2E | Kết nối keep-alive rảnh không giữ process (A4 bước 2): `corerun CORE_DATABASE_URL="$DSN"`; `waitcode $C/healthz 200 10`; `idleconn 8 $S/idle.txt; nap 0.5`; `sig TERM; waitexit 5`; `nap 0.5; cat $S/idle.txt`; `steps` | `exit=0`, `after` < `1.0`s (không chờ tới timeout 10s vì kết nối rảnh). `idle.txt` = `HTTP/1.1 200 OK\|CLOSED\|<< 2>`: server chủ động đóng kết nối rảnh. `steps` = `signal http_stopped pool_closed done` | ✅ |
| P0-T09-TC14 | E2E | Pool đóng **sau** HTTP server, cả khi pool đang có kết nối (A4): `corerun CORE_DATABASE_URL="$DSN" CORE_LOG_LEVEL=debug`; `waitcode $C/readyz 200 15`; `for i in $(seq 5); do code $C/readyz >/dev/null; done`; `slowreq 2 $S/slow.txt; nap 0.5`; `sig TERM; waitexit 5`; `cat $S/slow.txt`; `jq -rR 'fromjson? \| select(type=="object" and (has("shutdown_step") or has("latency_ms"))) \| (.shutdown_step // ("access " + (.status\|tostring)))' $S/core.log \| tail -5`; `python3 -c 'import json,sys,re,datetime as d; P=lambda s:d.datetime.fromisoformat(re.sub(r"\.(\d+)",lambda m:"."+(m.group(1)+"000000")[:6],s).replace("Z","+00:00")); L=[json.loads(l) for l in open(sys.argv[1]) if l.startswith("{")]; t={o["shutdown_step"]:P(o["time"]) for o in L if "shutdown_step" in o}; print(t["signal"]<=t["http_stopped"]<=t["pool_closed"]<=t["done"], round((t["http_stopped"]-t["signal"]).total_seconds(),2))' $S/core.log` | `exit=0`. 5 dòng cuối đúng thứ tự `access 200`, `signal`, `http_stopped`, `pool_closed`, `done` — access log cuối cùng (của `/readyz`) đứng **trước** `signal`, không có `access` sau `signal` (`slowreq` chỉ giữ `Shutdown` chờ, A10; `slow.txt` bắt đầu bằng `EOF`). Lệnh python in `True <x>` với `x` ≥ `1.0`: `signal` ≤ `http_stopped` ≤ `pool_closed` ≤ `done`, và `Shutdown` thực sự chờ kết nối (≥ 1 s) trước khi pool bị đóng. Unit test U6 (TC05) kiểm cùng thứ tự bằng fake với request đang chạy | ✅ |
| P0-T09-TC15 | E2E | Tín hiệu thứ hai trong lúc shutdown (A8): `corerun CORE_DATABASE_URL="$DSN"`; `waitcode $C/healthz 200 10`; `slowreq 4 $S/slow.txt; nap 0.5`; `sig TERM; nap 0.5; alive`; `sig TERM; waitexit 3`; `nap 4; cat $S/slow.txt` | Sau tín hiệu 1 process vẫn `ALIVE` (đang chờ, timeout 10s). Sau tín hiệu 2: thoát, `after` < `1.0`s tính từ tín hiệu 2, `exit` ≠ `0` (`1`, `143`). `slow.txt` không phải `HTTP/1.1 200` | ✅ |
| P0-T09-TC16 | E2E | Lỗi listen không treo (A9): chiếm cổng bằng socket IPv6 dual-stack (trên macOS, socket IPv4 bind `0.0.0.0` **không** chặn Go `Listen(":8080")` vốn bind `[::]` dual-stack): `python3 -c 'import socket,time;s=socket.socket(socket.AF_INET6);s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);s.setsockopt(socket.IPPROTO_IPV6,socket.IPV6_V6ONLY,0);s.bind(("::",8080));s.listen();time.sleep(8)' & BP=$!; nap 0.5` (`SO_REUSEADDR` để bind được khi 8080 còn `TIME_WAIT` từ các TC trước); `lsof -nP -iTCP:8080 -sTCP:LISTEN \| grep -ci python`; `corerun CORE_DATABASE_URL="$DSN"`; `T0=$(now); waitexit 3`; `jq -cR 'fromjson? \| select(type=="object" and .level=="ERROR") \| .err' $S/core.log`; `steps`; `kill $BP` | `lsof` thấy python đang LISTEN (`1`). `exit=1` (≠ `0`), `after` < `1.0`s — không chờ tín hiệu. Có dòng `ERROR` mà `err` chứa `address already in use`. Nếu có `steps` thì có `pool_closed` và `done` với `exit_code` `1`; không có `http_stopped` | ✅ |
| P0-T09-TC17 | E2E | Tải nhẹ + SIGTERM: mọi request server đã xử lý đều về client 200 (G3 ở máy dev, A11): `acc() { jq -cR 'fromjson? \| select(type=="object" and has("latency_ms"))' $S/core.log; }`; `corerun CORE_DATABASE_URL="$DSN"`; `waitcode $C/healthz 200 10`; `A0=$(acc \| wc -l \| tr -d ' ')`; `(loadgen 1500 8 $S/load.txt) & LP=$!; nap 1`; `sig TERM; waitexit 5`; `wait $LP 2>/dev/null; while kill -0 $LP 2>/dev/null; do nap 0.2; done`; `wc -l < $S/load.txt \| tr -d ' '`; `sort $S/load.txt \| uniq -c`; `grep -vcE '^(200 0\|000 7\|000 52\|000 56)$' $S/load.txt`; `grep -cE '^000 (52\|56)$' $S/load.txt`; `A1=$(acc \| wc -l \| tr -d ' '); echo "handled=$((A1-A0)) ok=$(grep -c '^200 0$' $S/load.txt)"`; `acc \| jq -r .status \| sort \| uniq -c` | `exit=0`. `load.txt` đủ `1500` dòng. Có `200 0` (≥ 1, request được nhận trước khi đóng listener) và `000 7` (≥ 1, connection refused sau khi đóng listener — SIGTERM rơi giữa tải). `grep -vc` = `0`: chỉ có 4 loại `200 0`, `000 7`, `000 52`, `000 56` — **không** có status 5xx, không `000 28` (timeout), không mã curl khác. Số `000 52` (empty reply) + `000 56` (reset) ≤ `2` trên 1500: kết nối ở backlog/đã `Accept` nhưng request đọc xong sau `shuttingDown()` (A10, A11), chỉ tránh được bằng drain + load balancer — ghi vào handbook như lý do G3 cần drain/LB, không tính là lỗi của task. `handled` = `ok`: mọi request server đã xử lý (dòng access log sau `waitcode`) đều tới client với `200`, không request nào được xử lý mà client mất response. Access log chỉ có status `200` | ✅ |
| P0-T09-TC18 | Unit | P0-AT08 đúng nguyên văn bằng Go test (A10): `cd $SRV && go test -race -count=1 -v ./services/core/... 2>&1 \| grep -iE -- '--- (PASS\|FAIL).*(AT08\|Sleep\|Slow\|InFlight)'`; `grep -rnE '3 \* time\.Second\|time\.Second \* 3\|3s' $CORE/cmd $CORE/internal --include='*_test.go' \| head` | Có ít nhất 1 test `--- PASS` mà: server dựng bằng **cùng** hàm/luồng shutdown với `main` (không chép lại logic), route test ngủ 3 s, trigger shutdown (huỷ `ctx` hoặc gửi SIGTERM thật tới chính process) sau 1 s; request nhận `200`; hàm trả về không lỗi (↔ exit `0`); thời gian từ trigger tới lúc trả về ≈ 2 s (≥ 1.5 s, < 3 s) | ✅ |
| P0-T09-TC19 | Hồi quy | Hành vi P0-T07/P0-T08 không đổi, log vẫn JSON: `corerun CORE_DATABASE_URL="$DSN"`; `waitcode $C/readyz 200 10`; `code $C/healthz; code $C/khong-co; code -X POST $C/metrics; code $C/metrics`; `curl -s -D - -o /dev/null $C/healthz \| grep -ci '^x-request-id:'`; `sig TERM; waitexit 5`; `jsonlines $S/core.log`; `bash scripts/test-all.sh server; echo $?` | `/readyz` `200`, `/healthz` `200`, `/khong-co` `404`, `POST /metrics` `405`, `/metrics` `200`. Header `X-Request-ID` có (`1`). `exit=0`. `jsonlines`: `bad=0` — mọi dòng, kể cả các mốc shutdown, là JSON slog. `test-all.sh server` exit `0` | ✅ |
| P0-T09-TC20 | Tài liệu | Docs khớp code (A3): `grep -n 'CORE_SHUTDOWN_TIMEOUT' com/tm/docs/technical/local-setup.md $CORE/.env.example`; `grep -n 'CORE_SHUTDOWN_TIMEOUT' -r $CORE --include='*.go' \| grep -v _test.go \| head -3` | `local-setup.md` có dòng `CORE_SHUTDOWN_TIMEOUT` trong bảng biến môi trường, ghi mặc định `10s` và định dạng duration. `.env.example` của core có biến này kèm chú thích. Tên hằng trong `config` khớp tên biến | ✅ |
| P0-T09-TC21 | Phạm vi | Không làm việc của task khác: `git -C $REPO status --porcelain --untracked-files=all`; `grep -rnE 'otel\.Set(TracerProvider\|MeterProvider)\|sdk/trace\|sdk/metric' $CORE $SRV/pkg --include='*.go' 2>/dev/null \| grep -v _test.go`; `grep -rnE 'time\.Sleep\|/debug\|/slow' $CORE --include='*.go' \| grep -v _test.go`; `grep -rnE '\br\.(Get\|Post\|Put\|Delete\|Method\|Handle\|HandleFunc\|Mount\|Route)\(' $CORE --include='*.go' \| grep -v _test.go`; `ls $CORE/cmd`; `git -C $REPO status --porcelain com/tm/app deploy` | Thay đổi chỉ nằm trong: `com/tm/server/services/core/**` (gồm `.env.example`), `com/tm/server/{go.mod,go.sum,MODULE.bazel,MODULE.bazel.lock}` (chỉ khi thêm thư viện test như goleak), `com/tm/docs/technical/local-setup.md`, `architecture.md` (nếu ghi trình tự shutdown), handbook (`handbook/phase-0/P0-T09.md`, `handbook/README.md`), planning của task (`planning/README.md`, `phase-0-foundation/README.md`, `tasks/P0-T09/**`, `acceptance-tests.md`), `planning/execute-all-notes.md`. Không đụng OTel SDK/provider (P0-T10). Không có `time.Sleep`/route debug trong code chạy thật. Route chạy thật vẫn chỉ `/healthz`, `/readyz`, `/metrics`. `cmd/` chỉ có `server`. `com/tm/app` và `deploy` không đổi (rỗng) | ✅ |
| P0-T09-TC22 | Dọn dẹp | Dọn process và stack: `corestop`; `pkill -f 'import socket' 2>/dev/null; true`; `pgrep -fl "$S/core" \| wc -l \| tr -d ' '`; `lsof -nP -iTCP:8080 -sTCP:LISTEN \| wc -l \| tr -d ' '`; `dc unpause postgres-core 2>/dev/null; true`; `dc down`; `docker ps -q --filter label=com.docker.compose.project=snaptix \| wc -l \| tr -d ' '`; `rm -rf $S` | Không còn process core (`0`), cổng 8080 trống (`0`), không còn container của project (`0`), không container nào bị bỏ ở trạng thái pause. Volume giữ nguyên (`down` không `-v`). Scratchpad đã xoá | ✅ |

Test nghiệm thu liên quan:
- **P0-AT08** (handler sleep 3s, SIGTERM sau 1s → 200, exit 0): dựa chính vào **TC18** (Go test đúng nguyên văn, cùng luồng shutdown với `main`), kèm **TC11** (E2E trên binary thật: `/readyz` đang chạy được trả đủ). TC09 **không** còn là bằng chứng cho P0-AT08 (A10). Đổi ✅ khi TC18 và TC11 pass.
- **P0-AT09** (SIGTERM rồi gửi request mới → bị từ chối): **TC10**. Đổi ✅ khi pass.
- **P0-AT10** (handler quá shutdown timeout → vẫn thoát, log cảnh báo): **TC12** (và U4 trong TC05). Đổi ✅ khi pass.
- **DoD** "Gửi SIGTERM khi đang có request chậm → request hoàn thành, service thoát sạch": TC18, TC11.
- Challenge **G3**: TC17 là phần đo được ở máy dev (số nhỏ `000 52`/`000 56` là lý do cần drain/LB, A11). Tiêu chí "rolling deploy dưới tải không mất/không lỗi request" cần nhiều instance + load balancer (chưa có ở phase 0), nên G3 chuyển 🟨, **không** ✅.
- Challenge **G14**: TC02, TC03 không hồi quy.

## Review

### Review lần 1 — CHANGES_REQUESTED

Reviewer độc lập (execute-all). Đã đối chiếu dòng P0-T09, G3, DoD, P0-NFR3, P0-AT08/09/10, `project-structure.md`, `execute-all-notes.md`, code `services/core/cmd/server/main.go`. Kiểm chứng bằng chương trình Go nhỏ trong scratchpad (Go 1.27.1 darwin/arm64, pgx v5.11.0, container `postgres:18.6-alpine3.23` tạm, đã dọn). Phạm vi, định dạng (tiêu đề, 5 cột, ID `P0-T09-TC01..TC22` liên tục, ⬜, dòng "Test nghiệm thu liên quan") và các giả định A1–A3, A5–A8, A11 hợp lý: docs không đòi drain / `/readyz` 503, P0-AT09 chấp nhận connection refused; exit `1` khi hết timeout không trái P0-AT10; G3 chỉ 🟨 là đúng. Các lỗi dưới là lỗi thực chất, làm TC fail dù code đúng.

1. **A10a sai, kéo theo TC09, TC10, TC14, TC17 fail chắc chắn.** `net/http` của Go 1.27.1 (`server.go` dòng 2054–2061, `conn.serve`): sau `readRequest`, nếu `c.server.shuttingDown()` thì `return` — đóng kết nối **không** trả response. `Shutdown` có chờ kết nối `StateNew` < 5 s (dòng 3312), nhưng khi client gửi nốt header thì request bị bỏ. Kiểm chứng: server tối giản `signal.NotifyContext` + `Shutdown(10s)`, client gửi nửa header, SIGTERM sau ~1 s, gửi nốt sau 2 s — **15/15 lần** client nhận `EOF|2.01`, không phải `HTTP/1.1 200 OK`, dù `Shutdown` chờ ~1.05 s rồi trả `nil`. Hệ quả: TC09 (`slow.txt` = 200, access log `/healthz 200`), TC10 (vế "slowreq vẫn nhận 200"), TC14 (`access 200` trước `http_stopped`), TC17 (`slowreq` 200, đếm access log) đều fail. Cần bỏ A10a làm cách kiểm "request đang chạy được phục vụ xong". Chỉ dùng slowreq để **giữ process sống** (TC10, TC12, TC15 vẫn dùng được, và kết quả mong đợi của slowreq phải là `EOF`), hoặc chuyển phần "request hoàn thành 200" sang handler thật đang chạy (A10b) / Go test (TC18). Sửa A10 cho khớp.

2. **TC11: `pool.Close()` treo ~15 s sau khi `/readyz` hết hạn lúc PG bị pause, `after < 5s` fail với cách làm đơn giản.** Kiểm chứng với pgxpool v5.11.0: pool đã ping OK, `docker pause`, `pool.Ping(ctx 2s)` → `context deadline exceeded 2.00s`, rồi `pool.Close()` mất **15.002 s**. Kết nối bị huỷ giữa chừng được huỷ qua đường cancel/đóng bất đồng bộ, và `Close` chờ nó bị huỷ xong (timeout 15 s trong pgxpool). Đối chứng: không ping lúc pause (chỉ có conn rảnh) thì `Close` mất 0.5 ms. Đây đúng là luồng của TC11. Cần chọn một trong hai và ghi thành giả định: (a) `pool.Close()` có giới hạn thời gian (chạy trong goroutine, chờ tối đa X s hoặc phần hạn còn lại, quá hạn thì log WARN và thoát). Khi đó nêu rõ mốc log, exit code, thêm unit test (fake closer treo) và cập nhật TC11 theo đúng hành vi; hoặc (b) giữ `pool.Close()` chặn và sửa TC11 (`waitexit` ≥ 20, `after` ≈ 2 + 15 s), ghi nhận là hạn chế trong handbook. Hiện A4 không nói gì, nên TC11 đang đòi một hành vi chưa được đặc tả.

3. **TC16: cách chiếm cổng không làm core lỗi listen trên macOS.** Python `socket()` (IPv4) + `SO_REUSEADDR` bind `0.0.0.0:8080`, rồi Go `net.Listen("tcp", ":8080")` (bind `[::]` dual-stack) **vẫn thành công** (đã thử trên cổng 18090). Thử cả khi bỏ `SO_REUSEADDR` với IPv4: Go vẫn listen được. Core sẽ chạy bình thường, TC16 fail. Sửa bằng socket IPv6 dual-stack. Đã kiểm, cách này cho `bind: address already in use`: `python3 -c 'import socket,time;s=socket.socket(socket.AF_INET6);s.setsockopt(socket.IPPROTO_IPV6,socket.IPV6_V6ONLY,0);s.bind(("::",8080));s.listen();time.sleep(8)' &`. Hoặc chạy một instance core thứ hai.

4. **TC17 flaky, tiêu chí "không có `000 52`/`000 56`" quá chặt so với hành vi của Go.** Mô phỏng đúng tải của TC17 (`xargs -P 8 curl`, 1500 request, mỗi request một kết nối mới, SIGTERM sau 1 s): **1/25 lần** có `000 52` (empty reply), 24 lần còn lại chỉ có `200 0` và `000 7`. Nguyên nhân giống điểm 1: kết nối đã được `Accept` nhưng request đọc xong sau khi `shuttingDown()` thì bị đóng không trả lời, hoặc kết nối còn trong backlog lúc đóng listener. Không có load balancer và drain thì server không tránh được việc này (A5 đã loại drain). Cần sửa tiêu chí sao cho vẫn đo được G3 mà không flaky. Ví dụ: mọi request **server đã xử lý** (access log) đều tới client với 200 — tức số `200 0` = số dòng access log của `/healthz` trừ các request của `waitcode`. Cho phép một số rất nhỏ `000 52`/`000 56` (vd ≤ 0.5 %, không có 5xx, không có `000 28`) và giải thích trong handbook. Hoặc chạy N lần rồi nêu ngưỡng. Bỏ luôn vế `slowreq` nhận 200 (điểm 1).

5. **Dòng "Test nghiệm thu liên quan" và A10 phải cập nhật theo điểm 1–2.** P0-AT08/DoD hiện dựa vào TC09 (slowreq). Sau khi sửa, TC18 (Go test, route ngủ 3 s, cùng luồng shutdown với `main`) là bằng chứng chính. E2E trên binary thật thì dùng TC11 (A10b), với điều kiện đã xử lý điểm 2.

Góp ý không chặn:
- TC08: đã kiểm chứng. Process nền trong shell không interactive vẫn nhận SIGINT khi dùng `signal.NotifyContext` (exit 0), nên giữ nguyên.
- TC13: `Shutdown` đóng kết nối rảnh ngay ở lượt `closeIdleConns` đầu tiên, nên `after < 1.0s` hợp lý.
- TC09/TC11: vòng poll của `Shutdown` tăng dần tới ~500 ms, nên `after` có thể trễ tới ~0.55 s so với lúc request xong. Các cận hiện có (`< 3.0s`, `< 5s`) đủ rộng.
- TC12, TC15: vẫn đúng với slowreq (kết quả mong đợi là "không phải 200").

### Sửa lần 1

Agent sinh test case (chế độ sửa, execute-all). Giữ ID `P0-T09-TC01..TC22` (22 test case, không thêm TC mới), 5 cột, mọi trạng thái ⬜.

1. **A10a / client chậm**: viết lại **A10**. `slowreq` không còn dùng để chứng minh "request đang chạy được phục vụ xong", vì `net/http` đóng kết nối không trả response khi đọc xong request lúc đang shutdown. `slowreq` chỉ còn để giữ process sống, kết quả mong đợi phía client là `EOF`. Comment của `slowreq` trong khối chuẩn bị cũng sửa theo.
   - **TC09**: đổi thành "kết nối đang mở giữ `Shutdown` chờ nhưng không kéo tới timeout". Mong đợi `slow.txt` = `EOF|≈3.0`, `exit=0`, `after` trong [1.5, 3.0) s, không có access log sau `signal`. Bỏ tham chiếu P0-AT08.
   - **TC10**: bỏ vế "slowreq nhận 200", thay bằng `slow.txt` bắt đầu bằng `EOF`. P0-AT09 (connection refused) giữ nguyên.
   - **TC14**: bỏ "`access 200` của slowreq trước `http_stopped`". 5 dòng cuối phải là `access 200, signal, http_stopped, pool_closed, done`, không có access sau `signal`. Lệnh python kiểm `signal ≤ http_stopped ≤ pool_closed ≤ done` và `http_stopped − signal ≥ 1.0 s` (Shutdown thực sự chờ). Parser thời gian cắt phần lẻ còn 6 chữ số để chạy được với python3 3.9 của máy.
   - **TC17**: bỏ `slowreq` (xem điểm 4).
   - TC12, TC15 giữ nguyên (đã mong đợi "không phải 200").
2. **Pool treo khi đóng**: chọn phương án (a), thêm giả định **A12**.
   - `pool.Close()` chạy trong goroutine. Hàm shutdown chờ phần còn lại của `CORE_SHUTDOWN_TIMEOUT`, tối thiểu 1 s (hằng không export). Quá hạn thì log `WARN` mốc `pool_close_timeout`, sang `done`, exit `1`. Tổng thời gian ≤ timeout + 1 s.
   - Cập nhật theo: **A4** bước 4, **A6** (thêm mốc `pool_close_timeout` mức WARN), **A7** (exit `1` khi đóng pool quá hạn).
   - **TC05**: U9 thêm trường hợp đóng pool quá hạn, thêm nhóm **U10** (fake closer treo → trả về trong hạn + 0.5 s, lỗi nhận diện được ↔ exit 1, log `pool_close_timeout`; fake closer trả ngay → `pool_closed`).
   - **TC11**: chạy với `CORE_SHUTDOWN_TIMEOUT=4s`, `after` ≤ 5.0 s. Mong đợi `signal http_stopped pool_close_timeout done`, WARN, `exit=1`, `after` ≥ 3.5 s. Nếu `Close` xong trong hạn thì chấp nhận nhánh `pool_closed`/`exit=0`, với điều kiện mốc log và exit code khớp nhau. `ready.txt` vẫn phải là `503 ≈2.0 rc=0`.
3. **TC16**: chiếm cổng bằng socket IPv6 dual-stack (`AF_INET6`, `IPV6_V6ONLY=0`, bind `::`), đúng lệnh reviewer đã kiểm. Thêm bước `lsof` xác nhận python đang LISTEN. Mong đợi `address already in use`, `exit=1` (≠ 0).
4. **TC17**: sửa tiêu chí.
   - Bỏ `slowreq`. Ghi `A0` = số dòng access log sau `waitcode`.
   - Chấp nhận 4 loại kết quả: `200 0`, `000 7`, `000 52`, `000 56`. Không có 5xx, không `000 28`, không mã khác.
   - Tổng `000 52` + `000 56` ≤ 2/1500, ghi vào handbook là lý do G3 cần drain/LB.
   - `handled` (access log mới) = `ok` (số `200 0`): mọi request server đã xử lý đều về client 200.
   - Cập nhật **A11** tương ứng.
5. **A10 và "Test nghiệm thu liên quan"**: P0-AT08 dựa chính vào TC18, kèm TC11 (E2E). P0-AT09 dựa vào TC10, P0-AT10 dựa vào TC12. DoD dựa vào TC18 và TC11. Ghi rõ TC09 không còn là bằng chứng cho P0-AT08.

### Review lần 2 — CHANGES_REQUESTED

Reviewer độc lập (execute-all), lần 2. Kiểm chứng bằng chương trình Go mô phỏng thiết kế trong scratchpad (Go 1.27.1 darwin/arm64, pgx v5.11.0, `slog.NewJSONHandler` giống `main.go`, `signal.NotifyContext` → `Shutdown(ctx timeout)` → `Close()` khi quá hạn → `pool.Close()` trong goroutine chờ `max(timeout − đã dùng, 1s)` → log `shutdown_step` → `os.Exit`). PG tạm `postgres:18.6-alpine3.23` trên cổng trống. Khối chuẩn bị chạy nguyên văn, chỉ đổi `BIN`, `DSN` và `dc pause/unpause` sang container tạm. `bash -n` (bash 3.2.57) và `zsh -n` đều qua. Đã dọn process, container, cổng và scratchpad. Không có lệnh ghi lịch sử git.

Đã xử lý đúng và kiểm chứng được:
- **Điểm 1 (A10, TC09/TC10/TC14/TC17)**:
  - TC09 chạy 4 lần (zsh + bash 3.2): `ALIVE`, `slow.txt` = `EOF|3.01`, `exit=0`, `after` 2.21–2.27 s (trong [1.5, 3.0)), `steps` đúng, không có `access` sau `signal`.
  - TC10: `000 rc=7` ×2, LISTEN `0`, `exit=0`, `EOF|3.01`.
- **Điểm 2 (A12, TC11)**: chạy 3 lần, cả 3 lần `ALIVE`, `ready.txt` = `503 2.003 rc=0`, `steps` = `signal http_stopped pool_close_timeout done`, `WARN`, `done.exit_code` = `1`, `exit=1`, `after` 4.07–4.12 s (≥ 3.5, ≤ 5.0). A12 nhất quán với A4 bước 4, A6, A7, U9/U10 của TC05, TC11 và TC12 (TC12 thực tế: `signal timeout pool_closed done`, `exit=1`, `after=1.13s`).
- **Điểm 4 (TC17)**: chạy 12 lần với tiêu chí mới, **12/12 PASS**. Mỗi lần: `n=1500`, chỉ có `200 0` (419–435) và `000 7`, `000 52`/`000 56` = 0, `handled` = `ok`, access log chỉ `200`, `exit=0`. Tiêu chí ổn định.
- **Điểm 5**: dòng "Test nghiệm thu liên quan" đã cập nhật đúng (TC18 + TC11 cho P0-AT08/DoD).
- Định dạng đạt: tiêu đề, 5 cột, ID `P0-T09-TC01..TC22` liên tục, ⬜, có dòng "Test nghiệm thu liên quan".

Lỗi thực chất còn lại (TC fail dù code đúng):

1. **P0-T09-TC14 — parser thời gian lỗi ngẫu nhiên trên python3 3.9.6 của máy (`/usr/bin/python3`).**
   - slog JSON ghi `time` theo RFC3339Nano, bỏ số 0 cuối, nên phần lẻ có số chữ số thay đổi. Ví dụ đo được: `23:24:40.53188+07:00` (5 chữ số); trong một log, 2/10 dòng có 5 chữ số.
   - `datetime.fromisoformat` của 3.9 chỉ nhận đúng 3 hoặc 6 chữ số. Regex hiện tại chỉ **cắt** phần dài hơn 6 chữ số, không **bù** phần ngắn hơn.
   - Chạy TC14 nguyên văn 8 lần: **3/8** lệnh python lỗi `ValueError: Invalid isoformat string`. Các phần khác của TC14 pass (`exit=0`, 5 dòng cuối đúng thứ tự, `True 1.6x` khi parse được).
   - Sửa: chuẩn hoá phần lẻ về đúng 6 chữ số, ví dụ `P=lambda s:d.datetime.fromisoformat(re.sub(r"\.(\d+)",lambda m:"."+(m.group(1)+"00000")[:6],s).replace("Z","+00:00"))`. Đã kiểm trên 3.9.6 với 5, 1, 9 chữ số, không có phần lẻ, và hậu tố `Z`.
2. **P0-T09-TC16 — lệnh chiếm cổng nguyên văn fail khi chạy theo thứ tự sau các TC E2E trước.**
   - (a) Socket python không đặt `SO_REUSEADDR`. Sau TC06–TC15, cổng 8080 còn nhiều kết nối `TIME_WAIT` phía server (đo được 15–517). Trên macOS, `bind` không có `SO_REUSEADDR` khi đó lỗi `OSError: [Errno 48] Address already in use`: python chết ngay, core listen **thành công** và chạy bình thường. Kết quả: `NOT_EXITED after 3s`, TC fail. Reviewer lần 1 kiểm trên cổng sạch 18090 nên không gặp.
   - Sửa: thêm `s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1)` trước `bind`. Đã kiểm: khi còn 517 `TIME_WAIT`, python vẫn LISTEN được, core vẫn nhận `listen tcp :8080: bind: address already in use`, `exit=1`, `after=0.03s`. Lý do: Go cũng đặt `SO_REUSEADDR`, nhưng hai socket cùng bind chính xác `[::]:8080` mà không có `SO_REUSEPORT` thì vẫn xung đột.
   - (b) `lsof ... | grep -c python` in `0` dù python đang LISTEN: trên macOS, cột COMMAND là `Python` (chữ hoa, binary framework của `/usr/bin/python3`). Sửa thành `grep -ci python`.

Góp ý không chặn:
- TC17: trên macOS `wc -l < file` in có khoảng trắng đầu dòng (`    1500`). Người chạy cần hiểu "đủ `1500` dòng" là so số.
- TC09: `after` thực tế ~2.2 s. Cận [1.5, 3.0) vẫn có biên ~0.7 s kể cả khi `Shutdown` poll trễ 0.5 s.

### Sửa lần 2

Agent sinh test case (chế độ sửa, execute-all). Giữ ID `P0-T09-TC01..TC22`, 5 cột, mọi trạng thái ⬜. Không đổi giả định hay kết quả mong đợi.

1. **P0-T09-TC14**: hàm parse thời gian chuẩn hoá phần lẻ giây về đúng 6 chữ số (bù `0` hoặc cắt): `re.sub(r"\.(\d+)",lambda m:"."+(m.group(1)+"000000")[:6],s)`, rồi đổi hậu tố `Z` thành `+00:00`; offset `+07:00` giữ nguyên. Đã chạy bằng `/usr/bin/python3` 3.9.6 với `2026-09-26T10:00:00.53188+07:00` (5 chữ số), `...00.123456789Z` (9 chữ số), `...00Z` (không có phần lẻ), `...00.5+07:00`, `...00+07:00`, `...00.123456-05:30`: tất cả parse được. `03:00:00.5Z` bằng `10:00:00.5+07:00`.
2. **P0-T09-TC16**:
   - (a) Socket python chiếm cổng thêm `SO_REUSEADDR` trước `bind`, để bind được khi 8080 còn `TIME_WAIT` từ các TC trước. Vẫn giữ `AF_INET6`, `IPV6_V6ONLY=0`, bind `::`.
   - (b) Đếm LISTEN bằng `grep -ci python`, vì trên macOS `lsof` in `Python` viết hoa. Đã chạy thử trên cổng 18091: in `1`.
3. **Góp ý nhỏ**: mọi chỗ đếm bằng `wc -l` trong bảng (TC06, TC07, TC10, TC12, TC17, TC22) thêm `\| tr -d ' '` để bỏ khoảng trắng đầu dòng trên macOS, output so khớp trực tiếp với số mong đợi.

### Review lần 3 — APPROVED

Reviewer độc lập (execute-all), lần 3. Chỉ kiểm lại các điểm của Sửa lần 2. Các phần khác đã được Review lần 2 xác nhận. Kiểm chứng trong scratchpad (Go 1.27.1 darwin/arm64, `/usr/bin/python3` 3.9.6 — `which -a python3` chỉ ra đúng binary này). Đã dọn process, cổng và scratchpad. Không có lệnh ghi lịch sử git.

1. **P0-T09-TC14 — đạt.**
   - Lệnh python được trích nguyên văn từ bảng (bỏ escape `\|`) và chạy trên log do một chương trình Go dùng `slog.NewJSONHandler` sinh ra. Có 5 lần chạy, mỗi lần 84 dòng: 60 dòng giờ `+07:00`, 4 dòng `shutdown_step`, 20 dòng UTC hậu tố `Z`. Phần lẻ giây gồm 383 dòng 6 chữ số, 33 dòng 5 chữ số, 4 dòng 4 chữ số.
   - Cả 5 lần in `True 0.0`, không có `ValueError`.
   - Parse toàn bộ 420 dòng, kể cả khi trộn `+07:00` với `Z`: thứ tự thời gian không giảm trong cả 5 file.
   - Kiểm thêm các mẫu 1, 4, 5, 9 chữ số, không có phần lẻ, `Z` và `+07:00`: tất cả chuẩn hoá đúng về 6 chữ số.
2. **P0-T09-TC16 — đạt.**
   - Cổng trống 18197. Chạy một HTTP server Go (`Connection: close`), bắn 40 request curl rồi dừng: còn 40 `TIME_WAIT`.
   - Đối chứng không có `SO_REUSEADDR`: `OSError: [Errno 48] Address already in use`. Đây đúng là lỗi Review lần 2 nêu.
   - Chạy lệnh socket mới nguyên văn (chỉ đổi số cổng) thì bind và LISTEN được. `lsof -nP -iTCP:<cổng> -sTCP:LISTEN | grep -ci python` in `1` (COMMAND = `Python`, IPv6 `*:18197`).
   - Go `net.Listen("tcp", ":18197")` báo `listen tcp :18197: bind: address already in use`. Sau `kill $BP` cổng được giải phóng.
3. **`| tr -d ' '` (TC06, TC07, TC10, TC12, TC17, TC22) — đạt.**
   - Trích mọi lệnh trong backtick của các TC này và TC14, TC16, đổi `\|` thành `|`, ghép với khối chuẩn bị. `bash -n` (bash 3.2) và `zsh -n` đều qua.
   - `A0=$(acc | wc -l | tr -d ' ')`, `A1=…` và `$((A1-A0))` vẫn là phép tính số. `wc -l < $S/load.txt | tr -d ' '` in đúng `1500`.
   - Escape `\|` trong ô bảng markdown đúng quy ước.
4. **Định dạng và phạm vi — đạt.**
   - Tiêu đề và bảng 5 cột đúng. ID `P0-T09-TC01..TC22` liên tục, 22 dòng, đều ⬜. Có dòng "Test nghiệm thu liên quan".
   - Không có lệnh ghi lịch sử git.
   - Sửa lần 2 không đổi giả định hay kết quả mong đợi, không phát sinh lỗi mới.

Góp ý không chặn: không có.

## Ghi chú khi thực thi

- Chạy 2026-09-26 trên máy dev (macOS arm64, Go 1.27.1, Bazel 8.7.0, golangci-lint 2.14.0, `/usr/bin/python3` 3.9.6). Bash tool chạy zsh. `prep.sh` tạo bằng Write tool. Mọi file trong `com/tm/server/**` sửa bằng Edit/Write.
- Thiết kế: logic nằm trong `services/core/cmd/server/shutdown.go` (`run`, `stopHTTP`, `closePool`, `exitCode`), test ở `shutdown_test.go` (`//services/core/cmd/server:server_test`). Tín hiệu qua `signal.Notify` vào channel buffer 2. Tín hiệu thứ hai: `srv.Close()` rồi trả `errSecondSignal`, exit `1`, không đóng pool. Không thêm thư viện (không goleak): U7 so `runtime.NumGoroutine()` có chờ ổn định.
- P0-T09-TC05: PG dừng, `go test -race -count=1 ./...` và `-count=5 ./services/core/...` đều `ok`. 0 FAIL/SKIP. 16 dòng bảng/`t.Run`. Nhóm U: U1 (SIGTERM, huỷ ctx), U2, U4, U10 ×2 trong `TestRunShutdown`. U3 là `TestRunRejectsNewConnections`, U5 là `TestRunListenError`. U6 kiểm thứ tự sự kiện trong mọi dòng của `TestRunShutdown`. U7 là `checkNoLeak` trong mọi test chạy `run`. U8 là `TestLoadShutdownTimeout`, U9 là `TestExitCode`. A8 có `TestRunSecondSignal`. Lần đầu `TestRunRejectsNewConnections` fail: `net.Dial` tới listener đang đóng trả `i/o timeout` trước khi thành `connection refused` (macOS). Đã sửa test để thử lại tới khi gặp `ECONNREFUSED`, tiêu chí giữ nguyên. `TestRunAT08...` lần đầu báo rò 1 goroutine: đó là goroutine của `os/signal`, sống suốt process sau `signal.Notify` đầu tiên. Đã đo mốc sau khi đăng ký.
- P0-T09-TC06: unit 33 dòng `--- PASS` (TestLoad + TestLoadShutdownTimeout + TestLoadConnectTimeout). E2E `abc`/`10`/`0`/`-1s`: đều `exit=1` sau 0.03–0.44 s, log có biến (`1`), cổng `0`. `' '` → `exit=0`, `timeout` = `10s`.
- P0-T09-TC07/TC08: `exit=0 after=0.03s`, `signal http_stopped pool_closed done`. `terminated` / `interrupt`.
- P0-T09-TC09: `ALIVE`, `EOF|3.01`, `exit=0 after=2.21s`. Chuỗi `access signal http_stopped pool_closed done`: `access` duy nhất là của `waitcode`, trước `signal`.
- P0-T09-TC10: `000 rc=7` ×2, LISTEN `0`, `exit=0 after=2.74s`, `EOF|3.01`.
- P0-T09-TC11: `ALIVE`, `ready.txt` = `503 2.001839 rc=0`, `exit=1 after=4.10s`, `signal http_stopped pool_close_timeout done`, `WARN`, `exit_code` `1`. Ngay sau `dc unpause`, `dc up -d --wait` báo `unhealthy` (healthcheck chưa kịp chạy lại). Chạy lại sau vài giây thì `healthy`.
- P0-T09-TC12: `exit=1 after=1.11s`, `signal timeout pool_closed done`, `WARN`, `exit_code` `1`, `EOF|4.01`, socket 8080 `0`.
- P0-T09-TC13: `exit=0 after=0.04s`, `HTTP/1.1 200 OK|CLOSED|0.55`.
- P0-T09-TC14: `exit=0 after=1.70s`, `EOF|2.01`, 5 dòng cuối `access 200, signal, http_stopped, pool_closed, done`, python `True 1.62`.
- P0-T09-TC15: `ALIVE`, sau tín hiệu 2 `exit=1 after=0.04s`, `EOF|4.01`. `steps` = `signal done`, có dòng WARN "nhận tín hiệu thứ hai".
- P0-T09-TC16: `lsof` python `1`, `exit=1 after=0.03s`, `err` = `listen tcp :8080: bind: address already in use`, `steps` = `pool_closed done`, `exit_code` `1`.
- P0-T09-TC17: `exit=0 after=0.10s`, `1500` dòng: `424 200 0` + `1076 000 7`. Loại khác `0`, `000 52/56` = `0`, `handled=424 ok=424`, access log chỉ `200`. Chạy thêm 3 lần: 422/428/427 `200 0`, lần nào cũng `handled` = `ok`, không có `52`/`56`.
- P0-T09-TC18: `TestRunAT08InFlightSleep3sRealSIGTERM` PASS: SIGTERM thật tới chính process sau 1 s, handler ngủ `3 * time.Second`, `200`, `run` trả `nil`, từ SIGTERM tới khi trả về 2.13 s.
- P0-T09-TC19: `200 200 404 405 200`, `X-Request-ID` `1`, `exit=0`, `lines=11 bad=0`, `test-all.sh server` exit `0`.
- P0-T09-TC21: grep `/debug` khớp `import "runtime/debug"` trong `httpx/recover.go` (có từ P0-T08, dùng `debug.Stack()`). Đây không phải route debug, cũng không có `time.Sleep`. Route chạy thật vẫn chỉ `/healthz`, `/readyz`, `/metrics`.
- P0-T09-TC22: process `0`, cổng `0`, container `0`, không container pause. `down` không `-v`, 7 volume còn nguyên. Scratchpad đã xoá.
