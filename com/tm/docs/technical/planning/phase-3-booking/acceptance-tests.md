# Test nghiệm thu — Phase 3

> Kiểm chứng requirement và challenge của cả phase; dùng cho **DoD** khi đóng phase. Một test nghiệm thu có thể pass nhờ nhiều task — đánh ✅ khi nó thực sự pass.

| ID | Loại | Kịch bản | Kết quả mong đợi | Requirement / Challenge | Trạng thái |
|---|---|---|---|---|---|
| P3-AT01 | Integration | Giữ ghế A05, A06 còn trống | 201; 2 ghế HELD; `expires_at` = now + 10 phút | P3-FR1 | ⬜ |
| P3-AT02 | Integration | Giữ A05 (trống) + A07 (đã SOLD) | 409 `SEAT_UNAVAILABLE`; A05 vẫn AVAILABLE | P3-FR1, P1 | ⬜ |
| P3-AT03 | Integration | Trả hold chủ động | Ghế về AVAILABLE; hold RELEASED | P3-FR2 | ⬜ |
| P3-AT04 | Integration | Hold quá hạn | Worker chuyển EXPIRED, ghế AVAILABLE; người khác giữ được | P3-FR2 | ⬜ |
| P3-AT05 | Integration | 3 instance worker hết hạn hold chạy cùng lúc trên 10.000 hold | Mỗi hold xử lý đúng 1 lần, không bỏ sót | G4 | ⬜ |
| P3-AT06 | Integration | Đặt vé từ hold hợp lệ, đủ tiền | Booking CONFIRMED; tickets ISSUED; ghế SOLD; ví trừ đúng; `available_seats` giảm; 1 outbox event | P3-FR3 | ⬜ |
| P3-AT07 | Integration | Đặt vé với hold của người khác | 403/404; không thay đổi gì | P3-FR3 | ⬜ |
| P3-AT08 | Integration | Đặt vé khi hold vừa hết hạn | 409 `HOLD_EXPIRED` | P3-FR3 | ⬜ |
| P3-AT09 | Integration | Số dư thiếu | 422 `INSUFFICIENT_BALANCE`; hold ACTIVE; ghế vẫn HELD | P3-FR4 | ⬜ |
| P3-AT10 | Integration | Client gửi kèm giá thấp hơn | Server bỏ qua, dùng giá từ `fares` | P3-FR3 | ⬜ |
| P3-AT11 | Integration | Lỗi giả lập ngay trước COMMIT booking | Rollback: ví, ghế, vé, outbox đều không đổi | P3-FR3 | ⬜ |
| P3-AT12 | Integration | Bấm đặt vé 5 lần cùng `Idempotency-Key` | 1 booking, 1 lần trừ tiền | P3-NFR3 | ⬜ |
| P3-AT13 | Concurrency | 500 goroutine cùng giữ ghế A05 | Đúng 1 thành công, 499 nhận `SEAT_UNAVAILABLE` | P3-NFR1, P1, G9 | ⬜ |
| P3-AT14 | Concurrency | User X giữ [A1, A2], user Y giữ [A2, A1] đồng thời lặp 1.000 lần | Không deadlock (khoá theo thứ tự) | P3 | ⬜ |
| P3-AT15 | Concurrency | Chèn thẳng 2 ticket cùng ghế bằng SQL | Unique index từ chối | P1 | ⬜ |
| P3-AT16 | Load | k6: 1.000 VU tranh 40 ghế | 40 vé bán; 0 trùng; 0 lỗi 5xx ngoài 503 quá tải | P3-NFR1, G1 | ⬜ |
| P3-AT17 | Load | k6: 500 booking/s trong 5 phút | p99 < 300ms; không leak goroutine (pprof trước/sau) | P3-NFR2, G1 | ⬜ |
| P3-AT18 | Benchmark | Chạy TC16/TC17 cho 3 chiến lược khoá | Bảng throughput/latency/lỗi trong ADR | P2 | ⬜ |
