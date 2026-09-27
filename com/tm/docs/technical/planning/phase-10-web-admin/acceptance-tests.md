# Test nghiệm thu — Phase 10

> Kiểm chứng requirement và challenge của cả phase; dùng cho **DoD** khi đóng phase. Một test nghiệm thu có thể pass nhờ nhiều task — đánh ✅ khi nó thực sự pass.

| ID | Loại | Kịch bản | Kết quả mong đợi | Requirement / Challenge | Trạng thái |
|---|---|---|---|---|---|
| P10-AT01 | E2E | Form lịch chạy nhập ngày kết thúc trước ngày bắt đầu | Báo lỗi ở client; gửi thẳng API cũng bị 400 với cùng thông điệp | R4 | ⬜ |
| P10-AT02 | E2E | Vẽ sơ đồ 2 tầng, 30 giường bằng kéo thả, lưu, mở lại | Sơ đồ giữ nguyên | R4 | ⬜ |
| P10-AT03 | E2E | Bảng đơn hàng 100.000 dòng, cuộn nhanh, lọc theo chuyến | Cuộn mượt (≥ 50 fps), lọc < 500ms | P10-NFR1, R5 | ⬜ |
| P10-AT04 | Integration | So sánh tổng doanh thu analytics với ledger `REVENUE` | Bằng nhau | P10-FR1 | ⬜ |
| P10-AT05 | E2E | Mở dashboard | Hiển thị đầy đủ < 2s; skeleton khi đang tải | R6 | ⬜ |
