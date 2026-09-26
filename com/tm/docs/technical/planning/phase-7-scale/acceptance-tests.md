# Test nghiệm thu — Phase 7

> Kiểm chứng requirement và challenge của cả phase; dùng cho **DoD** khi đóng phase. Một test nghiệm thu có thể pass nhờ nhiều task — đánh ✅ khi nó thực sự pass.

| ID | Loại | Kịch bản | Kết quả mong đợi | Requirement / Challenge | Trạng thái |
|---|---|---|---|---|---|
| P7-AT01 | Load | k6 search 2.000 req/s qua BFF, 10 phút | p99 < 150ms, lỗi < 0,1% | P7-NFR1 | ⬜ |
| P7-AT02 | Load | k6 booking 500/s qua BFF + load balancer, 10 phút | p99 < 300ms; 0 vé trùng | P7-NFR2 | ⬜ |
| P7-AT03 | Profile | pprof CPU dưới tải trước/sau tối ưu | Điểm nóng chính giảm ≥ 30% hoặc có giải thích | G8 | ⬜ |
| P7-AT04 | Profile | pprof mutex/block | Không có lock tranh chấp bất thường trong code Go | G8 | ⬜ |
| P7-AT05 | Load | Scale core 4 instance + 2 worker | Không lỗi `too many connections`; số connection PG < giới hạn | P7-NFR3, P9 | ⬜ |
| P7-AT06 | Integration | pgx qua PgBouncer transaction mode | Không lỗi prepared statement | P9 | ⬜ |
| P7-AT07 | Integration | Đặt vé rồi đọc ngay "vé của tôi" khi replica trễ 2s | Vẫn thấy vé mới (đọc từ primary) | P11 | ⬜ |
| P7-AT08 | Load | Tìm chuyến dưới tải | ≥ 80% truy vấn đọc đi vào replica | P11 | ⬜ |
| P7-AT09 | Soak | Soak 1h ở 50% tải NFR1/NFR2 | Dead tuples `trip_seats` ổn định; kích thước bảng 30 phút cuối tăng < 10% | P7-NFR4, P12 | ⬜ |
| P7-AT10 | Soak | Soak 1h ở 50% tải NFR1/NFR2 | RSS core và bff 30 phút cuối tăng < 10% | P7-NFR4, N9 | ⬜ |
| P7-AT11 | Load | Xuất CSV lớn trong khi phục vụ search | Event loop lag p99 < 50ms | N1 | ⬜ |
| P7-AT12 | Integration | 3 instance BFF, gửi 40 request tìm chuyến / 10s từ 1 IP xoay vòng qua các instance | Request thứ 31 trở đi nhận 429 | P7-FR1, N5 | ⬜ |
| P7-AT13 | Load | 1.000 request cùng lúc vào 1 key cache vừa hết hạn | Chỉ 1 request đi xuống core | N7 | ⬜ |
| P7-AT14 | Load | Search có cache | Request tới core giảm ≥ 70% | N7 | ⬜ |
| P7-AT15 | Load | Mở bán với Redis pre-check | Số transaction PG thất bại do tranh chấp giảm; dữ liệu Redis khớp PG sau test | D1 | ⬜ |
| P7-AT16 | Load | Traffic mở bán gấp 10 lần, bật waiting room | p99 người đã vào < 300ms; người chờ thấy vị trí hàng đợi | P7-NFR5, D2 | ⬜ |
| P7-AT17 | Chaos | Kill Redis giữa load test booking | Latency tăng; 0 vé trùng; script bất biến ledger = 0 | P7-NFR6, D3 | ⬜ |
| P7-AT18 | Chaos | Kill 1 instance core giữa load test | Load balancer chuyển tải; request idempotent retry thành công; không mất tiền | P7-NFR2 | ⬜ |
| P7-AT19 | Chaos | Tiêm độ trễ 200ms vào PG (toxiproxy) | Circuit breaker BFF hoạt động; không treo | P7-NFR2 | ⬜ |
| P7-AT20 | Load | Core giới hạn 2 CPU / 1GB, tải booking 10 phút | Không CPU throttling đáng kể, không OOMKill; GC pause p99 có số liệu | G13 | ⬜ |
