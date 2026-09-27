# Test nghiệm thu — Phase 7

> Kiểm chứng requirement và challenge của cả phase; dùng cho **DoD** khi đóng phase. Một test nghiệm thu có thể pass nhờ nhiều task — đánh ✅ khi nó thực sự pass.

| ID | Loại | Kịch bản | Kết quả mong đợi | Requirement / Challenge | Trạng thái |
|---|---|---|---|---|---|
| P7-AT01 | E2E | Đăng nhập Google lần đầu | Tạo document `users`, core có `users` + `accounts` số dư 0 | P7-FR1, M3 | ⬜ |
| P7-AT02 | Integration | Đăng nhập lần 2 cùng tài khoản | Không tạo user trùng; session mới | P7-FR1, M1 | ⬜ |
| P7-AT03 | Integration | Callback OAuth với `state` sai | 400, không tạo session | N3 | ⬜ |
| P7-AT04 | Integration | Session id trước và sau đăng nhập | Khác nhau (chống session fixation) | N3 | ⬜ |
| P7-AT05 | Integration | Đăng xuất rồi dùng lại cookie cũ | 401 | P7-FR2, N3 | ⬜ |
| P7-AT06 | Integration | Session quá hạn | Document bị TTL xoá; request trả 401 | M2 | ⬜ |
| P7-AT07 | Integration | POST không có CSRF token | 403 | P7-NFR1, N3 | ⬜ |
| P7-AT08 | Manual | Kiểm tra cookie trên trình duyệt | HttpOnly, Secure, SameSite=Lax | P7-NFR1 | ⬜ |
| P7-AT09 | Integration | Core chậm 10s | BFF trả 504 sau ≤ 2s | P7-NFR2, N2 | ⬜ |
| P7-AT10 | Integration | Core lỗi 5 lần liên tiếp | Circuit mở, request tiếp theo fail ngay không gọi core | N2 | ⬜ |
| P7-AT11 | Integration | Core trả 503 một lần rồi 200 cho GET | BFF retry, trả 200 | N2 | ⬜ |
| P7-AT12 | Integration | Tạo user ở core thất bại lúc đăng nhập | Job đối soát tạo bù; lần sau user có ví | M3 | ⬜ |
| P7-AT13 | Integration | Gọi `POST /internal/v1/users` 2 lần cùng id | Một user, một ví | P2-T10 | ⬜ |
| P7-AT14 | Build | Đổi kiểu field trong core OpenAPI | `pnpm build` BFF fail | N8 | ⬜ |
| P7-AT15 | Integration | Request body sai schema | 400 `VALIDATION_ERROR` | N8 | ⬜ |
| P7-AT16 | Manual | `explain()` truy vấn `users` theo `google_sub` | IXSCAN, không COLLSCAN | M2 | ⬜ |
| P7-AT17 | E2E | Tìm chuyến trên web, xem chi tiết | Dữ liệu khớp API core | P7-FR4, P9-FR1 | ⬜ |
| P7-AT18 | Manual | Gọi `bff /healthz?deep=1`, mở Grafana | Một trace chứa span bff → core → PG | P0-NFR2, G10 | ⬜ |
| P7-AT19 | Integration | Dừng MongoDB, gọi `bff /readyz` | 503 trong ≤ 3s; `/healthz` vẫn 200 | P7-FR5 | ⬜ |
| P7-AT20 | CI | PR có test TS fail | CI fail ở bước test | P7-FR5 | ⬜ |
| P7-AT21 | CI | PR chỉ sửa `com/tm/app/**` | Không chạy job Go | P7-FR5 | ⬜ |
