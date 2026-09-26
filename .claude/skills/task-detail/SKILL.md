---
name: task-detail
description: Mô tả task trước, task hiện tại và task tiếp theo của snaptix — làm gì, challenge nào, test case nào, trạng thái checklist và điều kiện bước 0. Dùng khi người dùng hỏi "task hiện tại là gì", "tiếp theo làm gì", "task-detail", hoặc đưa một ID task (vd P3-T05).
---

# task-detail

1. Chạy (tham số là ID task nếu người dùng đưa, bỏ trống để tự xác định task hiện tại):

   ```bash
   python3 .claude/scripts/planning.py task $ARGUMENTS
   ```

2. Đọc thêm `README.md` của phase chứa task hiện tại (mục tiêu, requirement liên quan, DoD) để giải thích đúng ngữ cảnh. Không tự bịa trạng thái — mọi con số và dấu ✅/⬜ lấy từ output script.

3. Trả lời bằng tiếng Việt, ngắn gọn, theo cấu trúc:

   **Task trước — `<ID>`**: đã làm gì (1–2 câu), đã xong/commit chưa.

   **Task hiện tại — `<ID>`** (phần chính):
   - Mục tiêu của task bằng lời dễ hiểu, gắn với mốc demo của phase.
   - Việc cụ thể cần làm (3–6 gạch đầu dòng), bám theo quy ước trong `com/tm/docs/technical/project-structure.md`.
   - Challenge và tiêu chí "hoàn thành khi"; kiến thức Go/PG/Node/React sẽ học được.
   - Test case đã duyệt hoặc liên quan.
   - Checklist con 6 bước: đang ở bước nào (nếu đã bắt đầu).
   - Kết quả validate bước 0: đạt hay chưa; chưa đạt thì nói rõ cần làm gì trước.

   **Task tiếp theo — `<ID>`**: sẽ làm gì (1–2 câu), phụ thuộc gì vào task hiện tại.

4. Chỉ mô tả, **không** bắt đầu làm task. Muốn làm thì người dùng sẽ yêu cầu và quy trình trong `CLAUDE.md` áp dụng.
