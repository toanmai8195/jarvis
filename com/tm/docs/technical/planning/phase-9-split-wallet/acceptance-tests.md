# Test nghiệm thu — Phase 9

> Kiểm chứng requirement và challenge của cả phase; dùng cho **DoD** khi đóng phase. Một test nghiệm thu có thể pass nhờ nhiều task — đánh ✅ khi nó thực sự pass.

| ID | Loại | Kịch bản | Kết quả mong đợi | Requirement / Challenge | Trạng thái |
|---|---|---|---|---|---|
| P9-AT01 | Integration | Đặt vé thành công qua saga | Vé ISSUED; wallet capture đúng số tiền; saga COMPLETED | P9-FR2, S1 | ⬜ |
| P9-AT02 | Integration | Reserve thành công, xác nhận vé thất bại | Wallet Release; ghế trả lại; saga COMPENSATED | P9-FR2, S1 | ⬜ |
| P9-AT03 | Chaos | Kill wallet sau Reserve, trước Capture | Sau khi wallet lên lại: saga tiếp tục hoặc bù trừ; không treo tiền mãi | P9-NFR1, S1 | ⬜ |
| P9-AT04 | Chaos | Timeout khi gọi Capture (không rõ kết quả), retry | Tiền capture đúng 1 lần | P9-FR4, S3 | ⬜ |
| P9-AT05 | Integration | Gọi `Reserve` 10 lần cùng key | 1 reservation | P9-FR4, S3 | ⬜ |
| P9-AT06 | Integration | Huỷ vé → Refund qua gRPC | Ví +hoàn đúng; đối soát khớp | P9-FR3 | ⬜ |
| P9-AT07 | CI | Xoá field bắt buộc trong `wallet.proto` | `buf breaking` fail | S2 | ⬜ |
| P9-AT08 | Integration | wallet phiên bản mới, core phiên bản cũ | Luồng đặt vé vẫn chạy | S2 | ⬜ |
| P9-AT09 | Integration | Kiểm tra quyền DB | Core không có credential DB wallet | P9-FR1 | ⬜ |
| P9-AT10 | Script | Đối soát ledger wallet vs booking core sau toàn bộ test | Chênh lệch 0 | P9-NFR1, S3 | ⬜ |
| P9-AT11 | Load | Bộ k6 phase 7 trên kiến trúc mới | Có số liệu so sánh latency/throughput | P9-NFR2, S4 | ⬜ |
