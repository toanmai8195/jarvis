# Test nghiệm thu — Phase 13

> Kiểm chứng requirement và challenge của cả phase; dùng cho **DoD** khi đóng phase. Một test nghiệm thu có thể pass nhờ nhiều task — đánh ✅ khi nó thực sự pass.

| ID | Loại | Kịch bản | Kết quả mong đợi | Requirement / Challenge | Trạng thái |
|---|---|---|---|---|---|
| P13-AT01 | Integration | Đặt vé thành công qua saga | Vé ISSUED; wallet capture đúng số tiền; saga COMPLETED | P13-FR2, S1 | ⬜ |
| P13-AT02 | Integration | Reserve thành công, xác nhận vé thất bại | Wallet Release; ghế trả lại; saga COMPENSATED | P13-FR2, S1 | ⬜ |
| P13-AT03 | Chaos | Kill wallet sau Reserve, trước Capture | Sau khi wallet lên lại: saga tiếp tục hoặc bù trừ; không treo tiền mãi | P13-NFR1, S1 | ⬜ |
| P13-AT04 | Chaos | Timeout khi gọi Capture (không rõ kết quả), retry | Tiền capture đúng 1 lần | P13-FR4, S3 | ⬜ |
| P13-AT05 | Integration | Gọi `Reserve` 10 lần cùng key | 1 reservation | P13-FR4, S3 | ⬜ |
| P13-AT06 | Integration | Huỷ vé → Refund qua gRPC | Ví +hoàn đúng; đối soát khớp | P13-FR3 | ⬜ |
| P13-AT07 | CI | Xoá field bắt buộc trong `wallet.proto` | `buf breaking` fail | S2 | ⬜ |
| P13-AT08 | Integration | wallet phiên bản mới, core phiên bản cũ | Luồng đặt vé vẫn chạy | S2 | ⬜ |
| P13-AT09 | Integration | Kiểm tra quyền DB | Core không có credential DB wallet | P13-FR1 | ⬜ |
| P13-AT10 | Script | Đối soát ledger wallet vs booking core sau toàn bộ test | Chênh lệch 0 | P13-NFR1, S3 | ⬜ |
| P13-AT11 | Load | Bộ k6 phase 7 trên kiến trúc mới | Có số liệu so sánh latency/throughput | P13-NFR2, S4 | ⬜ |
