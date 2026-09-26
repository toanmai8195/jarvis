# Test nghiệm thu — Phase 6

> Kiểm chứng requirement và challenge của cả phase; dùng cho **DoD** khi đóng phase. Một test nghiệm thu có thể pass nhờ nhiều task — đánh ✅ khi nó thực sự pass.

| ID | Loại | Kịch bản | Kết quả mong đợi | Requirement / Challenge | Trạng thái |
|---|---|---|---|---|---|
| P6-AT01 | Integration | Tạo 1.000 booking, chạy relay | 1.000 event có `published_at`; stats-worker nhận đủ | P6-FR1 | ⬜ |
| P6-AT02 | Integration | 3 instance relay chạy cùng lúc | Không event nào bị bỏ sót | P6-FR1 | ⬜ |
| P6-AT03 | Integration | Stats-worker crash sau khi ghi fact, trước khi ack | Khởi động lại: không ghi trùng fact | P6-FR2, A2 | ⬜ |
| P6-AT04 | Integration | Gửi lại cùng một event 5 lần | `fact_bookings` có 1 dòng | P6-FR2, A2 | ⬜ |
| P6-AT05 | Integration | Replay toàn bộ outbox 2 lần | Mọi aggregate không đổi | P6-FR2, A2 | ⬜ |
| P6-AT06 | Integration | Đặt vé 250k lúc 10:15 | `agg_revenue_hourly` khung 10:00 +250k, +1 vé | A3 | ⬜ |
| P6-AT07 | Integration | Huỷ vé đã tính trong aggregate | Aggregate trừ tương ứng | A3 | ⬜ |
| P6-AT08 | E2E | Đặt vé trên web, mở dashboard | Số liệu cập nhật trong < 5 phút | P6-NFR1, A3 | ⬜ |
| P6-AT09 | Integration | So sánh tổng doanh thu analytics với ledger `REVENUE` | Bằng nhau | P6-FR3 | ⬜ |
| P6-AT10 | Performance | Báo cáo doanh thu 12 tháng theo tuyến trên 100 triệu fact | < 2s; `EXPLAIN` cho thấy partition pruning | P6-NFR2, A4 | ⬜ |
| P6-AT11 | Manual | `EXPLAIN` truy vấn theo khoảng thời gian | Dùng BRIN, chỉ quét partition liên quan | A4 | ⬜ |
| P6-AT12 | Integration | Kiểm tra kết quả top tuyến, tăng trưởng tuần so với tính tay trên dữ liệu nhỏ | Khớp | A5 | ⬜ |
| P6-AT13 | Integration | `REFRESH MATERIALIZED VIEW CONCURRENTLY` khi đang có truy vấn đọc | Truy vấn đọc không bị chặn | A5 | ⬜ |
| P6-AT14 | Integration | Rebuild aggregate khi dashboard đang dùng | Dashboard vẫn trả số liệu cũ cho đến khi swap; sau swap là số mới | P6-FR5, A6 | ⬜ |
| P6-AT15 | Integration | Detach partition `outbox_events` tháng cũ | Hoàn tất tức thì, không khoá bảng chính | P6-NFR3, P8 | ⬜ |
| P6-AT16 | Integration | Insert vào tháng chưa có partition | Job tạo trước partition nên không lỗi | P8 | ⬜ |
| P6-AT17 | Integration | Xuất CSV 1 triệu dòng | Stream, bộ nhớ BFF không tăng tuyến tính | P6-FR4 | ⬜ |
| P6-AT18 | E2E | Mở dashboard | Hiển thị đầy đủ < 2s; skeleton khi đang tải | R6 | ⬜ |
