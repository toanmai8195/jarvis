# Test cases — P0-T05: Makefile / script: `make up`, `make migrate`, `make test`

> **Phạm vi**: `Makefile` ở gốc repo với ba target `up`, `migrate`, `test`, kèm script trong `scripts/` nếu người làm tách logic ra (và `_test.sh` của script đó). Căn cứ là các docs sau:
> - Phase 0 README: dòng P0-T05, **P0-FR1** "Một lệnh khởi động toàn bộ hạ tầng local", **P0-FR2** "Migration chạy được cho PG core và PG analytics". DoD: "Clone repo mới → `make up && make migrate` chạy thành công trong < 5 phút".
> - [acceptance-tests](../../acceptance-tests.md): **P0-AT01** "Máy sạch, clone repo, chạy `make up && make migrate`" → "Tất cả container healthy, migration áp dụng thành công".
> - [local-setup](../../../../local-setup.md): bước 1 `docker compose -f deploy/docker-compose.yml up -d --wait`. Bước 2 là hai lệnh `goose -dir com/tm/server/db/{core,analytics}/migrations postgres "$..._DATABASE_URL" up`, với URL mặc định ở bảng "Biến môi trường". goose pin **v3.28.0**, cài trên host (không chạy trong container). Mục "Build & kiểm thử" có `go test ./...`, `bazel test //...` (trong `com/tm/server`) và `pnpm lint && pnpm typecheck && pnpm test && pnpm build` (trong `com/tm/app`).
> - `CLAUDE.md` bước 4 (Server): `go vet ./...`, `go test -race ./...`, `bazel test`.
> - [execute-all-notes](../../../execute-all-notes.md): mục P0-T01 và P0-T02 gợi ý Makefile gọi `scripts/check-structure*.sh` và `scripts/check-compose*.sh`. Mục P0-T01a gợi ý gọi `bazel test //...` và `go test -race ./...`. Mục P0-T03: `make migrate` là hai lệnh bước 2 của local-setup, cần `$(go env GOPATH)/bin` trong `PATH`, và `~/go/bin` **chưa có** trong `PATH` của máy dev. Mục P0-T02: không bind mount repo vào container, vì Docker Desktop treo khi mount thư mục trong `~/Documents`.
>
> **Giả định** (reviewer xem kỹ):
> - A1. **Vị trí và công cụ**: `Makefile` nằm ở gốc repo, chạy bằng GNU make có sẵn trên macOS (**3.81**, `/usr/bin/make`). Makefile không dùng tính năng từ 3.82 trở lên: `.ONESHELL`, `.RECIPEPREFIX`, `undefine`, `!=`, `::=`, `$(file ...)`, `--output-sync`, `private`. `SHELL := bash` được phép. Ba target `up`, `migrate`, `test` khai báo `.PHONY`. Đường dẫn trong recipe tính theo vị trí Makefile, không theo cwd, nên `make -C <repo>` từ thư mục khác vẫn chạy đúng.
> - A2. **`make up`** tương đương bước 1 của local-setup: `docker compose -f deploy/docker-compose.yml up -d --wait` cho **cả 8 service**. Lệnh này idempotent và exit khác 0 khi Docker không dùng được.
> - A3. **`make migrate`** chạy **goose trên host**, không dùng container mount repo. Lệnh này migrate **cả** core và analytics bằng `CORE_DATABASE_URL` và `ANALYTICS_DATABASE_URL`. Nếu biến chưa đặt thì dùng mặc định theo bảng "Biến môi trường" (`postgres://snaptix:snaptix@localhost:5432/core`, `postgres://snaptix:snaptix@localhost:5433/analytics`). Nếu biến đã đặt thì lệnh dùng giá trị đó, không hard-code. Lệnh chạy được khi `PATH` **không** chứa `$(go env GOPATH)/bin`: tìm goose trong `PATH` trước, không có thì tìm trong `$(go env GOPATH)/bin`. Nếu vẫn không có goose thì chọn một trong hai cách: (a) exit khác 0 với thông báo nêu lệnh cài đúng `go install github.com/pressly/goose/v3/cmd/goose@v3.28.0`, hoặc (b) tự cài đúng bản đó. Không bao giờ dùng `@latest`. Lỗi kết nối DB phải làm lệnh exit khác 0, và thông báo phải cho biết DB nào lỗi (core/analytics) hoặc gợi ý chạy `make up`. Không được treo.
> - A4. **`make test`** chạy bộ test hiện có theo docs, không cần stack Docker đang chạy và không tự bật stack:
>   - trong `com/tm/server`: `go vet ./...`, `go test -race ./...`, `bazel test //...`;
>   - trong `com/tm/app`: `pnpm test` (đệ quy, bỏ qua package không có script). Người làm có thể chạy thêm `lint`/`typecheck`/`build`. Nếu có `pnpm install` thì phải là `--frozen-lockfile`;
>   - trong `scripts/`: mọi `scripts/*_test.sh`, cùng `scripts/check-structure.sh` và `scripts/check-compose.sh` (chỉ đọc config, không cần stack chạy).
>
>   Chỉ cần **một** bộ fail là `make test` exit khác 0: lỗi không được che bởi `;`, `|| true` hay một lệnh chạy sau nó. `golangci-lint` và `bazel run //:gazelle` **không bắt buộc** trong `make test`. Nếu người làm thêm thì `make test` vẫn không được sửa file đã track.
> - A5. Docs **không** định nghĩa `make help`, `make down` hay target khác, nên bộ test không kiểm các target đó. Người làm thêm target phụ thì được, nhưng target mặc định (`make` không đối số) **không** được xoá dữ liệu (`down -v`) hay chạy migration. Dọn stack trong test dùng trực tiếp lệnh `$DC down -v`.
> - A6. Ngưỡng "< 5 phút" của DoD tính trên máy dev khi image đã có trong cache Docker (`$DC pull` và build image observability đã chạy một lần). Thời gian tải image từ mạng không tính, giống cách P0-T02-TC05 đo.
>
> **Không thuộc task này**: dashboard Grafana (P0-T06), target chạy service core/bff/web (P0-T07+, P0-T11+), đổi `deploy/docker-compose.yml`, migration hay workflow CI (`.github/workflows/ci.yml` không bắt buộc gọi `make`), testcontainers (P0-T16), Vitest (P0-T17), seed dữ liệu.
>
> **Môi trường khi viết**: macOS arm64, GNU Make 3.81, Docker 29.8, go 1.27.1 (`/opt/homebrew/bin`), Bazelisk (`/opt/homebrew/bin/bazel`, `.bazelversion` 8.7.0), pnpm (`~/.local/bin`, tự chuyển sang 11.18.0), jq, nc, shellcheck. goose v3.28.0 **chỉ có ở `~/go/bin`**, thư mục này không nằm trong `PATH`. Máy không có `psql` trên host (truy vấn qua `docker compose exec`). Workspace `com/tm/app` chưa có package nào.
>
> **Chuẩn bị**: Bash tool mở shell mới mỗi lần gọi, nên lưu khối dưới thành file `$S/prep.sh` một lần (thay `<scratchpad>` bằng đường dẫn thật; tạo bằng Write tool hoặc heredoc `cat > $S/prep.sh <<'EOF'`). Mỗi lần gọi Bash bắt đầu bằng `source <scratchpad>/p0t05/prep.sh` (file tự `cd $REPO`). Biến so sánh trước/sau (`A`/`B`/`N0`/`N1`, `s`) phải tính trong **cùng một** lần gọi. Phần chuẩn bị và các TC **không** có lệnh ghi lịch sử git nào, kể cả trong clone tạm: hook `guard-bash` của execute-all đọc text lệnh và từ chối mọi lệnh như vậy không có tag task. Clone chỉ `git add -A`, và `git checkout -- <path>` khôi phục từ index là đủ.
> ```bash
> REPO=/Users/maiductoan/Documents/code/jarvis; cd $REPO
> S=<scratchpad>/p0t05; mkdir -p $S         # scratchpad nằm ở /private/tmp/..., ngoài ~/Documents
> DC="docker compose -f deploy/docker-compose.yml"
> MAKE=/usr/bin/make                      # GNU Make 3.81
> qcore() { $DC exec -T postgres-core      psql -U snaptix -d core      -Atc "$1"; }
> qan()   { $DC exec -T postgres-analytics psql -U snaptix -d analytics -Atc "$1"; }
> # PATH không có ~/go/bin (A3); vẫn có go, docker, pnpm, bazel, jq
> NOGOPATH=$(printf '%s' "$PATH" | tr ':' '\n' | grep -v "$(go env GOPATH)/bin" | paste -sd: -)
> # Bản clone tạm chứa working tree hiện tại (kể cả file chưa track của task), dùng cho ca cần sửa code
> # Tắt Bazel server của clone cũ trước khi xoá, để không để lại process/output base
> bzoff() { [ -d $S/repo/com/tm/server ] && (cd $S/repo/com/tm/server && bazel shutdown) >/dev/null 2>&1; true; }
> # Working tree hiện tại được đưa vào index của clone; git checkout -- <path> khôi phục về trạng thái này
> mkclone() { bzoff; rm -rf $S/repo; git clone -q "$REPO" $S/repo
>   rsync -a --delete --exclude .git --exclude node_modules --exclude 'bazel-*' "$REPO"/ $S/repo/
>   git -C $S/repo add -A; }
> # Ảnh chụp trạng thái stack (TC19): container ID + StartedAt, volume của project, bảng goose của core/analytics
> snap() { $DC ps -q | sort | xargs docker inspect -f '{{.Id}} {{.State.StartedAt}}'
>   docker volume ls --filter label=com.docker.compose.project=snaptix -q | sort
>   qcore "select coalesce(to_regclass('goose_db_version')::text,'none')"
>   qan   "select coalesce(to_regclass('goose_db_version')::text,'none')"; }
> # Makefile được phép gọi script trong scripts/ thay cho lệnh trực tiếp (Phạm vi). dryhas <target> <regex>:
> # tìm regex trong output `make -n <target>` HOẶC trong nội dung scripts/*.sh mà output đó gọi (theo tối đa 2 cấp script gọi script).
> # In dòng khớp; không khớp thì exit 1.
> dryhas() { local out f1 f2; out=$(cd "$REPO" && $MAKE -n "$1" 2>&1)
>   f1=$(printf '%s\n' "$out" | grep -oE 'scripts/[A-Za-z0-9_.-]+\.sh' | sort -u)
>   f2=$(for f in $f1; do grep -oE 'scripts/[A-Za-z0-9_.-]+\.sh' "$REPO/$f"; done | sort -u)
>   { printf '%s\n' "$out"; for f in $(printf '%s\n' $f1 $f2 | sort -u); do cat "$REPO/$f"; done; } | grep -E -- "$2"; }
> ```

| ID | Loại | Kịch bản | Kết quả mong đợi | Trạng thái |
|---|---|---|---|---|
| P0-T05-TC01 | Cấu trúc | Makefile tồn tại và khai báo đủ target: `ls Makefile`; `$MAKE -pn -f Makefile 2>/dev/null \| grep -E '^\.PHONY:'`; `$MAKE -n up migrate test >/dev/null; echo $?` | `Makefile` ở gốc repo. `.PHONY` chứa `up`, `migrate`, `test`. `make -n` exit `0` (cả ba target tồn tại, không lỗi cú pháp) | ✅ |
| P0-T05-TC02 | Tương thích | Make 3.81 (A1): `$MAKE --version \| head -1`; `grep -n -E '^[[:space:]]*(\.ONESHELL\|\.RECIPEPREFIX\|undefine[[:space:]]\|private[[:space:]])\|^[A-Za-z_][A-Za-z0-9_.-]*[[:space:]]*(!=\|::=)\|\$\(file \|--output-sync' Makefile` (chỉ bắt `!=`/`::=` ở dòng gán biến, không bắt `!=` trong lệnh shell của recipe); nếu có script mới trong `scripts/` thì chạy `shellcheck scripts/<script mới>.sh` | In `GNU Make 3.81`. `grep` không in gì. `shellcheck` không có cảnh báo (nếu có script) | ✅ |
| P0-T05-TC03 | Dry-run | `make -n` không chạy lệnh thật, và lệnh (trực tiếp hoặc qua script) đúng docs: `$DC down -v`; `$MAKE -n up migrate test; echo $?`; `docker ps -a --filter label=com.docker.compose.project=snaptix -q \| wc -l`; rồi kiểm nội dung bằng `dryhas` (Chuẩn bị): `dryhas up 'deploy/docker-compose\.yml'`; `dryhas up 'up .*-d.*--wait\|--wait.*-d'`; `dryhas migrate goose`; `dryhas migrate 'migrations'`; `dryhas migrate core`; `dryhas migrate analytics`; `dryhas test 'go vet'`; `dryhas test 'go test.*-race'`; `dryhas test 'bazel.* test'`; `dryhas test 'pnpm'`; `dryhas test '_test\.sh'`; `dryhas test 'check-structure\.sh'`; `dryhas test 'check-compose\.sh'` | `make -n` exit `0`. Không có container nào được tạo (`0`). **Mọi** lời gọi `dryhas` đều in ít nhất một dòng (exit `0`). Hợp lệ theo cả hai cách: recipe là lệnh trực tiếp (`docker compose -f deploy/docker-compose.yml up -d --wait`, `goose -dir .../db/{core,analytics}/migrations ... up`, lệnh test A4) **hoặc** recipe gọi `scripts/<x>.sh` mà nội dung script chứa các lệnh đó (script được phép dùng biến/vòng lặp cho `core`/`analytics`; việc migrate đúng cả hai DB được TC08 kiểm bằng hành vi) | ✅ |
| P0-T05-TC04 | .PHONY | Có file trùng tên target mà target vẫn chạy. Trong clone: `mkclone; cd $S/repo; touch test up migrate`; `$MAKE -n test up migrate` | Output **không** có `is up to date` / `Nothing to be done`. Output in lệnh thật của cả ba target | ✅ |
| P0-T05-TC05 | Up | Khởi động từ trạng thái sạch (image đã có trong cache, A6): `$DC down -v`; `time $MAKE up; echo $?`; `$DC ps --format '{{.Service}} {{.State}} {{.Health}}' \| sort` | Exit `0`. `ps` liệt kê đủ 8 service `grafana`, `mongodb`, `otel-collector`, `postgres-analytics`, `postgres-core`, `prometheus`, `redis`, `tempo`, tất cả `running healthy` | ✅ |
| P0-T05-TC06 | Idempotent | `make up` lần 2 không tạo lại container: `B=$($DC ps -q \| sort \| xargs docker inspect -f '{{.Id}} {{.State.StartedAt}}')`; `$MAKE up; echo $?`; lấy lại `A` bằng cùng lệnh rồi `[ "$A" = "$B" ] && echo same` | Exit `0`. In `same`: container ID và `StartedAt` không đổi. Vẫn đủ 8 service healthy | ✅ |
| P0-T05-TC07 | Negative | Docker không dùng được: `DOCKER_HOST=unix:///tmp/khong-co.sock $MAKE up; echo $?` | Exit **khác 0**. Output có lỗi kết nối Docker. Lệnh không treo (xong trong < 30 s, đo bằng `date +%s`). Chỉ đổi đích của CLI, không động tới Docker Desktop; Makefile/script không hard-code `--context`/`-H` (nếu hard-code thì `DOCKER_HOST` không có tác dụng và ca này fail) | ✅ |
| P0-T05-TC08 | Migrate | Trên DB sạch, `PATH` **không** có `~/go/bin` (A3), không đặt biến URL: `$DC down -v && $MAKE up`; `unset CORE_DATABASE_URL ANALYTICS_DATABASE_URL`; `env PATH=$NOGOPATH bash -c 'command -v goose \|\| echo none'` (`command` là builtin nên phải chạy qua `bash -c`); `env PATH=$NOGOPATH $MAKE migrate; echo $?`; `qcore "select version_id, is_applied from goose_db_version order by id"`; `qan "select version_id, is_applied from goose_db_version order by id"` | `command -v` in `none`. `make migrate` exit `0`, output có `OK` cho **cả** file `00001_init.sql` của core lẫn analytics. Mỗi DB có bảng `goose_db_version` với dòng version `1` và `is_applied` = `t` | ✅ |
| P0-T05-TC09 | Idempotent | `make migrate` lần 2: `N0=$(qcore "select count(*)\|\|'/'\|\|max(id) from goose_db_version")/$(qan "select count(*)\|\|'/'\|\|max(id) from goose_db_version")`; `env PATH=$NOGOPATH $MAKE migrate; echo $?`; tính lại `N1` bằng cùng lệnh | Exit `0`. Output có `no migrations to run` cho cả hai DB. `N1` = `N0` | ✅ |
| P0-T05-TC10 | Biến môi trường | Lệnh dùng URL từ biến, không hard-code (A3): `env PATH=$NOGOPATH CORE_DATABASE_URL=postgres://snaptix:snaptix@localhost:5499/core $MAKE migrate; echo $?` (kiểm trước bằng `! nc -z localhost 5499`), rồi làm tương tự với `ANALYTICS_DATABASE_URL=postgres://snaptix:snaptix@localhost:5499/analytics` | Cả hai lần exit **khác 0**. Output có `5499` hoặc lỗi kết nối, và nêu DB bị lỗi (`core` ở lần 1, `analytics` ở lần 2). Chứng minh make dùng giá trị biến | ✅ |
| P0-T05-TC11 | Negative | DB chưa chạy: `$DC down`; `s=$(date +%s); env PATH=$NOGOPATH $MAKE migrate; echo $?; echo $(( $(date +%s)-s ))` | Exit **khác 0**. Thông báo rõ: nêu DB core và/hoặc analytics không kết nối được, hoặc gợi ý `make up`. Xong trong < 30 s (không treo) | ✅ |
| P0-T05-TC12 | Negative | Chỉ analytics lỗi thì kết quả chung vẫn là fail: `$DC down -v && $MAKE up && $DC stop postgres-analytics`; `env PATH=$NOGOPATH $MAKE migrate; echo $?`; `qcore "select count(*) from goose_db_version where version_id=1"`; dọn: `$DC up -d --wait` | Exit **khác 0** (lỗi analytics không bị che). Output nêu `analytics`. Core vẫn được migrate (in `1`) | ✅ |
| P0-T05-TC13 | Negative | Không có goose (A3): `mkdir -p $S/emptygopath`; `env PATH=$NOGOPATH GOPATH=$S/emptygopath $MAKE migrate; echo $?`; sau đó `grep -rn 'goose@latest' Makefile scripts/` | Theo cách (a): exit **khác 0**, thông báo nêu `go install github.com/pressly/goose/v3/cmd/goose@v3.28.0`. Theo cách (b): `$S/emptygopath/bin/goose -version` in `v3.28.0` và migrate exit `0`. `grep` không in gì | ✅ |
| P0-T05-TC14 | Chạy từ thư mục khác | `cd /tmp && $MAKE -C $REPO -n up migrate test; echo $?`; `cd /tmp && env PATH=$NOGOPATH $MAKE -C $REPO migrate; echo $?`; `cd $REPO` | Hai lần exit `0`. Lần 2 in `no migrations to run` cho cả hai DB: chứng minh đường dẫn migration đúng khi chạy từ thư mục khác, dù recipe là lệnh trực tiếp hay gọi script. Chỉ khi recipe là lệnh trực tiếp: đường dẫn in ở lần 1 trỏ đúng `deploy/docker-compose.yml`, `com/tm/server/db/{core,analytics}/migrations` của repo (tương đối sau `-C`, hoặc tuyệt đối dưới `$REPO`). Khi recipe gọi script: đường dẫn script trỏ tới `scripts/` của repo (không phải `/tmp/scripts/...`) | ✅ |
| P0-T05-TC15 | DoD | Clone mới, `make up && make migrate` < 5 phút (DoD, A6). Clone đặt trong scratchpad (`$S/repo`, dưới `/private/tmp/...`, ngoài `~/Documents`, nên không vướng lỗi treo bind mount TCC của P0-T02; compose có `name: snaptix` nên `$DC` ở `$REPO` vẫn thấy đúng project): `$DC down -v`; `mkclone`; `cd $S/repo`; `s=$(date +%s); env PATH=$NOGOPATH $MAKE up && env PATH=$NOGOPATH $MAKE migrate; echo "rc=$? t=$(( $(date +%s)-s ))"`; `$DC ps --format '{{.Service}} {{.Health}}'`; `cd $REPO` | `rc=0`, `t` < `300`. Đủ 8 service `healthy`. Cả hai DB có version `1` (kiểm bằng `qcore`/`qan` như TC08) | ✅ |
| P0-T05-TC16 | Test | `make test` pass khi stack **không** chạy (A4): `$DC down -v`; `git status --porcelain > $S/st0`; `$MAKE test; echo $?`; `docker ps -a --filter label=com.docker.compose.project=snaptix -q \| wc -l`; `git status --porcelain \| diff $S/st0 -` | Exit `0`. Output cho thấy đã chạy `go vet`, `go test -race`, `bazel test //...` (trong `com/tm/server`), `pnpm test` (trong `com/tm/app`), từng `scripts/*_test.sh`, `check-structure.sh` và `check-compose.sh`. Không có container nào (`0`). `diff` không in gì: `make test` không sửa hay tạo file được track/không ignore (lockfile, `BUILD.bazel` giữ nguyên) | ✅ |
| P0-T05-TC17 | Negative | Go test fail thì `make test` fail. Trong clone: `mkclone; cd $S/repo`; tạo `com/tm/server/pkg/tcfail/tcfail_test.go` (`package tcfail` + `func TestFail(t *testing.T){ t.Fatal("tc17") }`); `(cd com/tm/server && bazel run //:gazelle)`; `$MAKE test; echo $?`; `bzoff` (tắt Bazel server của clone); `cd $REPO` | Exit **khác 0**. Output có `tc17` / `FAIL` | ✅ |
| P0-T05-TC18 | Negative | Lỗi ở bộ **chạy cuối** không bị che, và lỗi ở bộ app bị bắt. Trong clone: `mkclone; cd $S/repo`. **(a)** Sửa một `scripts/*_test.sh` để exit 1 ngay đầu file (ví dụ thêm `exit 1` sau dòng shebang của `scripts/check-structure_test.sh`), rồi `$MAKE test; echo $?`. **(b)** Hoàn nguyên (a) (`git checkout -- scripts`), tạo `com/tm/app/packages/tc-fail/package.json` (`{"name":"tc-fail","private":true,"scripts":{"test":"echo tc18-app; exit 1"}}`), `(cd com/tm/app && pnpm install)` (không ghi lịch sử git, chỉ cần working tree), rồi `$MAKE test; echo $?`; `bzoff`; `cd $REPO` | (a) và (b) đều exit **khác 0**. Ở (b), output có `tc18-app`. Chứng minh không bộ nào bị che bởi `;` / `\|\| true` (A4) | ✅ |
| P0-T05-TC19 | Mặc định | Target mặc định không phá dữ liệu, kiểm theo **hành vi** (A5), không dựa vào text của `make -n`. Stack chạy nhưng **chưa** migrate (để phát hiện được cả việc chạy migration): `$DC down -v && $MAKE up`; trong cùng một lần gọi Bash (hàm `snap` ở Chuẩn bị): `B=$(snap)`; `env PATH=$NOGOPATH $MAKE; echo $?` (chạy thật, không `-n`, không đối số, trong `$REPO`); `A=$(snap)`; `[ "$A" = "$B" ] && echo same`; `echo "$B" \| tail -2` | In `same`: container ID, `StartedAt`, danh sách volume không đổi (không `down`/`down -v`/tạo lại container). `tail -2` in `none` cho cả core và analytics, trước và sau như nhau: `make` không đối số không chạy migration (goose tạo `goose_db_version` ngay lần chạy đầu). Exit code của `make` không bắt buộc (target mặc định là `help`, in hướng dẫn hay `up` đều được, miễn không phá dữ liệu). Text hướng dẫn nhắc tới `down`/`migrate` không làm ca này fail | ✅ |
| P0-T05-TC20 | Docs | Docs dùng make và khớp Makefile: `grep -n -E 'make (up\|migrate\|test)' com/tm/docs/technical/local-setup.md`; đọc các lệnh gốc docs ghi là tương đương (vd `docker compose -f deploy/docker-compose.yml up -d --wait`, `goose -dir ... up`, lệnh test) và với mỗi lệnh, đối chiếu bằng `dryhas <target> '<phần đặc trưng của lệnh>'` (tìm trong output `make -n` hoặc trong script mà recipe gọi) | `local-setup.md` hướng dẫn `make up` (bước 1), `make migrate` (bước 2) và `make test` (mục Build & kiểm thử). Lệnh gốc tương đương vẫn còn hoặc được giải thích là việc make (hoặc script make gọi) thực hiện. Ghi chú `PATH`/goose khớp hành vi A3. Mọi lệnh docs nêu là việc của target đều tìm thấy bằng `dryhas` (trong recipe trực tiếp **hoặc** trong script được gọi); không có lệnh nào trong docs mâu thuẫn với Makefile/script (khác file compose, khác thư mục migration, khác cờ `up -d --wait`) | ✅ |
| P0-T05-TC21 | Phạm vi | Không lấn phạm vi: `git status --porcelain -uall` (trước commit) | Chỉ có file trong: `Makefile`, `scripts/*` (script của task kèm `_test.sh`), `com/tm/docs/**` (local-setup, handbook, planning, execute-all-notes, project-structure nếu cần). Không đổi `deploy/`, `com/tm/server/db/**`, `.github/workflows/**`, `com/tm/server/**` code, `com/tm/app/**` | ✅ |
| P0-T05-TC22 | Dọn dẹp | Chạy cuối cùng: `$DC down -v`; `bzoff` (hoặc `(cd $S/repo/com/tm/server && bazel clean --expunge)`); `rm -rf $S/repo`; `docker ps -a --filter label=com.docker.compose.project=snaptix --format '{{.Names}}'`; `docker volume ls --filter label=com.docker.compose.project=snaptix -q` | Hai lệnh `docker` không in gì: không còn container hay volume của project `snaptix` | ✅ |

Test nghiệm thu liên quan:
- **P0-AT01** (Manual): máy sạch, clone repo, chạy `make up && make migrate` thì tất cả container healthy và migration áp dụng thành công (P0-FR1, P0-FR2). TC05, TC08 và TC15 kiểm phần này trên máy dev. Nghiệm thu chính thức chạy khi đóng phase.
- DoD phase 0: "Clone repo mới → `make up && make migrate` chạy thành công trong < 5 phút" (TC15).

## Review

### Review lần 1 — CHANGES_REQUESTED

Reviewer độc lập (review agent, execute-all). Bộ test phủ đủ 3 target, P0-FR1/FR2, DoD "< 5 phút" và P0-AT01. Nhánh lỗi (Docker không dùng được, DB down, chỉ analytics lỗi, thiếu goose, go/pnpm/script fail) hợp lý. Ca cần sửa code chạy trong bản clone tạm. Định dạng đúng: tiêu đề, 5 cột, ID TC01..TC22 liên tục, ⬜, có dòng "Test nghiệm thu liên quan". Có hai lỗi thực chất có thể làm task fail vô lý:

1. **TC19: regex không làm đúng điều nó nói.** `make -n` in cả recipe `echo`/`@echo`, nên text hướng dẫn cũng bị bắt. Reviewer đã thử trong scratchpad: Makefile có `help:` với `@echo "make down: docker compose down -v"` thì `/usr/bin/make -n | grep -E 'compose.* down|goose .*-dir'` **in ra dòng đó**. Một target `help` hợp lệ liệt kê target phụ `down` (A5 cho phép) sẽ làm TC19 fail sai. Sửa theo một trong hai cách:
   - (a) Kiểm theo hành vi: stack đang chạy, đã migrate, rồi chạy `$MAKE` thật (không `-n`, `PATH=$NOGOPATH`). So container ID + `StartedAt` (như TC06) và `count(*)/max(id)` của `goose_db_version` (như TC09) trước/sau: không đổi. Volume vẫn còn.
   - (b) Kiểm theo cấu trúc: lấy tên target mặc định (target đầu tiên, hoặc `.DEFAULT_GOAL`) rồi kiểm prerequisite/recipe của **target đó** không gọi `down` hay `migrate`/goose. Bỏ qua dòng `echo`/`printf`.
   Nên dùng (a) vì đo được và không phụ thuộc cách viết.
2. **TC03, TC14, TC20 mâu thuẫn với phạm vi.** Mục Phạm vi cho phép tách logic ra `scripts/` (vd `scripts/migrate.sh`). Khi đó `make -n migrate` chỉ in `bash scripts/migrate.sh`, không in lệnh `goose -dir .../core/migrations ...`. TC03 (kỳ vọng output có lệnh goose cho cả hai thư mục và "các lệnh test của A4"), TC14 ("đường dẫn trong lệnh trỏ đúng ...") và TC20 ("Không có lệnh nào trong docs khác với `make -n`") sẽ fail với một cách làm hợp lệ. Sửa: kỳ vọng chấp nhận **hoặc** lệnh trực tiếp **hoặc** lời gọi script trong `scripts/`, và khi gọi script thì kiểm nội dung script chứa đúng các lệnh đó (`grep` trong script). TC14 phần chạy thật (lần 2 in `no migrations to run`) đã đủ chứng minh đường dẫn đúng khi chạy từ thư mục khác, nên phần "đường dẫn trong lệnh" chỉ cần áp cho trường hợp lệnh trực tiếp.

Góp ý không chặn:
- TC02: regex đã được thử trên dữ liệu mẫu. Nó bắt đúng `!=`, `::=`, `.ONESHELL`, `private`, `undefine`, `$(file`, `--output-sync`, và không bắt `!=` trong `if [ ... != ... ]` của recipe. Hợp lệ.
- TC07: `DOCKER_HOST=unix:///tmp/khong-co.sock` chỉ đổi đích của CLI, không động tới Docker Desktop. Đây là cách an toàn trên máy dev, giữ nguyên. Chỉ cần Makefile không hard-code `--context`/`-H`.
- TC08: ghi rõ `env PATH=$NOGOPATH bash -c 'command -v goose || echo none'`. `command -v` là builtin nên `env PATH=... command -v` không chạy được.
- TC15: clone nằm trong scratchpad (`/private/tmp/...`), ngoài `~/Documents`. Compose có `name: snaptix` và không bind mount (chỉ build context, do CLI đọc), nên không vướng lỗi treo TCC của P0-T02. `$DC` ở `$REPO` vẫn thấy đúng project. Ổn.
- TC17/TC18/TC22: Bazel trong clone tạo output base và server riêng. Trước `rm -rf $S/repo` nên chạy `(cd $S/repo/com/tm/server && bazel clean --expunge)` hoặc `bazel shutdown`, để không để lại process/đĩa.
- A3, A4, A6 hợp lý và có căn cứ: A3 lấy từ execute-all-notes P0-T03 và local-setup. A4 gộp từ CLAUDE.md bước 4, local-setup và execute-all-notes, nặng vừa phải, không bắt golangci-lint/gazelle. A6 giống cách P0-T02-TC05 đo. A5 do bộ test tự thêm, docs không có, nhưng chi phí thấp và bảo vệ dữ liệu nên chấp nhận, miễn là TC19 sửa như điểm 1.

### Sửa lần 1

Agent sinh test case (chế độ sửa, execute-all) đã xử lý từng điểm của Review lần 1. ID giữ nguyên TC01..TC22, không thêm TC mới.

1. **(Chặn) TC19**: bỏ `make -n | grep`. Đổi sang kiểm theo hành vi, theo phương án (a): chạy `$MAKE` thật, không đối số, `PATH=$NOGOPATH`, trong `$REPO`. So trước/sau bằng hàm `snap`, gồm container ID + `StartedAt`, danh sách volume của project `snaptix`, và `to_regclass('goose_db_version')` của core/analytics. Kết quả mong đợi là không đổi (`same`). Chọn trạng thái stack chạy nhưng **chưa migrate** thay cho "đã migrate". Lý do: trên DB đã migrate, `goose up` không thêm dòng, nên `count(*)/max(id)` không đổi kể cả khi target mặc định chạy migration. Còn DB chưa migrate sẽ có bảng `goose_db_version` ngay khi goose chạy, nên đo được. Text hướng dẫn trong `help` không còn ảnh hưởng. Exit code của `make` không bắt buộc.
2. **(Chặn) TC03, TC14, TC20**: thêm hàm `dryhas <target> <regex>` vào mục Chuẩn bị. Hàm tìm regex trong output `make -n <target>` **hoặc** trong nội dung `scripts/*.sh` mà output đó gọi, theo tối đa 2 cấp script gọi script.
   - TC03: kỳ vọng mọi lời gọi `dryhas` đều khớp: compose file, `up -d --wait`, `goose`, `migrations`, `core`, `analytics`, `go vet`, `go test -race`, `bazel test`, `pnpm`, `_test.sh`, `check-structure.sh`, `check-compose.sh`. Cả lệnh trực tiếp lẫn gọi script đều pass. Regex không bắt buộc chuỗi `db/core/migrations` nguyên văn, vì script có thể lặp qua biến. Việc migrate đúng cả hai DB do TC08 kiểm bằng hành vi.
   - TC14: phần chạy thật (`no migrations to run` cho cả hai DB) là bằng chứng chính cho cả hai cách làm. Kiểm "đường dẫn trong lệnh" chỉ áp dụng khi recipe là lệnh trực tiếp. Khi recipe gọi script, chỉ kiểm đường dẫn script trỏ về `scripts/` của repo.
   - TC20: thay "không khác `make -n`" bằng đối chiếu từng lệnh docs qua `dryhas`. Lệnh có thể nằm trong recipe hoặc trong script được gọi.
3. **Góp ý nhỏ**:
   - TC08: đổi sang `env PATH=$NOGOPATH bash -c 'command -v goose || echo none'`.
   - TC15: ghi rõ clone đặt trong scratchpad (`$S/repo` dưới `/private/tmp/...`, ngoài `~/Documents`). Dòng `S=` ở Chuẩn bị cũng có chú thích này.
   - TC17, TC18, TC22: thêm hàm `bzoff` (`bazel shutdown` trong `$S/repo/com/tm/server`), gọi ở cuối TC17/TC18 và trước `rm -rf` ở TC22 (có thể thay bằng `bazel clean --expunge`). `mkclone` cũng gọi `bzoff` trước khi `rm -rf $S/repo`.
   - TC07: giữ cách `DOCKER_HOST=unix:///tmp/khong-co.sock`, bổ sung vào kỳ vọng: Makefile/script không hard-code `--context`/`-H`.

### Review lần 2 — CHANGES_REQUESTED

Reviewer độc lập (review agent, execute-all). Đã tự kiểm chứng các helper mới trong scratchpad (bash 3.2.57, `/usr/bin/make` 3.81, stack compose thật, đã `down -v` sau khi thử, không còn container/volume).

Điểm của Review lần 1 đã xử lý đúng:
- **TC19**: kiểm theo hành vi là hợp lý và đo được. Đã chạy `snap` trên stack thật chưa migrate: hai lần `snap` liên tiếp cho `same`. `tail -2` in `none`/`none`. Sau khi restart một container, `snap` khác. Sau khi chạy `goose ... up` cho core, dòng core đổi thành `goose_db_version`, `snap` khác. Vậy `snap` bắt được cả tạo lại/restart container, `down -v` (volume) lẫn việc chạy migration. Chọn stack **chưa** migrate là đúng: trên DB đã migrate, `goose up` không để lại dấu vết. Ca này không tránh được một trường hợp: target mặc định là `migrate` nhưng không tìm thấy goose. Trường hợp đó đã được A3 và TC08 loại trừ (bắt buộc tìm trong `$(go env GOPATH)/bin`), nên chấp nhận.
- **TC03/TC14/TC20 + `dryhas`**: đã thử hai Makefile mẫu. (1) Lệnh trực tiếp dùng `$(ROOT)` tuyệt đối, có `help` với `@echo`. (2) Recipe gọi `bash $(ROOT)scripts/up.sh`, `scripts/migrate.sh` (vòng lặp `for db in core analytics`), `./scripts/test.sh`, và `test.sh` gọi tiếp `scripts/test-server.sh` (2 cấp). Cả 13 lời gọi `dryhas` của TC03 đều exit `0` ở cả hai kiểu. Regex không có (đối chứng) exit `1`. Không có lỗi cú pháp bash 3.2 / make 3.81. Đường dẫn tuyệt đối `/.../scripts/x.sh` vẫn được trích đúng thành `scripts/x.sh`.
- Góp ý nhỏ (TC08 `bash -c`, TC15 scratchpad, `bzoff`, TC07) đã xử lý. `bzoff` đã thử: chạy được khi chưa có clone, và tắt đúng Bazel server của clone (`pgrep -f $S/repo`: 1 → 0).
- Định dạng đúng: tiêu đề, 5 cột, ID `P0-T05-TC01..TC22` liên tục, ⬜, có dòng "Test nghiệm thu liên quan".

Lỗi thực chất (chặn):

1. **`mkclone` (TC04, TC15, TC17, TC18) và TC18(b) có `git ... commit` nên bị hook `guard-bash` chặn trong chế độ execute-all.** Reviewer đã chạy thử: mọi lệnh Bash có chữ `git -C $S/repo commit -qm base ...` đều bị từ chối với `[snaptix guard] Commit message phải chứa [<Task ID>] hoặc [phase-N]`. Hook đọc text lệnh, không phân biệt repo thật hay clone tạm. Thêm tag `[P0-T05]` cũng bị chặn ("Đánh [x] bước 6 ... trước khi commit"). Agent chạy test ở bước 5 sẽ bị chặn ở 4 ca, hoặc phải "lách hook" (CLAUDE.md cấm). Commit này cũng không cần thiết. `git -C $S/repo add -A` là đủ, vì `git checkout -- scripts` (TC18) khôi phục từ index. Reviewer đã thử: sửa `scripts/check-structure_test.sh` rồi `git checkout -- scripts` thì sạch lại (0 thay đổi). Sửa:
   - `mkclone`: bỏ lệnh commit, chỉ giữ `git -C $S/repo add -A`.
   - TC18(b): bỏ bước "commit". Nếu vẫn cần mốc để hoàn nguyên thì dùng `git add -A`.
   - Không có TC nào khác dựa vào HEAD của clone. Đã thử `mkclone` không commit: clone đúng working tree (có file chưa commit của task), không có `node_modules`/`bazel-*`.

Góp ý không chặn:
- Mục Chuẩn bị ghi "mọi lệnh chạy trong cùng một phiên bash". Nhưng Bash tool của agent mở shell mới mỗi lần gọi. Nên ghi thêm: lưu phần Chuẩn bị thành `$S/prep.sh` rồi `source $S/prep.sh` ở đầu mỗi lần gọi (biến `A`/`B`/`N0` phải tính trong cùng một lần gọi).
- TC19: `snap` định nghĩa trong ô bảng. Nên chuyển vào Chuẩn bị cùng `qcore`/`qan` cho dễ tái dùng. Không bắt buộc.

### Sửa lần 2

Agent sinh test case (chế độ sửa, execute-all) đã xử lý các điểm của Review lần 2. ID giữ nguyên TC01..TC22, không thêm TC mới.

1. **(Chặn) Bỏ mọi lệnh ghi lịch sử git** (TC04, TC15, TC17, TC18 dùng `mkclone`):
   - `mkclone`: bỏ lệnh tạo commit `base`, chỉ giữ `git -C $S/repo add -A`. Working tree hiện tại nằm trong index của clone, nên `git checkout -- <path>` khôi phục về đúng trạng thái đó.
   - TC18(b): bỏ bước "commit". Chỉ cần working tree (`package.json` mới + `pnpm install`) rồi chạy `$MAKE test`. Hoàn nguyên (a) vẫn dùng `git checkout -- scripts` (khôi phục từ index).
   - Đã rà toàn file: không TC nào dựa vào HEAD/lịch sử của clone. Không còn lệnh commit nào trong Chuẩn bị hay TC. Chuẩn bị ghi rõ lý do (hook `guard-bash` đọc text lệnh).
   - Đã chạy thử `mkclone` mới trong scratchpad bằng `/bin/bash` 3.2.57: exit `0`, clone có file chưa track của task (`tasks/P0-T05/test-cases.md` ở trạng thái `A`). Thêm `exit 1` vào `scripts/check-structure_test.sh` rồi `git checkout -- scripts` thì thay đổi biến mất (chỉ còn 2 file staged từ working tree gốc). `bzoff` exit `0`. Đã xoá clone thử.
2. **Góp ý nhỏ**:
   - Chuẩn bị: ghi thành file `$S/prep.sh`, `source` ở đầu mỗi lần gọi Bash (Bash tool mở shell mới mỗi lần). Biến so sánh trước/sau (`A`/`B`/`N0`/`N1`, `s`) phải tính trong cùng một lần gọi. Áp dụng cho mọi TC.
   - TC19: chuyển hàm `snap` vào Chuẩn bị, ô TC19 chỉ tham chiếu. Nội dung `snap` giữ nguyên (container ID + `StartedAt`, volume, `to_regclass('goose_db_version')` core/analytics).

### Review lần 3 — APPROVED

Reviewer độc lập (review agent, execute-all). Đã tự kiểm chứng trong scratchpad bằng `/bin/bash` 3.2.57 và `/usr/bin/make` 3.81. Không sửa repo, không để lại clone, container hay Bazel server.

Điểm chặn của Review lần 2 đã xử lý:
- **`mkclone` / TC04, TC15, TC17, TC18**: Chuẩn bị và bảng TC không còn lệnh ghi lịch sử git nào. Chữ "commit" chỉ còn trong văn mô tả (TC21 "(trước commit)", phần Review/Sửa). Reviewer trích khối Chuẩn bị thành `$S/prep.sh` (grep `commit` được `0`), `bash -n` hợp lệ, rồi `source` và chạy `mkclone` trong một lần gọi Bash. Hook `guard-bash` không chặn, exit `0`. Clone có file chưa track của task (`A tasks/P0-T05/test-cases.md`), không có `node_modules`/`bazel-*`.
- **TC18(a) hoàn nguyên**: thêm `exit 1` sau shebang của `scripts/check-structure_test.sh` trong clone thì `git status` báo ` M`. Sau `git checkout -- scripts` (exit `0`), `git status --short scripts` và `git diff` đều rỗng. Khôi phục từ index hoạt động đúng. TC18(b) chỉ dùng working tree và `pnpm install`, không cần commit.

Góp ý nhỏ của Review lần 2 đã xử lý:
- Chuẩn bị ghi rõ cách lưu `$S/prep.sh`, `source` ở đầu mỗi lần gọi, và biến trước/sau phải tính trong cùng một lần gọi.
- `snap` đã chuyển vào Chuẩn bị, TC19 chỉ tham chiếu. Đã thử `snap` khi stack không chạy: trả về ngay (rc `1`), không treo. TC19 luôn bật stack trước nên không ảnh hưởng.

Kiểm tra hồi quy:
- `type mkclone bzoff dryhas snap qcore qan` đều có. `bzoff` exit `0`, sau đó `pgrep -f $S/repo` = `0`.
- `dryhas`: thử lại 13 lời gọi của TC03 trên hai Makefile mẫu, (1) lệnh trực tiếp với `$(ROOT)` và `help` có `@echo`, (2) gọi `scripts/up.sh`, `scripts/migrate.sh` (vòng lặp core/analytics) và `./scripts/test.sh` → `scripts/test-server.sh` (2 cấp). Cả hai đều `fail=0`. Regex đối chứng không có trong Makefile exit `1`. Makefile mẫu không có `check-*.sh` nên `grep`/`cat` in cảnh báo ra stderr; repo thật có các file đó nên cảnh báo này không xuất hiện.
- Định dạng: tiêu đề đúng, 22 dòng bảng đều 5 cột, ID `P0-T05-TC01..TC22` liên tục, tất cả ⬜. Dòng "Test nghiệm thu liên quan" (P0-AT01, DoD) vẫn còn. Nội dung các TC khác giữ nguyên như bản đã review ở lần 2.

Góp ý không chặn:
- TC18(a): ví dụ "exit 1 sau dòng shebang" nên làm bằng một lệnh cụ thể chạy được trên BSD sed, vd `sed -i '' '1a\
exit 1
' scripts/check-structure_test.sh`, hoặc dùng Edit tool trong clone. Reviewer đã thử cách này.
