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
| P5-AT11 | E2E | Setup tuyến → sinh slot → đặt vé trên web client | Thành công | P5-FR1..3 | ⬜ |
