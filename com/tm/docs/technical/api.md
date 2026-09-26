# API

Có hai lớp API:

- **Public API (BFF)** — `https://<host>/api/...`, dùng bởi web-client và web-admin.
- **Internal API (core)** — `http://core:8080/internal/v1/...`, chỉ BFF gọi.

Hợp đồng chi tiết: `com/tm/app/api/bff.openapi.yaml`, `com/tm/server/api/core.openapi.yaml`.

## Quy ước chung

| Hạng mục | Quy ước |
|---|---|
| Định dạng | JSON, `snake_case` |
| Tiền | Chuỗi số nguyên VND, ví dụ `"250000"` |
| Thời gian | RFC 3339 UTC |
| Phân trang | Cursor: `?limit=20&cursor=...` → `{ "data": [], "next_cursor": "..." }` |
| Idempotency | Header `Idempotency-Key` (UUID) **bắt buộc** với các lệnh tạo hold, booking, topup, cancel, refund |
| Trace | Header `traceparent` |
| Request ID | Header `X-Request-ID`: client gửi giá trị khớp `^[A-Za-z0-9._:-]{1,128}$` thì được giữ nguyên, thiếu hoặc không hợp lệ thì server sinh UUID v7 (không trả 400). Mọi response (kể cả lỗi) có header này; log của request mang cùng `request_id` |

### Lỗi

```json
{
  "error": {
    "code": "SEAT_UNAVAILABLE",
    "message": "Ghế A05 đã được giữ",
    "details": { "seat_ids": ["A05"] }
  }
}
```

| HTTP | Code | Ý nghĩa |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Dữ liệu không hợp lệ |
| 401 | `UNAUTHENTICATED` | Chưa đăng nhập |
| 403 | `FORBIDDEN` | Không đủ quyền |
| 404 | `NOT_FOUND` | Không tồn tại |
| 409 | `SEAT_UNAVAILABLE` | Ghế đã được giữ / bán |
| 409 | `HOLD_EXPIRED` | Giữ chỗ đã hết hạn |
| 409 | `IDEMPOTENCY_CONFLICT` | Cùng key nhưng khác nội dung request |
| 422 | `INSUFFICIENT_BALANCE` | Số dư không đủ |
| 422 | `CANCEL_NOT_ALLOWED` | Quá hạn huỷ |
| 429 | `RATE_LIMITED` | Vượt giới hạn |
| 500 | `INTERNAL` | Lỗi hệ thống (vd panic trong handler). `message` chung chung, không chứa chi tiết nội bộ; tra log theo `X-Request-ID` |

## Public API (BFF)

### Auth
| Method | Path | Mô tả |
|---|---|---|
| GET | `/api/auth/google` | Bắt đầu đăng nhập Google (redirect) |
| GET | `/api/auth/google/callback` | Callback OAuth, tạo session |
| POST | `/api/auth/logout` | Đăng xuất, xoá session |
| GET | `/api/me` | Thông tin người dùng hiện tại |

### Tìm chuyến
| Method | Path | Mô tả |
|---|---|---|
| GET | `/api/stations?q=&type=` | Tìm trạm |
| GET | `/api/trips?from=&to=&date=&type=` | Tìm chuyến |
| GET | `/api/trips/{trip_id}` | Chi tiết chuyến, giá theo hạng ghế |
| GET | `/api/trips/{trip_id}/seats` | Sơ đồ và trạng thái ghế |
| GET | `/api/trips/{trip_id}/seats/stream` | SSE cập nhật trạng thái ghế |

### Đặt vé
| Method | Path | Mô tả |
|---|---|---|
| POST | `/api/holds` | Giữ ghế |
| DELETE | `/api/holds/{hold_id}` | Trả ghế |
| POST | `/api/bookings` | Đặt vé từ hold, thanh toán bằng ví |
| GET | `/api/bookings?status=` | Vé của tôi |
| GET | `/api/bookings/{booking_id}` | Chi tiết đơn / vé |
| POST | `/api/bookings/{booking_id}/cancel` | Huỷ vé |
| GET | `/api/bookings/{booking_id}/refund-quote` | Xem trước số tiền hoàn |

Ví dụ:

```http
POST /api/holds
Idempotency-Key: 5f1c...

{ "trip_id": "0192...", "seat_ids": ["A05", "A06"] }
```

```json
201
{ "hold_id": "0192...", "expires_at": "2026-09-26T10:10:00Z", "seats": ["A05", "A06"] }
```

```http
POST /api/bookings
Idempotency-Key: 9a2e...

{
  "hold_id": "0192...",
  "passengers": [
    { "seat_id": "A05", "full_name": "Nguyen Van A", "phone": "0900000000" },
    { "seat_id": "A06", "full_name": "Tran Thi B", "phone": "0900000001" }
  ]
}
```

```json
201
{
  "booking_id": "0192...",
  "status": "CONFIRMED",
  "total_amount": "500000",
  "tickets": [{ "ticket_id": "...", "seat_id": "A05", "qr_code": "..." }]
}
```

### Ví
| Method | Path | Mô tả |
|---|---|---|
| GET | `/api/wallet` | Số dư |
| GET | `/api/wallet/transactions` | Lịch sử giao dịch |
| POST | `/api/wallet/topups` | Tạo lệnh nạp tiền, trả `payment_url` |
| GET | `/api/wallet/topups/{topup_id}` | Trạng thái lệnh nạp |

### Admin (`/api/admin/*`, yêu cầu vai trò)
| Method | Path | Vai trò |
|---|---|---|
| GET/POST/PATCH | `/api/admin/stations` | operator |
| GET/POST/PATCH | `/api/admin/seat-layouts` | operator |
| GET/POST/PATCH | `/api/admin/vehicles` | operator |
| GET/POST/PATCH | `/api/admin/routes` | operator |
| GET/POST/PATCH | `/api/admin/fares` | operator |
| GET/POST | `/api/admin/schedules` | operator |
| POST | `/api/admin/schedules/{id}/preview` | operator |
| GET/PATCH | `/api/admin/trips/{id}` | operator |
| POST | `/api/admin/trips/{id}/cancel` | operator |
| POST | `/api/admin/trips/{id}/block-seats` | operator |
| GET | `/api/admin/bookings` | support |
| POST | `/api/admin/bookings/{id}/refund` | support |
| GET/PATCH | `/api/admin/users/{id}` | support |
| GET | `/api/admin/stats/revenue?from=&to=&group_by=` | analyst |
| GET | `/api/admin/stats/occupancy` | analyst |
| GET | `/api/admin/stats/top-routes` | analyst |
| GET | `/api/admin/stats/export?report=` | analyst |
| GET | `/api/admin/audit-logs` | super_admin |

## Internal API (core)

Header bắt buộc: `Authorization: Bearer <service-token>`, `X-User-Id`, `X-User-Role`.

| Method | Path | Mô tả |
|---|---|---|
| POST | `/internal/v1/users` | Tạo user + ví (idempotent) |
| GET | `/internal/v1/trips/search` | Tìm chuyến |
| GET | `/internal/v1/trips/{id}` | Chi tiết chuyến |
| GET | `/internal/v1/trips/{id}/seats` | Trạng thái ghế |
| POST | `/internal/v1/holds` | Giữ ghế |
| DELETE | `/internal/v1/holds/{id}` | Trả ghế |
| POST | `/internal/v1/bookings` | Xác nhận đặt vé |
| GET | `/internal/v1/bookings` | Danh sách đơn |
| GET | `/internal/v1/bookings/{id}` | Chi tiết đơn |
| POST | `/internal/v1/bookings/{id}/cancel` | Huỷ vé |
| GET | `/internal/v1/wallets/{user_id}` | Số dư |
| GET | `/internal/v1/wallets/{user_id}/entries` | Bút toán |
| POST | `/internal/v1/topups` | Tạo lệnh nạp |
| POST | `/internal/v1/payments/webhook` | Webhook cổng thanh toán (xác thực HMAC, không cần service token) |
| POST | `/internal/v1/refunds` | Hoàn tiền thủ công |
| `*` | `/internal/v1/admin/...` | CRUD catalog, fare, schedule, trip — tương ứng Admin API |
| GET | `/healthz`, `/readyz` | Health check |
| GET | `/metrics` | Prometheus |

## Rate limit (BFF)

| Nhóm | Giới hạn |
|---|---|
| Tìm chuyến (ẩn danh) | 30 req / 10s / IP |
| Hold | 10 req / phút / user |
| Booking, topup | 5 req / phút / user |
| Admin | 300 req / phút / user |
