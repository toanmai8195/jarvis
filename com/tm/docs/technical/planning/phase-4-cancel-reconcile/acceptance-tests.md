# Test nghiệm thu — Phase 4

> Kiểm chứng requirement và challenge của cả phase; dùng cho **DoD** khi đóng phase. Một test nghiệm thu có thể pass nhờ nhiều task — đánh ✅ khi nó thực sự pass.

| ID | Loại | Kịch bản | Kết quả mong đợi | Requirement / Challenge | Trạng thái |
|---|---|---|---|---|---|
| P4-AT01 | Unit | Huỷ trước 30h, vé 200k | Hoàn 180k | P4-FR1 | ⬜ |
| P4-AT02 | Unit | Huỷ trước đúng 24h | Hoàn 90% (biên bao gồm) | P4-FR1 | ⬜ |
| P4-AT03 | Unit | Huỷ trước 3h | Hoàn 0; API huỷ trả 422 `CANCEL_NOT_ALLOWED` hoặc cho huỷ không hoàn (theo cấu hình) | P4-FR1 | ⬜ |
| P4-AT04 | Integration | Huỷ 1 vé | Vé CANCELLED; ghế AVAILABLE; ví +hoàn; `available_seats` +1; outbox `booking.cancelled` | P4-FR2 | ⬜ |
| P4-AT05 | Integration | Huỷ cùng vé 2 lần (cùng hoặc khác key) | Hoàn tiền 1 lần | P4-FR2 | ⬜ |
| P4-AT06 | Integration | Sau khi huỷ, người khác giữ và mua lại ghế đó | Thành công (unique index partial cho phép) | P4-FR2 | ⬜ |
| P4-AT07 | Integration | Huỷ 1 trong 2 vé của đơn | Đơn PARTIALLY_CANCELLED | P4-FR3 | ⬜ |
| P4-AT08 | Integration | Huỷ chuyến có 40 vé | 40 vé CANCELLED, mỗi vé hoàn 100% | P4-FR4 | ⬜ |
| P4-AT09 | Integration | Kill job huỷ chuyến sau 15 vé, chạy lại | Hoàn nốt 25 vé; không vé nào hoàn 2 lần | P4-FR4 | ⬜ |
| P4-AT10 | Integration | Support hoàn thủ công vượt số đã trả | Từ chối | P4-FR5 | ⬜ |
| P4-AT11 | Integration | Support hoàn thủ công hợp lệ | Ledger đúng; audit log có lý do | P4-FR5 | ⬜ |
| P4-AT12 | Integration | Chèn thẳng một entry lệch vào DB test | Job đối soát phát hiện, bắn cảnh báo | P4-FR6 | ⬜ |
| P4-AT13 | Integration | Topup SUCCEEDED ở mock provider nhưng webhook bị mất | Đối soát phát hiện và cộng bù (hoặc báo cáo) | P4-FR6 | ⬜ |
| P4-AT14 | Migration | Thêm cột + backfill 10 triệu dòng `tickets` khi k6 booking đang chạy | Không request lỗi do lock; p99 tăng < 20% | P4-NFR1, P10 | ⬜ |
| P4-AT15 | Migration | `CREATE INDEX CONCURRENTLY` dưới tải | Không chặn ghi | P10 | ⬜ |
| P4-AT16 | Migration | Migration chờ lock quá `lock_timeout` | Thất bại nhanh, không treo hàng đợi lock | P10 | ⬜ |
| P4-AT17 | Regression | Toàn bộ test case phase 0–7 | Pass | — | ⬜ |
