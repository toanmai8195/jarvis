---
name: sumup
description: Tổng kết toàn bộ dự án snaptix — đã làm được những gì, còn những gì, tiến độ theo phase, challenge theo công nghệ, kiến thức đã ghi trong handbook, commit gần đây. Dùng khi người dùng hỏi "sumup", "tổng kết", "đã làm được gì", "còn bao nhiêu việc".
---

# sumup

1. Chạy:

   ```bash
   python3 .claude/scripts/planning.py sumup
   ```

2. Mọi con số và trạng thái lấy từ output script, không tự suy. Nếu cần, đọc thêm `com/tm/docs/technical/handbook/` để nêu cụ thể kiến thức đã học.

3. Trả lời bằng tiếng Việt, theo cấu trúc:

   **Tổng quan**: một câu về vị trí hiện tại (phase, task) và % hoàn thành toàn dự án. Kèm bảng tiến độ theo phase.

   **Đã làm được**
   - Các mốc demo đã đạt (phase ✅).
   - Challenge đã xong, nhóm theo công nghệ — diễn giải thành năng lực: ví dụ "đã xử lý được lost update trên ví bằng ...".
   - Kiến thức nổi bật trong handbook (nếu có).

   **Còn lại**
   - Phần còn lại của phase hiện tại.
   - Các phase chưa làm, mỗi phase một dòng với mốc demo.
   - Challenge còn lại theo công nghệ — chỉ ra công nghệ nào đang tụt lại so với mục tiêu (ưu tiên Go và PostgreSQL).

   **Cần chú ý**: task đã xong nhưng chưa commit, phase đã đóng nhưng chưa viết lessons learned, working tree chưa sạch.

4. Chỉ tổng kết, không chỉnh sửa file.
