---
name: execute-all
description: Chạy tự động toàn bộ task của snaptix từ task hiện tại đến hết phase 8 (hoặc phase chỉ định) theo quy trình 6 bước trong CLAUDE.md, mỗi task có agent riêng sinh test case và agent riêng review, tự commit từng task, tự đóng phase và push, cuối cùng verify sản phẩm. Dùng khi người dùng nói "execute-all", "chạy hết", "chạy tự động đến cuối", hoặc "tiếp tục execute-all".
---

# execute-all

Điều phối (orchestrator) chạy vòng lặp cho đến khi `ACTION: DONE` hoặc bị chặn thật sự. Quy trình gốc vẫn là `CLAUDE.md`; chế độ này chỉ dùng các ngoại lệ ghi trong mục **"Chế độ execute-all"** của `CLAUDE.md` (duyệt test case bằng review agent, tự commit, push khi đóng phase). Mọi quy tắc khác giữ nguyên.

Trạng thái suy ra từ planning + git (`planning.py auto next`); `.claude/state/execute-all.json` (gitignore) chỉ lưu running/blocked, số lần thử và việc đang chờ. Stop hook (`auto stop-hook`) không cho kết thúc lượt khi còn việc mà không có subagent đang chờ.

## Bắt đầu / chạy tiếp

```bash
python3 .claude/scripts/planning.py auto start ${ARGUMENTS:-8}   # lần đầu hoặc chạy lại sau khi bị chặn
python3 .claude/scripts/planning.py auto next                     # các vòng sau
```

Đọc `ACTION` và làm theo bảng dưới, xong một hành động thì gọi lại `auto next`. Không hỏi người dùng giữa chừng — chỉ dừng ở `DONE` hoặc `BLOCKED`.

## Subagent

Spawn bằng tool `Agent` (`subagent_type: "general-purpose"`, chạy nền). **Ngay trước khi spawn** gọi `auto wait <nhãn>` (đếm số lần thử; exit 1 = hết lượt thử → đã tự chuyển sang blocked, báo người dùng và dừng). Spawn xong thì **kết thúc lượt**; khi có thông báo subagent hoàn tất, đọc kết quả rồi `auto next`.

Prompt cho subagent phải tự đủ (subagent không thấy hội thoại này): Task ID, đường dẫn README phase, file test case, `CLAUDE.md`, `project-structure.md`, việc cần làm, việc **không** được làm, và định dạng báo cáo cuối. Mỗi lần spawn là agent mới — không dùng lại agent đã sinh test case để review.

## Hành động

| ACTION | Làm gì |
|---|---|
| `START` | Có `PROBLEM` → xử lý nếu là việc hành chính của chính execute-all (vd tree bẩn do file state — không được xảy ra vì đã gitignore); còn lại `auto block "<problem>"` và dừng. Không có → `mark start <ID> "<ID>-TC01..TCnn"`; phase ⬜ thì `mark phase <N> 🟨`. |
| `GEN_TC` | `auto wait gen-tc` → spawn **agent sinh test case** (prompt A). |
| `APPROVE_TC` | Đọc mục `## Review` cuối `tasks/<ID>/test-cases.md`, xem mục con cuối cùng: **không có** hoặc `### Sửa lần k` → `auto wait review-tc` → spawn **agent review** mới (prompt B). `### Review lần k — CHANGES_REQUESTED` → `auto wait fix-tc` → spawn agent sinh test case mới ở chế độ sửa (prompt A, kèm yêu cầu review). `### Review lần k — APPROVED` → sửa dải ID cho khớp file rồi `mark step <ID> 1 "Test case: <ID>-TC01..TCnn — đã được duyệt (review agent, execute-all)"`. |
| `EXECUTE` (bước 2–5) | `auto wait impl` → spawn **agent triển khai** (prompt C). Khi xong: tự chạy lại lệnh bước 4 cho phạm vi bị ảnh hưởng và `planning.py step <ID>` phải báo bước 6. Không khớp → `auto next` (sẽ spawn lại, tối đa số lần thử). |
| `COMMIT` | Xem `git status`/`git diff --stat`, soạn message Conventional Commits có `[<ID>]` (+ tag challenge đã đạt). `mark step <ID> 6 "Commit: \`<message>\` · Push: không (execute-all push khi đóng phase)"`, rồi `git add -A && git commit -m "<message>"` kèm dòng attribution. Tree phải sạch sau commit. |
| `CLOSE_PHASE` | Có `PROBLEM` → `auto wait close-<N>` → spawn **agent đóng phase** (prompt D). Xong → `auto close-check <N>`; đạt → `auto close <N>` rồi commit `docs(planning): đóng phase <N> [phase-<N>]`. |
| `PUSH` | `git push` (không force, không `--no-verify`). Lỗi xác thực/mạng → `auto block "push lỗi: ..."`. |
| `FINAL_VERIFY` | `auto wait final` → spawn **agent verify cuối** (prompt E). Đạt → `auto done`, rồi báo cáo cuối cho người dùng. Không đạt → sửa qua agent triển khai (ghi rõ lỗi) hoặc `auto block`. |
| `DONE` | Báo cáo cuối (xem dưới) rồi dừng. |
| `BLOCKED` | Báo lý do + cách gỡ, dừng. Người dùng gỡ xong gọi lại skill → `auto start`. |

## Prompt mẫu

**A — sinh test case** (mới hoặc sửa): đọc `CLAUDE.md` (Bước 1), README phase (requirement, challenge, DoD, dòng task), `acceptance-tests.md`, `project-structure.md`, docs liên quan (`api.md`, `database.md`...). Viết/sửa `tasks/<ID>/test-cases.md` đúng định dạng: tiêu đề, bảng `ID | Loại | Kịch bản | Kết quả mong đợi | Trạng thái`, ID `<ID>-TCnn`, trạng thái ⬜, chỉ trong phạm vi task, có nhánh lỗi và biên, mỗi kết quả kiểm chứng được (lệnh/HTTP/SQL cụ thể); dòng "Test nghiệm thu liên quan". Chế độ sửa: xử lý từng điểm của review gần nhất, thêm `### Sửa lần k` vào `## Review` liệt kê đã sửa gì. Không sửa file nào khác, không tick checklist, không code. Báo cáo: dải ID + số test case.

**B — review test case** (agent độc lập): đọc cùng tài liệu như A, **không** nhận lập luận của agent sinh. Kiểm tra: phủ đủ phạm vi dòng task và challenge của task; không vượt phạm vi; nhánh lỗi/biên/đồng thời khi task có; kết quả mong đợi đo được, không mơ hồ; khớp docs/ADR; đúng định dạng và ID; tiền không dùng số thực. Chỉ được thêm vào cuối file mục `## Review` (tạo nếu chưa có) một mục `### Review lần k — APPROVED` hoặc `### Review lần k — CHANGES_REQUESTED` kèm danh sách điểm cần sửa (cụ thể, có ID). Không sửa bảng. Báo cáo: verdict + tóm tắt.

**C — triển khai** (bước 2–5 của một task): làm đúng Bước 2→5 trong `CLAUDE.md`, đánh dấu bằng `python3 .claude/scripts/planning.py mark step <ID> <n>` ngay khi xong từng bước, `mark tc <ID>` khi mọi test case pass; handbook `handbook/phase-<N>/<ID>.md` + 2 bảng trong `handbook/README.md`; cập nhật challenge, acceptance test liên quan, docs nếu code khác thiết kế. Không commit/push, không sửa test case đã duyệt, không skip test, không lách hook. Việc ngoài phạm vi → ghi vào `com/tm/docs/technical/planning/execute-all-notes.md` (mục theo Task ID), không làm. Build/test fail không tự sửa được sau 3 vòng → dừng và báo lỗi thật. Báo cáo: file đã đổi, output tóm tắt lệnh bước 4, bảng test case pass/fail.

**D — đóng phase**: đọc README phase (DoD, Checklist đóng phase) và `acceptance-tests.md`. Chạy từng test nghiệm thu chưa ✅ (tự động, hoặc manual theo hướng dẫn; test trên trình duyệt chỉ với localhost). Pass → ✅. Chỉ đánh ⚠️ khi **không thể** chạy tự động (vd cần tài khoản thật) và ghi lý do ở `## Miễn trừ` cuối file. Fail vì lỗi code → sửa trong phạm vi task gốc, chạy lại build/test, ghi vào `execute-all-notes.md`. Tick DoD và checklist đóng phase đã thực sự đạt; **không** viết `lessons-learned.md` (người dùng tự viết). Không commit.

**E — verify cuối**: `docker compose up` toàn hệ thống; chạy lại toàn bộ lệnh Bước 4 (server + app); chạy E2E; đi qua mốc demo phase 0–<until> (UI trên localhost). Ghi kết quả vào `execute-all-notes.md` mục `## Verify cuối`. Không commit.

## Báo cáo cuối (DONE)

- Task/phase đã xong, số commit, đã push chưa.
- Test nghiệm thu miễn trừ (⚠️) và lý do.
- Nội dung `execute-all-notes.md`: việc ngoài phạm vi, lỗi đã sửa ngoài task.
- Nhắc người dùng viết `lessons-learned.md` cho từng phase đã đóng.

## Không được

- Dùng lại một agent vừa sinh test case để tự review.
- Code khi bước 1 chưa `[x]`; lách hook `snaptix guard`/`guard-bash` bằng Bash.
- Sửa/skip test để pass; đánh ✅/`[x]` khi chưa chạy.
- Force push, `--no-verify`, `--amend`; push ngoài commit đóng phase.
- Coi `BLOCKED` là xong; tự `auto done` khi verify cuối chưa đạt.
