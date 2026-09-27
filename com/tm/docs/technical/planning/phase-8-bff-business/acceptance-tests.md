# Test nghiệm thu — Phase 8

> Kiểm chứng requirement và challenge của cả phase; dùng cho **DoD** khi đóng phase. Một test nghiệm thu có thể pass nhờ nhiều task — đánh ✅ khi nó thực sự pass.

| ID | Loại | Kịch bản | Kết quả mong đợi | Requirement / Challenge | Trạng thái |
|---|---|---|---|---|---|
| P8-AT01 | E2E | Hai trình duyệt xem cùng chuyến, A giữ ghế | B thấy ghế chuyển HELD trong < 1s | P8-FR1, N6 | ⬜ |
| P8-AT02 | Load | 10.000 kết nối SSE, 1 giờ | Bộ nhớ BFF ổn định; kết nối đóng được dọn | N6 | ⬜ |
| P8-AT03 | Integration | Ma trận: 4 vai trò × toàn bộ route admin | Chỉ vai trò được phép nhận 2xx; còn lại 403 | P8-FR2, N4 | ⬜ |
| P8-AT04 | Integration | Route admin mới chưa khai báo quyền | Mặc định 403 | N4 | ⬜ |
| P8-AT05 | Integration | Analyst gọi API ghi | 403, có audit log lần từ chối | N4 | ⬜ |
| P8-AT06 | Integration | Operator sửa giá | Audit log có actor, before, after, ip, thời gian | P8-FR3, M4 | ⬜ |
| P8-AT07 | Integration | Dùng user DB của BFF chạy `updateOne` / `deleteOne` trên `audit_logs` | Bị từ chối quyền | M4 | ⬜ |
| P8-AT08 | Integration | Xuất CSV 1 triệu dòng | Stream, bộ nhớ BFF không tăng tuyến tính | P8-FR4 | ⬜ |
