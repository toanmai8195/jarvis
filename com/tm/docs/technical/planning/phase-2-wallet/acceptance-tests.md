# Test nghiệm thu — Phase 2

> Kiểm chứng requirement và challenge của cả phase; dùng cho **DoD** khi đóng phase. Một test nghiệm thu có thể pass nhờ nhiều task — đánh ✅ khi nó thực sự pass.

| ID | Loại | Kịch bản | Kết quả mong đợi | Requirement / Challenge | Trạng thái |
|---|---|---|---|---|---|
| P2-AT01 | E2E | Nạp 500.000đ, mock provider báo thành công | Số dư +500.000đ; lịch sử có 1 dòng TOPUP | P2-FR1, P2-FR2 | ⬜ |
| P2-AT02 | Integration | Webhook thành công gửi 10 lần | Chỉ 1 `ledger_transaction`; số dư +1 lần | P2-FR2, P6 | ⬜ |
| P2-AT03 | Integration | Webhook sai chữ ký HMAC | 401, không đổi số dư | P2-FR2 | ⬜ |
| P2-AT04 | Integration | Webhook đến sau khi topup đã EXPIRED | Không cộng tiền; ghi log cảnh báo để đối soát | P2-FR4 | ⬜ |
| P2-AT05 | Integration | Topup PENDING 31 phút | Worker chuyển EXPIRED | P2-FR4 | ⬜ |
| P2-AT06 | Integration | 100 request tạo topup song song cùng `Idempotency-Key` | 1 topup; 100 response giống hệt nhau | P2-FR5, P6 | ⬜ |
| P2-AT07 | Integration | Cùng key, body khác | 409 `IDEMPOTENCY_CONFLICT` | P2-FR5, P6 | ⬜ |
| P2-AT08 | Integration | Lịch sử 55 giao dịch, `limit=20` | 3 trang, không trùng không thiếu | P2-FR3 | ⬜ |
| P2-AT09 | Unit | `Ledger.Post` với entries tổng ≠ 0 | Trả lỗi, không ghi gì | P2-NFR1, P5 | ⬜ |
| P2-AT10 | Integration | Ghi thẳng SQL làm số dư ví user âm | Constraint từ chối | P2-NFR2, P5 | ⬜ |
| P2-AT11 | Integration | `UPDATE ledger_entries` bằng user ứng dụng | Bị từ chối | P5 | ⬜ |
| P2-AT12 | Integration | Hai transaction trừ cùng ví song song ở READ COMMITTED, không khoá | **Tái hiện** lost update (test đỏ ở nhánh chưa sửa) | P4 | ⬜ |
| P2-AT13 | Integration | Như TC12 sau khi áp dụng giải pháp | Số dư đúng | P4 | ⬜ |
| P2-AT14 | Integration | Chạy SERIALIZABLE, gây xung đột | Nhận `40001`, retry thành công, kết quả đúng | P4 | ⬜ |
| P2-AT15 | Integration | 200 goroutine cùng cộng/trừ một ví (`-race`) | Số dư cuối = tổng kỳ vọng; không data race | G9 | ⬜ |
| P2-AT16 | Property | Cộng/trừ `Money` ngẫu nhiên | Không overflow âm thầm; phép toán đúng | G6 | ⬜ |
| P2-AT17 | Lint | Thêm `float64` vào package wallet | Lint fail | P2-NFR3, G6 | ⬜ |
| P2-AT18 | Unit | Hàm thuần tạo bút toán cho topup/payment/refund | Tổng = 0, đúng tài khoản; test không cần DB | G5 | ⬜ |
| P2-AT19 | Integration | Use case lỗi giữa chừng (sau khi ghi entry) | Rollback toàn bộ: không entry, không đổi balance, không idempotency record | G5 | ⬜ |
| P2-AT20 | Script | Chạy script bất biến sau toàn bộ test | Tổng entries = 0; balance khớp | P2-NFR1 | ⬜ |
