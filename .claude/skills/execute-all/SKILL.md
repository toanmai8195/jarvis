---
name: execute-all
description: Chạy tự động toàn bộ task của snaptix từ task hiện tại đến hết dự án — phase 12, hoặc phase chỉ định (13 là tuỳ chọn) theo quy trình 6 bước trong CLAUDE.md, mỗi task có agent riêng sinh test case và agent riêng review, tự làm hết subtask, tự commit từng task, tự đóng phase và push, cuối cùng verify sản phẩm — không có bước nào chờ người duyệt. Dùng khi người dùng nói "execute-all", "chạy hết", "chạy tự động đến cuối", hoặc "tiếp tục execute-all".
---

# execute-all

Đây là cách **duy nhất** để làm task của snaptix. Điều phối (orchestrator) chạy vòng lặp cho đến khi `ACTION: DONE` hoặc bị chặn thật sự — mục tiêu là **một lần chạy hoàn thành cả dự án**. Quy trình gốc là `CLAUDE.md` (6 bước, không có bước chờ người duyệt): test case + kế hoạch subtask do review agent duyệt, subtask làm liền không dừng, tự commit mỗi task, push khi đóng phase.

Trạng thái suy ra từ planning + git (`planning.py auto next`); `.claude/state/execute-all.json` (gitignore) chỉ lưu running/blocked, số lần thử và việc đang chờ. Stop hook (`auto stop-hook`) không cho kết thúc lượt khi còn việc mà không có subagent đang chờ.

## Bắt đầu / chạy tiếp

```bash
python3 .claude/scripts/planning.py auto start ${ARGUMENTS}      # không có số phase → giữ phase đích đã lưu (mặc định 12); MỖI lần skill được gọi (lần đầu, chạy lại sau khi bị chặn/hết token/session chết)
python3 .claude/scripts/planning.py auto next                     # các vòng sau trong cùng lượt chạy
```

Đọc `ACTION` và làm theo bảng dưới, xong một hành động thì gọi lại `auto next`. Không hỏi người dùng giữa chừng — chỉ dừng ở `DONE` hoặc `BLOCKED`.

**Chạy lại sau khi bị ngắt** (hết token, session chết, watchdog gõ "tiếp tục execute-all"): luôn `auto start` (xoá `pending` và bộ đếm thử cũ). Trạng thái suy ra từ planning + git nên làm tiếp đúng chỗ. Working tree có thể còn thay đổi dở của agent trước — agent mới làm tiếp trên đó, không xoá.

**Nhận "tiếp tục execute-all" khi đang có subagent chạy** (bạn đã spawn, chưa nhận thông báo hoàn tất): **không** spawn thêm, không `auto start` — chỉ kết thúc lượt và chờ thông báo.

## Subagent

Spawn bằng tool `Agent` (`subagent_type: "general-purpose"`, chạy nền). **Ngay trước khi spawn** gọi `auto wait <nhãn>` (đếm số lần thử; exit 1 = hết lượt thử → đã tự chuyển sang blocked, báo người dùng và dừng). Spawn xong thì **kết thúc lượt**; khi có thông báo subagent hoàn tất, đọc kết quả rồi `auto next`. Tại một thời điểm chỉ có **một** subagent của execute-all.

Mọi prompt subagent phải kèm các quy tắc chung:
- Kiểm tra UI (test case UI của task, test nghiệm thu E2E/Manual, mốc demo) trên **Chrome đang mở của người dùng** qua Claude in Chrome (tool `mcp__claude-in-chrome__*`, session chạy với `claude --chrome`): mở tab mới trong tab group của Claude, **chỉ** truy cập `localhost`/`127.0.0.1`, không đọc/thao tác tab khác, không đăng nhập tài khoản thật (dùng mock OIDC), đóng tab khi xong. Chụp màn hình làm bằng chứng khi test case yêu cầu.
- Chrome không kết nối được (extension chưa bật, Chrome đóng) → thử lại một lần; vẫn lỗi thì `auto block "Claude in Chrome không kết nối: ..."` — không tự chuyển sang headless.
- Cần hai phiên người dùng khác nhau cùng lúc ("hai trình duyệt") → phiên A trên Chrome đang mở, phiên B bằng Playwright (`channel: 'chrome'`, context riêng) vì các tab Chrome dùng chung cookie.
- Bộ E2E tự động của repo (Playwright, chạy trong CI — phase 12) vẫn chạy headless; Chrome đang mở chỉ dùng để kiểm tra/nghiệm thu trên máy dev.
- Lệnh chạy lâu (> 5 phút: k6, soak, docker build lớn) chạy nền (`run_in_background`) và chờ bằng Monitor, không chạy foreground.
- Không `git checkout`/`switch` ở working tree chính; cần nhánh khác thì `git worktree add` vào scratchpad.
- Thiếu công cụ CLI → `brew install <tool>`; không cài được → báo lỗi thật.

Prompt cho subagent phải tự đủ (subagent không thấy hội thoại này): Task ID, đường dẫn README phase, file test case, `CLAUDE.md`, `project-structure.md`, việc cần làm, việc **không** được làm, và định dạng báo cáo cuối. Mỗi lần spawn là agent mới — không dùng lại agent đã sinh test case để review.

## Hành động

| ACTION | Làm gì |
|---|---|
| `START` | Có `PROBLEM` → xử lý nếu là việc hành chính của chính execute-all (vd tree bẩn do file state — không được xảy ra vì đã gitignore); còn lại `auto block "<problem>"` và dừng. Không có → `mark start <ID> "<ID>-TC01..TCnn"`; phase ⬜ thì `mark phase <N> 🟨`. |
| `GEN_TC` | `auto wait gen-tc` → spawn **agent sinh test case** (prompt A). |
| `APPROVE_TC` | Đọc mục `## Review` cuối `tasks/<ID>/test-cases.md`, xem mục con cuối cùng: **không có** hoặc `### Sửa lần k` → `auto wait review-tc` → spawn **agent review** mới (prompt B). `### Review lần k — CHANGES_REQUESTED` → `auto wait fix-tc` → spawn agent sinh test case mới ở chế độ sửa (prompt A, kèm yêu cầu review). `### Review lần k — APPROVED` → sửa dải ID cho khớp file rồi `mark step <ID> 1 "Test case: <ID>-TC01..TCnn + kế hoạch subtask — đã được duyệt (review agent, execute-all)"`, rồi `mark subs <ID> "<subtask 1>" "<subtask 2>" ...` chép đúng các subtask trong `## Kế hoạch subtask` thành checklist 2.1, 2.2... |
| `EXECUTE` (bước 2–5) | `auto wait impl` → spawn **agent triển khai** (prompt C). Khi xong: tự chạy lại lệnh bước 4 cho phạm vi bị ảnh hưởng và `planning.py step <ID>` phải báo bước 6. Không khớp → `auto next` (sẽ spawn lại, tối đa số lần thử). |
| `COMMIT` | Xem `git status`/`git diff --stat`, soạn message Conventional Commits có `[<ID>]` (+ tag challenge đã đạt). `mark step <ID> 6 "Commit: \`<message>\` · Push: không (execute-all push khi đóng phase)"`, rồi `git add -A && git commit -m "<message>"` kèm dòng attribution. Tree phải sạch sau commit. Có `RESUME: True` (bước 6 đã tick nhưng chưa commit do bị ngắt) → **không** `mark` lại, lấy message ghi ở dòng bước 6 rồi `git add -A && git commit`. |
| `COMMIT_CLOSE` | Phase đã ✅ nhưng chưa có commit đóng phase (bị ngắt) → `git add -A && git commit -m "docs(planning): đóng phase <N> [phase-<N>]"`. |
| `CLOSE_PHASE` | Có `PROBLEM` → `auto wait close-<N>` → spawn **agent đóng phase** (prompt D). Xong → `auto close-check <N>`; đạt → `auto close <N>` rồi commit `docs(planning): đóng phase <N> [phase-<N>]`. Trong lúc đóng phase hook cho phép sửa code và push `main`/`ci-check/*` để kiểm chứng CI. |
| `PUSH` | `git push` (không force, không `--no-verify`). Lỗi xác thực/mạng → `auto block "push lỗi: ..."`. |
| `FINAL_VERIFY` | `auto wait final` → spawn **agent verify cuối** (prompt E). Đạt → `auto done`, rồi báo cáo cuối cho người dùng. Không đạt → sửa qua agent triển khai (ghi rõ lỗi) hoặc `auto block`. |
| `DONE` | Báo cáo cuối (xem dưới) rồi dừng. |
| `BLOCKED` | Báo lý do + cách gỡ, dừng. Người dùng gỡ xong gọi lại skill → `auto start`. |

## Prompt mẫu

**A — sinh test case** (mới hoặc sửa): đọc `CLAUDE.md` (Bước 1), README phase (requirement, challenge, DoD, dòng task), `acceptance-tests.md`, `project-structure.md`, docs liên quan (`api.md`, `database.md`...). Viết/sửa `tasks/<ID>/test-cases.md` đúng định dạng: tiêu đề, bảng `ID | Loại | Kịch bản | Kết quả mong đợi | Trạng thái`, ID `<ID>-TCnn`, trạng thái ⬜, chỉ trong phạm vi task, có nhánh lỗi và biên, mỗi kết quả kiểm chứng được (lệnh/HTTP/SQL cụ thể); dòng "Test nghiệm thu liên quan"; mục `## Kế hoạch subtask` (Bước 1 mục 3 của `CLAUDE.md`): các bước nhỏ, mỗi bước chạy được và thêm **một** khái niệm mới, ghi làm gì, file nào, kiến thức Go/PG/Node/React mới và lý do chọn cách đó. Chế độ sửa: xử lý từng điểm của review gần nhất, thêm `### Sửa lần k` vào `## Review` liệt kê đã sửa gì. Không sửa file nào khác, không tick checklist, không code. Báo cáo: dải ID + số test case.

**B — review test case** (agent độc lập): đọc cùng tài liệu như A, **không** nhận lập luận của agent sinh. Kiểm tra: phủ đủ phạm vi dòng task và challenge của task; không vượt phạm vi; nhánh lỗi/biên/đồng thời khi task có; kết quả mong đợi đo được, không mơ hồ; khớp docs/ADR; đúng định dạng và ID; tiền không dùng số thực; kế hoạch subtask đủ, đúng thứ tự, mỗi subtask nhỏ và chạy được, phủ hết phạm vi task. Chỉ được thêm vào cuối file mục `## Review` (tạo nếu chưa có) một mục `### Review lần k — APPROVED` hoặc `### Review lần k — CHANGES_REQUESTED` kèm danh sách điểm cần sửa (cụ thể, có ID). Không sửa bảng. Báo cáo: verdict + tóm tắt.

**C — triển khai** (bước 2–5 của một task): làm đúng Bước 2→5 trong `CLAUDE.md`. Bước 2: làm **lần lượt từng subtask chưa `[x]`** theo checklist 2.x (code → chạy thử chứng minh hoạt động → `mark sub <ID> <k>`), không dừng chờ ai, không gộp subtask; xong hết → `mark step <ID> 2`. Đánh dấu bằng `python3 .claude/scripts/planning.py mark step <ID> <n>` ngay khi xong từng bước, `mark tc <ID>` khi mọi test case pass; handbook `handbook/phase-<N>/<ID>.md` + 2 bảng trong `handbook/README.md`; cập nhật challenge, acceptance test liên quan, docs nếu code khác thiết kế. Không commit/push, không sửa test case đã duyệt, không skip test, không lách hook. Việc ngoài phạm vi → ghi vào `com/tm/docs/technical/planning/execute-all-notes.md` (mục theo Task ID), không làm. Build/test fail không tự sửa được sau 3 vòng → dừng và báo lỗi thật. Báo cáo: subtask đã làm, file đã đổi, output tóm tắt lệnh bước 4, bảng test case pass/fail.

**D — đóng phase**: đọc README phase (DoD, Checklist đóng phase) và `acceptance-tests.md`. Chạy từng test nghiệm thu chưa ✅ (tự động, hoặc manual theo hướng dẫn; UI trên Chrome đang mở qua Claude in Chrome, chỉ localhost; "hai trình duyệt" = Chrome đang mở + một context Playwright `channel: 'chrome'`). Pass → ✅. Test/DoD cần CI GitHub: commit hiện tại phải được push (`git push origin main`, được phép khi đang đóng phase) rồi `gh run watch`/`gh run list` xác nhận xanh; test "mở PR có lỗi" → `git worktree add <scratchpad>/ci -b ci-check/<AT-ID> origin/main`, commit lỗi cố ý (message chứa `[phase-<N>]`), `git push -u origin ci-check/<AT-ID>`, `gh pr create --draft`, `gh pr checks --watch`, ghi kết quả, rồi `gh pr close --delete-branch` và `git worktree remove`. Chỉ đánh ⚠️ khi **không thể** chạy tự động (vd cần tài khoản thật) và ghi lý do ở `## Miễn trừ` cuối file. Fail vì lỗi code → sửa trong phạm vi task gốc, chạy lại build/test, ghi vào `execute-all-notes.md`. Tick DoD và checklist đóng phase đã thực sự đạt; **không** viết `lessons-learned.md` (người dùng tự viết sau, không chặn đóng phase). Không commit.

**E — verify cuối**: `docker compose up` toàn hệ thống; chạy lại toàn bộ lệnh Bước 4 (server + app); chạy E2E; đi qua mốc demo phase 0–<until> trên Chrome đang mở (Claude in Chrome, localhost). Ghi kết quả vào `execute-all-notes.md` mục `## Verify cuối`. Không commit.

## Báo cáo cuối (DONE)

- Task/phase đã xong, số commit, đã push chưa.
- Test nghiệm thu miễn trừ (⚠️) và lý do.
- Nội dung `execute-all-notes.md`: việc ngoài phạm vi, lỗi đã sửa ngoài task.
- Nhắc người dùng viết `lessons-learned.md` cho từng phase đã đóng.

## Không được

- Dùng lại một agent vừa sinh test case để tự review.
- Hỏi người dùng hoặc dừng chờ duyệt ở bất kỳ bước nào (trừ `BLOCKED` thật sự).
- Code khi bước 1 chưa `[x]`; lách hook `snaptix guard`/`guard-bash` bằng Bash.
- Sửa/skip test để pass; đánh ✅/`[x]` khi chưa chạy.
- Force push, `--no-verify`, `--amend`; push ngoài: commit đóng phase, lúc đóng phase (kiểm chứng CI), nhánh `ci-check/*`.
- Spawn subagent thứ hai khi subagent trước chưa báo xong.
- Coi `BLOCKED` là xong; tự `auto done` khi verify cuối chưa đạt.
