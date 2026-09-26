---
name: execute-task
description: Thực hiện task của snaptix theo quy trình 6 bước trong CLAUDE.md — tiếp tục task đang dở, hoặc bắt đầu task tiếp theo khi task trước đã xong và commit. Dùng khi người dùng nói "execute-task", "làm tiếp", "làm task tiếp theo", "tiếp tục task", "làm <Task ID>", hoặc duyệt test case / đồng ý commit cho task đang làm.
---

# execute-task

Quy trình gốc nằm trong `CLAUDE.md` (mục "Quy trình làm một task"). Skill này chỉ xác định **đang ở đâu** và **làm tiếp từ bước đó**, không thay thế hay nới lỏng quy trình.

## 1. Xác định task và bước

```bash
python3 .claude/scripts/planning.py step $ARGUMENTS
```

- Không có tham số → task hiện tại (task đầu tiên chưa xong hoặc chưa commit).
- Có Task ID → task đó; nếu nó không phải task hiện tại thì bước 0 phải chặn, trừ khi người dùng nói rõ cho phép làm vượt.
- Output cho biết: file README phase, file test case của task, acceptance tests, handbook, challenge, **BƯỚC TIẾP THEO**, checklist con.

Đọc `README.md` của phase (mục tiêu, requirement, challenge, DoD) và `com/tm/docs/technical/project-structure.md` trước khi làm.

## 2. Làm tiếp từ đúng bước

| Bước tiếp theo | Làm gì | Dừng ở đâu |
|---|---|---|
| **0** | Xem kết quả validate trong output. Có ❌ → **dừng**, báo điều kiện chưa đạt và cách khắc phục. Đạt hết → thêm checklist con dưới dòng task, phase ⬜ → 🟨 nếu cần, rồi sang bước 1 | Chỉ dừng nếu validate fail |
| **1** | Chưa có file test case → tạo `tasks/<Task ID>/test-cases.md` theo CLAUDE.md, trình bày, **DỪNG chờ duyệt**. Đã có file → nếu tin nhắn hiện tại của người dùng **duyệt rõ ràng** (vd "duyệt", "ok test case", "approve") thì đánh `[x]` bước 1 và làm tiếp bước 2; nếu yêu cầu sửa thì sửa rồi trình bày lại; nếu chưa có ý kiến thì trình bày lại bộ test case và hỏi duyệt | **Luôn dừng** cho đến khi người dùng duyệt |
| **2** | Code trong phạm vi task | Không dừng |
| **3** | Viết unit test | Không dừng |
| **4** | Build + chạy toàn bộ unit test, lint. Fail → sửa code, chạy lại | Dừng và báo nếu không tự sửa được |
| **5** | Chạy từng test case của task, đánh ✅; ghi handbook; cập nhật challenge, acceptance tests liên quan, docs; đánh `[x]` dòng task | Dừng và báo nếu test case fail mà không sửa được |
| **6** | Tóm tắt thay đổi, đề xuất commit message, **hỏi commit? push?**. Nếu tin nhắn hiện tại đã đồng ý rõ ràng → đánh `[x]` bước 6 kèm commit message và quyết định push **trước**, rồi commit tất cả trong một commit (message chứa `[<Task ID>]`), push nếu được đồng ý | **Luôn dừng** chờ người dùng quyết định commit/push |

Sau mỗi bước: đánh `[x]` bước đó trong checklist con ngay, rồi sang bước kế tiếp trong cùng lượt nếu bước đó không có điểm dừng.

## 3. Sau khi xong task

Khi bước 6 đã `[x]` (đã commit): báo ngắn task vừa xong, rồi chạy `python3 .claude/scripts/planning.py step` để giới thiệu task tiếp theo. **Không tự bắt đầu task tiếp theo** — chờ người dùng gọi lại skill.

## Không được

- Bỏ qua điểm dừng ở bước 1 (duyệt test case) và bước 6 (commit/push).
- Coi sự im lặng hoặc câu mơ hồ là đồng ý.
- Lách hook `snaptix guard` bằng Bash để sửa code — hook chặn nghĩa là quy trình chưa đúng.
- Làm nhiều task trong một lần gọi.
