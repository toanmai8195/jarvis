# Hướng dẫn cho quản trị viên

## 1. Đăng nhập và phân quyền

Admin đăng nhập bằng Google tại trang quản trị. Tài khoản phải được cấp quyền trước.

| Vai trò | Quyền |
|---|---|
| **Super admin** | Toàn quyền, quản lý tài khoản admin |
| **Operator** | Setup tuyến, phương tiện, lịch chạy, giá vé |
| **Support** | Xem đơn hàng, người dùng; xử lý hoàn tiền |
| **Analyst** | Chỉ xem thống kê, báo cáo |

## 2. Setup dữ liệu nền

Thực hiện theo thứ tự: **Trạm → Phương tiện & sơ đồ ghế → Tuyến → Giá vé → Lịch chạy/slot**.

### 2.1. Trạm / bến
**Danh mục → Trạm** → **Thêm trạm**: tên, mã, loại (bến xe / ga tàu), địa chỉ, toạ độ.

### 2.2. Sơ đồ ghế
**Danh mục → Sơ đồ ghế** → **Tạo sơ đồ**:
1. Chọn loại: xe ghế ngồi, xe giường nằm, toa tàu.
2. Khai báo số tầng, số hàng, số cột.
3. Kéo thả để đánh dấu ghế, lối đi, vị trí trống.
4. Gán **hạng ghế** (thường, VIP, giường tầng dưới/trên...) cho từng ghế.

> Sơ đồ đã được dùng cho chuyến có vé bán ra thì không sửa được; hãy tạo phiên bản mới.

### 2.3. Phương tiện
**Danh mục → Phương tiện** → **Thêm**: biển số / mã đoàn tàu, loại, sơ đồ ghế, trạng thái (hoạt động / bảo trì).

### 2.4. Tuyến
**Danh mục → Tuyến** → **Thêm tuyến**:
1. Chọn loại phương tiện.
2. Thêm các trạm theo thứ tự, kèm thời gian di chuyển dự kiến từ trạm đầu.
3. Lưu. Tuyến có thể bật/tắt mà không xoá dữ liệu.

### 2.5. Giá vé
**Giá vé** → chọn tuyến:
- Giá cơ bản theo **đoạn** (trạm đi → trạm đến) và **hạng ghế**.
- Quy tắc điều chỉnh: phụ thu cuối tuần, ngày lễ, khung giờ cao điểm (theo % hoặc số tiền cố định).
- Thay đổi giá chỉ áp dụng cho vé bán **sau** thời điểm lưu; vé đã bán giữ nguyên giá.

### 2.6. Lịch chạy và slot
**Lịch chạy** → **Tạo lịch**:
1. Chọn tuyến, phương tiện, giờ khởi hành.
2. Chọn lặp lại: một lần, hằng ngày, các ngày trong tuần, khoảng ngày.
3. Chọn thời điểm **mở bán** (ví dụ trước 30 ngày).
4. Xem trước danh sách slot sẽ sinh ra → **Xác nhận**.

Thao tác với từng slot:
- **Tạm dừng bán** / **Mở bán lại**
- **Đổi phương tiện** (chỉ khi sơ đồ ghế tương thích với các ghế đã bán)
- **Huỷ chuyến**: toàn bộ vé được huỷ và **hoàn 100%** vào ví khách hàng
- **Khoá ghế**: giữ lại ghế cho mục đích nội bộ

## 3. Quản lý đơn hàng

**Đơn hàng** — tìm theo mã đơn, email, số điện thoại, chuyến, khoảng thời gian.
- Xem chi tiết: ghế, hành khách, lịch sử trạng thái, giao dịch ví liên quan.
- **Hoàn tiền thủ công** (vai trò Support): nhập số tiền và lý do; mọi thao tác được ghi audit log.

## 4. Quản lý người dùng

**Người dùng** — xem hồ sơ, số dư ví, lịch sử giao dịch, lịch sử đặt vé; khoá/mở khoá tài khoản.

## 5. Thống kê

**Dashboard** gồm:

| Chỉ số | Mô tả |
|---|---|
| Doanh thu | Theo ngày/tuần/tháng, theo tuyến, theo loại phương tiện |
| Số vé bán | Số vé bán, huỷ, tỉ lệ huỷ |
| Tỉ lệ lấp đầy | % ghế đã bán trên tổng ghế, theo chuyến/tuyến/khung giờ |
| Top tuyến | Tuyến doanh thu cao nhất, lấp đầy cao nhất |
| Nạp tiền | Tổng nạp, số dư ví toàn hệ thống |
| Hành vi | Thời gian đặt trước trung bình, khung giờ đặt vé cao điểm |

- Bộ lọc: khoảng thời gian, tuyến, loại phương tiện.
- **Xuất CSV** cho mọi bảng.
- Số liệu cập nhật gần realtime (trễ tối đa vài phút).

## 6. Audit log

**Hệ thống → Audit log** — mọi thao tác thay đổi dữ liệu của admin (ai, lúc nào, thay đổi gì) đều được lưu và không thể sửa/xoá.
