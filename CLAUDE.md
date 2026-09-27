# snaptix — Quy tắc cho Claude

Hệ thống đặt vé xe khách / tàu điện, dự án học tập để lên senior Go, PostgreSQL, Node.js, React.

**Mục tiêu: một lần chạy `/execute-all` hoàn thành cả dự án, không có bước nào chờ người duyệt.** Đây là dự án đầu tiên của người dùng với Go, PG, Node, Mongo, React: vẫn chia code thành subtask nhỏ và ghi handbook kỹ, để người dùng đọc lại theo commit và handbook sau khi chạy xong.

| Thư mục | Nội dung | Build / test |
|---|---|---|
| `com/tm/server` | Go: core, stats-worker, `pkg/`, migration | `go test -race ./...`, `bazel run //:gazelle`, `bazel test //...` |
| `com/tm/app` | Node (Fastify BFF) + React (Vite) | `pnpm --filter <app> lint test build` |
| `com/tm/docs` | Tài liệu, planning, handbook, ADR | — |

Tài liệu quan trọng:
- Planning & task: `com/tm/docs/technical/planning/` — mỗi phase có `README.md` (requirement, task, challenge, DoD), `acceptance-tests.md` (test nghiệm thu phase), `tasks/<Task ID>/test-cases.md` (test case của từng task), `lessons-learned.md`
- Handbook kỹ thuật theo task: `com/tm/docs/technical/handbook/`
- Quy ước code: `com/tm/docs/technical/project-structure.md` (mục "Go cho người từ Java", quy tắc 3 tầng dùng chung)
- Quyết định kiến trúc: `com/tm/docs/technical/adr/`

---

## Quy trình làm một task — BẮT BUỘC, theo đúng thứ tự

Task được gọi bằng ID, ví dụ `P3-T05`. Không bỏ bước, không đảo thứ tự. Mọi task chạy qua skill `execute-all` (xem [Chế độ execute-all](#chế-độ-execute-all)); không có bước nào dừng hỏi người dùng.

### Bước 0 — Validate trước khi làm

Kiểm tra tất cả, thiếu một điều kiện → không làm task; tự khắc phục nếu là việc của execute-all, không thì `auto block "<lý do>"`:

1. **Task liền trước** (trong cùng phase, hoặc task cuối của phase trước) đã:
   - `[x]` ở dòng task và đủ `[x]` cả 6 bước trong checklist con;
   - đã commit: `git log --grep "[<Task ID>]"` có commit của task đó.
2. Phase trước đã đóng (✅ trong `planning/README.md`) nếu đây là task đầu của phase.
3. Working tree sạch (`git status` không có thay đổi chưa commit của task khác).
4. Task không mâu thuẫn với docs/ADR; thiếu thông tin → chọn phương án hợp lý nhất theo docs, ghi quyết định vào `planning/execute-all-notes.md`, không hỏi.

> **Được enforce bằng hook**: `.claude/settings.json` chạy `python3 .claude/scripts/planning.py guard` trước mọi Edit/Write vào `com/tm/server/**` và `com/tm/app/**`. Hook từ chối nếu task trước chưa đủ checklist + commit, phase trước chưa đóng, task hiện tại chưa có checklist con, hoặc bước 1 (review agent duyệt test case) chưa `[x]`. Hook `guard-bash` chặn commit/push sai quy trình, Stop hook không cho dừng khi execute-all còn việc. Không lách hook bằng Bash (`sed`, heredoc...) — hook bị chặn nghĩa là quy trình chưa đúng. Kiểm tra nhanh: `python3 .claude/scripts/planning.py validate`.

Đạt → đọc `README.md` của phase (mục tiêu, requirement, challenge trong `[...]`, DoD), rồi **thêm checklist con** ngay dưới dòng task:

```markdown
- [ ] **P3-T05** <mô tả task>
  - [ ] 1. Test case: P3-T05-TC01..TCnn + kế hoạch subtask — đã được duyệt
  - [ ] 2. Code
    - [ ] 2.1 <subtask nhỏ, vd service rỗng in hello world>
    - [ ] 2.2 <subtask kế tiếp, vd thêm HTTP server>
  - [ ] 3. Unit test
  - [ ] 4. Build + unit test pass
  - [ ] 5. Test case pass + handbook
  - [ ] 6. Commit: `<type(scope): mô tả [Task ID]>` · Push: không (execute-all push khi đóng phase)
```

Đánh `[x]` từng bước ngay khi bước đó xong. Phase đang ⬜ → 🟨 khi bắt đầu task đầu tiên.

### Bước 1 — Gen test case → review agent duyệt

**Mỗi task một bộ test case riêng, trong thư mục riêng.**

| Loại | File | ID | Khi nào viết | Dùng để |
|---|---|---|---|---|
| **Test case theo task** | `planning/phase-<N>-*/tasks/<Task ID>/test-cases.md` | `<Task ID>-TCnn`, vd `P0-T01-TC01` | Bước 1 của task | Nghiệm thu **task** (bước 5) |
| **Test nghiệm thu phase** | `planning/phase-<N>-*/acceptance-tests.md` | `P<N>-ATnn`, vd `P4-AT13` | Có sẵn trong planning | Nghiệm thu **phase** (DoD) |

1. Tạo `tasks/<Task ID>/test-cases.md` (tiêu đề `# Test cases — <Task ID>: <tên task>`), bảng cột `ID | Loại | Kịch bản | Kết quả mong đợi | Trạng thái`, mỗi test case gồm: ID `<Task ID>-TCnn` (đánh số từ `TC01` trong task), loại, kịch bản, kết quả mong đợi, trạng thái ⬜. Test case phải kiểm chứng đúng phạm vi của task, không mượn ID của task hay phase khác.
2. Nếu task góp phần làm pass test nghiệm thu (`P<N>-ATnn`) nào, nêu thêm dòng "Test nghiệm thu liên quan: ..." dưới bảng — để biết, không thay cho test case của task.
3. Viết **kế hoạch subtask** cho bước 2 vào `tasks/<Task ID>/test-cases.md` (mục `## Kế hoạch subtask`): chia code thành các bước nhỏ, mỗi bước chạy được và thêm **một** khái niệm mới (vd: service rỗng in hello world → thêm HTTP server → thêm router → thêm DAO → thêm handler → thêm service → cập nhật docker). Mỗi subtask ghi: làm gì, file nào, **kiến thức Go/PG/Node/React mới** và lý do chọn cách đó.
4. Một agent **khác, độc lập** review bộ test case + kế hoạch subtask, ghi `### Review lần k — APPROVED` hoặc `— CHANGES_REQUESTED` vào mục `## Review` cuối file. Có yêu cầu sửa → agent sinh sửa, ghi `### Sửa lần k`, rồi review lại.
5. `APPROVED` → đánh `[x]` bước 1 (ghi dải ID test case và `(review agent, execute-all)`), chép các subtask thành checklist `2.1`, `2.2`... dưới bước 2.

Test case đã duyệt là tiêu chí nghiệm thu của task: muốn thêm/sửa/xoá sau đó phải qua review agent lại.

### Bước 2 — Code, từng subtask một

Với **mỗi** subtask theo thứ tự:
1. Code đúng phạm vi subtask (nhỏ, chạy được).
2. Chạy thử để chứng minh nó hoạt động (build, `go run`, `curl`...).
3. Chạy được → đánh `[x]` subtask đó, sang subtask tiếp theo ngay (không dừng). Không gộp nhiều subtask làm một.

Đủ các subtask → đánh `[x]` bước 2.

- Theo `project-structure.md`: package theo nghiệp vụ, interface phía dùng, wiring tay, không DI framework; Fastify plugin, không NestJS.
- Chỉ làm trong phạm vi task. Việc ngoài phạm vi → ghi vào `planning/execute-all-notes.md`, không tự làm.
- Thêm/xoá file Go hoặc đổi import → `bazel run //:gazelle`.

### Bước 3 — Agent tự viết unit test

- Unit test cho code vừa viết: logic thuần, nhánh lỗi, biên. Go: table-driven. TS: Vitest / Testing Library.
- Test cần DB/Docker thật → integration test với testcontainers, gắn tag theo `project-structure.md`.

### Bước 4 — Build và chạy lại unit test

Chạy đủ, không chỉ test mới:
- Server: `bazel run //:gazelle`, `go vet ./...`, `go test -race ./...`, `bazel test` cho target bị ảnh hưởng.
- App: `pnpm --filter "...[origin/main]" lint test build`.

Fail → sửa code (không sửa/skip test cho pass) → chạy lại đến khi pass. Báo kết quả thật kèm output tóm tắt.

### Bước 5 — Test theo test case + ghi handbook

1. Thực hiện từng test case **của task** đã duyệt ở bước 1 (tự động hoặc theo hướng dẫn manual). Pass → ⬜ → ✅ trong `tasks/<Task ID>/test-cases.md`. Có test case fail → quay lại bước 2.
   Test nghiệm thu liên quan (`acceptance-tests.md`) nếu đã chạy được và pass → ✅; chưa chạy được (cần task sau) → để ⬜.
2. Ghi **handbook** cho task theo mục [Handbook](#handbook) bên dưới.
3. Cập nhật challenge (⬜ → 🟨 / ✅ khi đạt tiêu chí "Hoàn thành khi").
4. Code khác thiết kế trong docs (schema, API, luồng) → cập nhật docs.
5. Đánh `[x]` bước 5 **và** `[x]` dòng task.
6. Task cuối của phase → execute-all tự đóng phase (test nghiệm thu, DoD, checklist đóng phase; `lessons-learned.md` để người dùng tự viết sau).

### Bước 6 — Tự commit

1. Soạn commit message Conventional Commits có `[<Task ID>]` (+ tag challenge đã đạt).
2. **Trước khi commit**, đánh `[x]` bước 6 và ghi commit message + `Push: không` vào dòng đó, rồi commit tất cả trong **một** commit. Hash tra bằng `git log --grep "[<Task ID>]"`, không ghi hash vào file (tránh để lại thay đổi chưa commit).
3. Không push ở bước này — execute-all chỉ push khi đóng phase. Working tree phải sạch sau commit.

---

## Chế độ execute-all

Người dùng đã quyết định (2026-09-27): toàn bộ dự án chạy tự động bằng skill `execute-all`, từ task hiện tại đến hết phase 12 (phase 13 tuỳ chọn, chạy khi gọi `/execute-all 13`), **không có bước chờ người duyệt**.

- **Bước 1**: agent riêng sinh test case + kế hoạch subtask, agent **khác, độc lập** review thay người dùng.
- **Bước 2**: làm hết subtask liền nhau, mỗi subtask phải chạy thử được trước khi tick.
- **Bước 6**: tự commit mỗi task, `Push: không`. Chỉ push: ngay sau commit đóng phase `docs(planning): đóng phase N [phase-N]`; trong lúc đóng phase để kiểm chứng CI trên `main`; nhánh `ci-check/*` cho test nghiệm thu CI (xoá sau khi xong). Không force push, không `--no-verify`, không amend.
- **Đóng phase**: agent tự chạy test nghiệm thu, DoD, checklist đóng phase — trừ `lessons-learned.md` (người dùng tự viết sau, không chặn đóng phase). Test nghiệm thu không thể chạy tự động → ⚠️ kèm lý do ở mục `## Miễn trừ` cuối `acceptance-tests.md`.
- **Chỉ tiêu tải** (phase 11): máy dev không đạt được mục tiêu production → ghi số đo thực tế, chỉ tiêu đã hạ cho máy dev và lý do vào `## Miễn trừ`.
- **Việc ngoài phạm vi / quyết định thiếu thông tin**: không hỏi, ghi vào `planning/execute-all-notes.md` để báo cáo cuối.
- **Chạy lại sau khi bị ngắt** (hết token, session chết): `.claude/scripts/execute-all-watchdog.sh start` chạy claude + watchdog trong tmux session `snaptix`; watchdog gõ "tiếp tục execute-all"; skill luôn `auto start` rồi làm tiếp từ trạng thái planning + git.
- Bị chặn thật sự (cần người: đăng nhập, quyền, hạ tầng) → `auto block "<lý do>"` và dừng; người dùng gỡ xong gọi lại `/execute-all`.

---

## Handbook

`com/tm/docs/technical/handbook/` — ghi chú kỹ thuật **theo task**, do agent viết ở bước 5, để người dùng ôn lại kiến thức đã áp dụng.

- **Mỗi task một file**: `handbook/phase-<N>/<Task ID>.md`, tiêu đề `# Handbook — <Task ID>: <tên task>`, mỗi bài học là một mục `## <tên bài học>`.
- Mỗi bài học gồm:
  - **Chủ đề** (tag): ví dụ `go/channel`, `go/errgroup`, `go/context`, `pg/lock`, `pg/isolation`, `pg/index`, `node/event-loop`, `node/stream`, `react/useState`, `react/memo`...
  - **Bối cảnh**: vấn đề gặp trong task.
  - **Cách làm & lý do**: đã dùng kỹ thuật gì, vì sao chọn nó, phương án khác bị loại vì sao.
  - **Bẫy / lưu ý**: lỗi dễ mắc, điều bất ngờ gặp phải.
  - **Code**: link tương đối tới file + dải dòng + tên hàm/type, ví dụ `[booking/service.go#L40-L62 — Service.Confirm](../../../../server/services/core/internal/booking/service.go#L40-L62)` (từ `handbook/phase-<N>/`). Luôn ghi tên symbol để tìm lại được khi dòng thay đổi.
  - **Tham khảo** (nếu có): tài liệu chính thức, bài viết.
- Chỉ ghi điều thực sự đã làm và học trong task, cụ thể, ngắn gọn. Không chép lại lý thuyết chung chung.
- Thêm dòng vào bảng "Theo task" và bảng "Chỉ mục theo chủ đề" trong `handbook/README.md`.

`lessons-learned.md` của phase là phần **người dùng tự viết** — agent không viết thay, chỉ gợi ý khi được yêu cầu.

---

## Không được

- Bỏ qua bước 0 hoặc làm task khi task trước chưa đủ checklist + commit.
- Code trước khi test case + kế hoạch subtask được review agent duyệt.
- Gộp nhiều subtask vào một lần; tick subtask chưa chạy thử được.
- Dừng hỏi người dùng giữa chừng (trừ `auto block` khi bị chặn thật sự).
- Đánh `[x]` / ✅ khi chưa chạy hoặc còn test fail.
- Sửa, xoá, `skip` test để cho pass; sửa test case đã duyệt mà không xin duyệt lại.
- Push ngoài các trường hợp ở [Chế độ execute-all](#chế-độ-execute-all); force push, `--no-verify`, amend.
- Viết Go theo kiểu Java (`service/`, `repository/`, `IFoo`/`FooImpl`, DI framework, `utils/`).
- Dùng số thực cho tiền ở bất kỳ tầng nào.

## Commit

Conventional Commits, **bắt buộc** ghi ID task (và challenge nếu có): `feat(core): hold seats with conditional update [P4-T03][P1]`.
