# Test nghiệm thu — Phase 4

> Kiểm chứng requirement và challenge của cả phase; dùng cho **DoD** khi đóng phase. Một test nghiệm thu có thể pass nhờ nhiều task — đánh ✅ khi nó thực sự pass.

| ID | Loại | Kịch bản | Kết quả mong đợi | Requirement / Challenge | Trạng thái |
|---|---|---|---|---|---|
| P4-AT01 | Integration | Giữ ghế A05, A06 còn trống | 201; 2 ghế HELD; `expires_at` = now + 10 phút | P4-FR1 | ⬜ |
| P4-AT02 | Integration | Giữ A05 (trống) + A07 (đã SOLD) | 409 `SEAT_UNAVAILABLE`; A05 vẫn AVAILABLE | P4-FR1, P1 | ⬜ |
| P4-AT03 | Integration | Trả hold chủ động | Ghế về AVAILABLE; hold RELEASED | P4-FR2 | ⬜ |
| P4-AT04 | Integration | Hold quá hạn | Worker chuyển EXPIRED, ghế AVAILABLE; người khác giữ được | P4-FR2 | ⬜ |
| P4-AT05 | Integration | 3 instance worker hết hạn hold chạy cùng lúc trên 10.000 hold | Mỗi hold xử lý đúng 1 lần, không bỏ sót | G4 | ⬜ |
| P4-AT06 | Integration | Đặt vé từ hold hợp lệ, đủ tiền | Booking CONFIRMED; tickets ISSUED; ghế SOLD; ví trừ đúng; `available_seats` giảm; 1 outbox event | P4-FR3 | ⬜ |
| P4-AT07 | Integration | Đặt vé với hold của người khác | 403/404; không thay đổi gì | P4-FR3 | ⬜ |
| P4-AT08 | Integration | Đặt vé khi hold vừa hết hạn | 409 `HOLD_EXPIRED` | P4-FR3 | ⬜ |
| P4-AT09 | Integration | Số dư thiếu | 422 `INSUFFICIENT_BALANCE`; hold ACTIVE; ghế vẫn HELD | P4-FR4 | ⬜ |
| P4-AT10 | Integration | Client gửi kèm giá thấp hơn | Server bỏ qua, dùng giá từ `fares` | P4-FR3 | ⬜ |
| P4-AT11 | Integration | Lỗi giả lập ngay trước COMMIT booking | Rollback: ví, ghế, vé, outbox đều không đổi | P4-FR3 | ⬜ |
| P4-AT12 | Integration | Bấm đặt vé 5 lần cùng `Idempotency-Key` | 1 booking, 1 lần trừ tiền | P4-NFR3 | ⬜ |
| P4-AT13 | Concurrency | 500 goroutine cùng giữ ghế A05 | Đúng 1 thành công, 499 nhận `SEAT_UNAVAILABLE` | P4-NFR1, P1, G9 | ⬜ |
| P4-AT14 | Concurrency | User X giữ [A1, A2], user Y giữ [A2, A1] đồng thời lặp 1.000 lần | Không deadlock (khoá theo thứ tự) | P3 | ⬜ |
| P4-AT15 | Concurrency | Chèn thẳng 2 ticket cùng ghế bằng SQL | Unique index từ chối | P1 | ⬜ |
| P4-AT16 | Load | k6: 1.000 VU tranh 40 ghế | 40 vé bán; 0 trùng; 0 lỗi 5xx ngoài 503 quá tải | P4-NFR1, G1 | ⬜ |
| P4-AT17 | Load | k6: 500 booking/s trong 5 phút | p99 < 300ms; không leak goroutine (pprof trước/sau) | P4-NFR2, G1 | ⬜ |
| P4-AT18 | Benchmark | Chạy TC16/TC17 cho 3 chiến lược khoá | Bảng throughput/latency/lỗi trong ADR | P2 | ⬜ |
| P4-AT19 | E2E | Hai trình duyệt xem cùng chuyến, A giữ ghế | B thấy ghế chuyển HELD trong < 1s | P4-FR5, N6 | ⬜ |
| P4-AT20 | Load | 10.000 kết nối SSE, 1 giờ | Bộ nhớ BFF ổn định; kết nối đóng được dọn | N6 | ⬜ |
| P4-AT21 | Unit (React) | Đổi trạng thái 1 ghế trên sơ đồ 300 ghế | Chỉ ghế đó render lại (React Profiler) | R1 | ⬜ |
| P4-AT22 | E2E | Chọn ghế vừa bị người khác giữ | UI rollback, hiện thông báo | R2 | ⬜ |
| P4-AT23 | E2E | Refresh ở bước thanh toán | Khôi phục đúng bước, đồng hồ đếm tiếp | R3 | ⬜ |
| P4-AT24 | E2E | Bấm nút thanh toán liên tục | Một booking; nút bị khoá khi đang xử lý | R3, P4-NFR3 | ⬜ |
| P4-AT25 | E2E | Thanh toán xong, vào trang ví | Số dư mới, không phải số cũ từ cache | R9 | ⬜ |
