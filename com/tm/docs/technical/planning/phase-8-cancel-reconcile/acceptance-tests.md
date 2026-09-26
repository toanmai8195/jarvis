# Test nghiệm thu — Phase 8

> Kiểm chứng requirement và challenge của cả phase; dùng cho **DoD** khi đóng phase. Một test nghiệm thu có thể pass nhờ nhiều task — đánh ✅ khi nó thực sự pass.

| ID | Loại | Kịch bản | Kết quả mong đợi | Requirement / Challenge | Trạng thái |
|---|---|---|---|---|---|
| P8-AT01 | Unit | Huỷ trước 30h, vé 200k | Hoàn 180k | P8-FR1 | ⬜ |
| P8-AT02 | Unit | Huỷ trước đúng 24h | Hoàn 90% (biên bao gồm) | P8-FR1 | ⬜ |
| P8-AT03 | Unit | Huỷ trước 3h | Hoàn 0; API huỷ trả 422 `CANCEL_NOT_ALLOWED` hoặc cho huỷ không hoàn (theo cấu hình) | P8-FR1 | ⬜ |
| P8-AT04 | Integration | Huỷ 1 vé | Vé CANCELLED; ghế AVAILABLE; ví +hoàn; `available_seats` +1; outbox `booking.cancelled` | P8-FR2 | ⬜ |
| P8-AT05 | Integration | Huỷ cùng vé 2 lần (cùng hoặc khác key) | Hoàn tiền 1 lần | P8-FR2 | ⬜ |
| P8-AT06 | Integration | Sau khi huỷ, người khác giữ và mua lại ghế đó | Thành công (unique index partial cho phép) | P8-FR2 | ⬜ |
| P8-AT07 | Integration | Huỷ 1 trong 2 vé của đơn | Đơn PARTIALLY_CANCELLED | P8-FR3 | ⬜ |
| P8-AT08 | Integration | Huỷ chuyến có 40 vé | 40 vé CANCELLED, mỗi vé hoàn 100% | P8-FR4 | ⬜ |
| P8-AT09 | Integration | Kill job huỷ chuyến sau 15 vé, chạy lại | Hoàn nốt 25 vé; không vé nào hoàn 2 lần | P8-FR4 | ⬜ |
| P8-AT10 | Integration | Support hoàn thủ công vượt số đã trả | Từ chối | P8-FR5 | ⬜ |
| P8-AT11 | Integration | Support hoàn thủ công hợp lệ | Ledger đúng; audit log có lý do | P8-FR5 | ⬜ |
| P8-AT12 | Integration | Chèn thẳng một entry lệch vào DB test | Job đối soát phát hiện, bắn cảnh báo | P8-FR6 | ⬜ |
| P8-AT13 | Integration | Topup SUCCEEDED ở mock provider nhưng webhook bị mất | Đối soát phát hiện và cộng bù (hoặc báo cáo) | P8-FR6 | ⬜ |
| P8-AT14 | Migration | Thêm cột + backfill 10 triệu dòng `tickets` khi k6 booking đang chạy | Không request lỗi do lock; p99 tăng < 20% | P8-NFR1, P10 | ⬜ |
| P8-AT15 | Migration | `CREATE INDEX CONCURRENTLY` dưới tải | Không chặn ghi | P10 | ⬜ |
| P8-AT16 | Migration | Migration chờ lock quá `lock_timeout` | Thất bại nhanh, không treo hàng đợi lock | P10 | ⬜ |
| P8-AT17 | Manual | Chọn ghế và thanh toán chỉ bằng bàn phím | Hoàn thành được; focus hiển thị rõ | R7 | ⬜ |
| P8-AT18 | Manual | Screen reader đọc sơ đồ ghế | Đọc được mã ghế, hạng, trạng thái | R7 | ⬜ |
| P8-AT19 | Automated | Lighthouse accessibility các trang chính | ≥ 95 | P8-NFR2, R7 | ⬜ |
| P8-AT20 | E2E | Đăng nhập → nạp tiền → đặt vé → huỷ vé | Pass trong CI | P8-NFR3, R8 | ⬜ |
| P8-AT21 | E2E | Admin setup → khách đặt vé → admin huỷ chuyến → khách thấy hoàn tiền | Pass trong CI | P8-NFR3, R8 | ⬜ |
| P8-AT22 | Regression | Toàn bộ test case phase 0–7 | Pass | — | ⬜ |
