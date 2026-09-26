---
name: phase-detail
description: Mô tả chi tiết một phase của snaptix — mục tiêu, mốc demo, tiến độ task/test case/challenge/DoD theo workstream, phase trước và sau. Dùng khi người dùng hỏi "phase này làm gì", "phase-detail", "tiến độ phase", hoặc đưa số phase (vd 3).
---

# phase-detail

1. Chạy (tham số là số phase nếu người dùng đưa, bỏ trống để lấy phase hiện tại):

   ```bash
   python3 .claude/scripts/planning.py phase $ARGUMENTS
   ```

2. Nếu cần ngữ cảnh sâu hơn, đọc `README.md` của phase đó trong `com/tm/docs/technical/planning/`. Mọi con số và trạng thái lấy từ output script, không tự suy.

3. Trả lời bằng tiếng Việt, theo cấu trúc:

   **Phase `<N>` — `<tên>`** · trạng thái · mốc demo

   - **Mục tiêu**: 2–3 câu, phase này giải quyết bài toán gì trong hệ thống.
   - **Tiến độ**: task, test case, challenge, DoD (dạng x/y); lessons learned đã viết chưa.
   - **Theo workstream**: mỗi workstream một dòng — đã xong gì, còn gì (không liệt kê lại toàn bộ task đã xong nếu dài; nêu task đang dở `[~]` và task kế tiếp).
   - **Challenge**: nhóm theo công nghệ; nêu kiến thức trọng tâm sẽ học (Go / PG / Node / React).
   - **Còn thiếu để đóng phase**: DoD chưa đạt và checklist đóng phase chưa làm.
   - **Liên kết**: phase trước đã đóng chưa; phase sau sẽ làm gì.

4. Chỉ mô tả, không chỉnh sửa file planning.
