# Test nghiệm thu — Phase 9

> Kiểm chứng requirement và challenge của cả phase; dùng cho **DoD** khi đóng phase. Một test nghiệm thu có thể pass nhờ nhiều task — đánh ✅ khi nó thực sự pass.

| ID | Loại | Kịch bản | Kết quả mong đợi | Requirement / Challenge | Trạng thái |
|---|---|---|---|---|---|
| P9-AT01 | E2E | Chưa đăng nhập, vào `/tickets` | Chuyển sang đăng nhập, xong quay về `/tickets` | P9-FR2, R10 | ⬜ |
| P9-AT02 | E2E | Chưa đăng nhập, reload trang protected | Không nháy nội dung protected | R10 | ⬜ |
| P9-AT03 | Build | Chạy bundle analyzer | JS ban đầu < 200KB gzip | P9-NFR1, R11 | ⬜ |
| P9-AT04 | Unit (React) | Đổi trạng thái 1 ghế trên sơ đồ 300 ghế | Chỉ ghế đó render lại (React Profiler) | R1 | ⬜ |
| P9-AT05 | E2E | Chọn ghế vừa bị người khác giữ | UI rollback, hiện thông báo | R2 | ⬜ |
| P9-AT06 | E2E | Refresh ở bước thanh toán | Khôi phục đúng bước, đồng hồ đếm tiếp | R3 | ⬜ |
| P9-AT07 | E2E | Bấm nút thanh toán liên tục | Một booking; nút bị khoá khi đang xử lý | R3, P3-NFR3 | ⬜ |
| P9-AT08 | E2E | Thanh toán xong, vào trang ví | Số dư mới, không phải số cũ từ cache | R9 | ⬜ |
| P9-AT09 | Manual | Chọn ghế và thanh toán chỉ bằng bàn phím | Hoàn thành được; focus hiển thị rõ | R7 | ⬜ |
| P9-AT10 | Manual | Screen reader đọc sơ đồ ghế | Đọc được mã ghế, hạng, trạng thái | R7 | ⬜ |
| P9-AT11 | Automated | Lighthouse accessibility các trang chính | ≥ 95 | P9-NFR2, R7 | ⬜ |
