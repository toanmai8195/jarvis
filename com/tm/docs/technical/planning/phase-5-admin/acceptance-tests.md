# Test nghiệm thu — Phase 5

> Kiểm chứng requirement và challenge của cả phase; dùng cho **DoD** khi đóng phase. Một test nghiệm thu có thể pass nhờ nhiều task — đánh ✅ khi nó thực sự pass.

| ID | Loại | Kịch bản | Kết quả mong đợi | Requirement / Challenge | Trạng thái |
|---|---|---|---|---|---|
| P5-AT01 | Integration | Tạo tuyến 4 trạm, offset tăng dần | Lưu thành công, thứ tự đúng | P5-FR1 | ⬜ |
| P5-AT02 | Integration | Tạo tuyến có offset giảm dần | 400 `VALIDATION_ERROR` | P5-FR1, R4 | ⬜ |
| P5-AT03 | Integration | Sửa sơ đồ ghế đã có vé bán | Từ chối; gợi ý tạo phiên bản mới | P5-FR1 | ⬜ |
| P5-AT04 | Integration | Đổi giá khi đã có vé bán | Vé cũ giữ giá; vé mới dùng giá mới; bản ghi giá cũ còn nguyên | P5-FR2 | ⬜ |
| P5-AT05 | Integration | Preview lịch hằng ngày T2–T6 trong 1 tháng | Đúng số slot, đúng ngày | P5-FR3 | ⬜ |
| P5-AT06 | Integration | Xác nhận sinh slot 2 lần | Không tạo slot trùng | P5-FR3 | ⬜ |
| P5-AT07 | Performance | Sinh 1 năm slot, xe 40 ghế | < 5s | P5-NFR1 | ⬜ |
| P5-AT08 | Integration | Đổi sang xe có sơ đồ thiếu ghế đã bán | Từ chối, liệt kê ghế xung đột | P5-FR4 | ⬜ |
| P5-AT09 | Integration | Tạm dừng bán slot | Hold mới bị từ chối; slot ẩn khỏi tìm kiếm | P5-FR4 | ⬜ |
| P5-AT10 | Integration | Khoá ghế A01 | A01 trạng thái BLOCKED, không giữ được | P5-FR4 | ⬜ |
| P5-AT11 | Integration | Ma trận: 4 vai trò × toàn bộ route admin | Chỉ vai trò được phép nhận 2xx; còn lại 403 | P5-FR6, N4 | ⬜ |
| P5-AT12 | Integration | Route admin mới chưa khai báo quyền | Mặc định 403 | N4 | ⬜ |
| P5-AT13 | Integration | Analyst gọi API ghi | 403, có audit log lần từ chối | N4 | ⬜ |
| P5-AT14 | Integration | Operator sửa giá | Audit log có actor, before, after, ip, thời gian | P5-FR7, M4 | ⬜ |
| P5-AT15 | Integration | Dùng user DB của BFF chạy `updateOne` / `deleteOne` trên `audit_logs` | Bị từ chối quyền | M4 | ⬜ |
| P5-AT16 | E2E | Form lịch chạy nhập ngày kết thúc trước ngày bắt đầu | Báo lỗi ở client; gửi thẳng API cũng bị 400 với cùng thông điệp | R4 | ⬜ |
| P5-AT17 | E2E | Vẽ sơ đồ 2 tầng, 30 giường bằng kéo thả, lưu, mở lại | Sơ đồ giữ nguyên | R4 | ⬜ |
| P5-AT18 | E2E | Bảng đơn hàng 100.000 dòng, cuộn nhanh, lọc theo chuyến | Cuộn mượt (≥ 50 fps), lọc < 500ms | P5-NFR2, R5 | ⬜ |
| P5-AT19 | E2E | Setup tuyến → sinh slot → đặt vé trên web client | Thành công | P5-FR1..3 | ⬜ |
