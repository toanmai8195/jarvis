# snaptix

> Hệ thống đặt vé **xe khách** và **tàu điện** chịu tải lớn, tính tiền và giữ chỗ chính xác tuyệt đối.

snaptix cho phép người dùng tìm chuyến, chọn ghế, đặt vé và thanh toán bằng ví nội bộ; đồng thời cung cấp cho nhà vận hành công cụ quản trị lịch chạy, giá vé và theo dõi thống kê kinh doanh.

Đây là dự án học tập theo hướng production-grade, nhằm rèn luyện lên level **Senior** với **Golang, PostgreSQL, Node.js, React**.

## Đặc điểm nổi bật

- **Chịu tải lớn** — xử lý các đợt mở bán cao điểm (Tết, lễ) với hàng nghìn lượt đặt vé mỗi giây.
- **Chính xác tuyệt đối** — không bán trùng ghế, không sai lệch một đồng nào trong ví và đơn hàng.
- **Tách biệt giao dịch và phân tích** — truy vấn thống kê nặng không ảnh hưởng luồng đặt vé.

## Tính năng

### Client (người dùng)
- Đăng nhập bằng **Google**
- Tìm chuyến theo tuyến, ngày, loại phương tiện (xe khách / tàu điện)
- Xem sơ đồ ghế realtime, chọn ghế, giữ chỗ tạm thời
- Đặt vé và thanh toán bằng **ví nội bộ**
- **Nạp tiền**, xem **số dư** và lịch sử giao dịch
- Quản lý vé: xem vé, mã QR, huỷ/hoàn vé theo chính sách

### Admin
- **Setup** tuyến, trạm/bến, phương tiện, sơ đồ ghế, lịch chạy, **slot**/chuyến
- Cấu hình giá vé theo tuyến, hạng ghế, khung giờ
- Quản lý người dùng, đơn hàng, hoàn tiền
- **Dashboard thống kê**: doanh thu, tỉ lệ lấp đầy, top tuyến, tỉ lệ huỷ

### Server
- Quản lý tồn kho ghế, giữ chỗ, đặt vé, chống bán trùng
- Sổ cái ví theo mô hình kế toán kép
- Phát sự kiện nghiệp vụ tin cậy cho các hệ thống khác

### Thống kê & phân tích
- Database phân tích tách riêng
- Báo cáo doanh thu, lấp đầy, hành vi đặt vé theo giờ/ngày/tuyến

## Công nghệ

| Thành phần | Stack |
|---|---|
| Web | React, Vite, TypeScript, shadcn/ui |
| BackToFront | Node.js, Fastify, TypeScript, MongoDB |
| Server | Golang, PostgreSQL |
| Thống kê | PostgreSQL |

## Tài liệu

Xem thư mục [`com/tm/docs/`](com/tm/docs/README.md):

- [Hướng dẫn sử dụng](com/tm/docs/user-guide/README.md) — dành cho người dùng và admin
- [Tài liệu kỹ thuật](com/tm/docs/technical/README.md) — kiến trúc, cấu trúc, API, service, database
