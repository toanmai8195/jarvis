# Bài toán kỹ thuật cốt lõi

## 1. Chống bán trùng ghế khi traffic lớn

**Vấn đề**: hàng nghìn người cùng chọn một số ít ghế trong vài giây mở bán.

**Giải pháp nhiều lớp**:

1. **Giữ chỗ có TTL** — `trip_seats.status = HELD` + `seat_holds.expires_at`. Worker trả ghế hết hạn định kỳ; khi đọc cũng coi hold quá hạn là trống.
2. **Cập nhật có điều kiện, atomic**:
   ```sql
   UPDATE trip_seats
      SET status = 'HELD', hold_id = $1, updated_at = now()
    WHERE trip_id = $2
      AND seat_id = ANY($3)
      AND status = 'AVAILABLE'
   RETURNING seat_id;
   ```
   Số dòng trả về khác số ghế yêu cầu → rollback, trả `SEAT_UNAVAILABLE`. Luôn khoá ghế theo thứ tự `seat_id` để tránh deadlock.
3. **Chốt chặn cuối** — unique index `tickets (trip_id, seat_id) WHERE status <> 'CANCELLED'`.
4. **Giảm tải DB** — cache sơ đồ ghế ở Redis/BFF vài giây; pre-check ghế trong Redis trước khi vào PG.
5. **Hàng đợi ảo** (waiting room) cho sự kiện mở bán lớn: cấp token vào cửa theo tốc độ DB chịu được.

**Thực nghiệm**: so sánh `FOR UPDATE`, `FOR UPDATE SKIP LOCKED`, optimistic version, Redis lock bằng k6 — ghi kết quả vào ADR.

## 2. Tính tiền chính xác

- **Tiền là số nguyên** `bigint` VND. Không dùng float ở bất kỳ tầng nào; JSON truyền dạng string để JS không mất chính xác.
- **Double-entry ledger** — mỗi giao dịch có các entry tổng bằng 0:

  | Giao dịch | Entry |
  |---|---|
  | Nạp 500.000đ | `CASH_IN −500000`, `ví user +500000` |
  | Mua vé 250.000đ | `ví user −250000`, `REVENUE +250000` |
  | Hoàn 225.000đ | `REFUND_EXPENSE −225000`, `ví user +225000` |

- **Số dư** — `accounts.balance` cập nhật trong cùng transaction với entry; `CHECK (balance >= 0)` cho ví user chặn âm tiền kể cả khi code có bug.
- **Idempotency key** — lưu cả response; retry trả lại đúng kết quả cũ, không thực thi lại.
- **Giá chốt phía server** — client chỉ gửi ghế; core tự tính giá từ `fares` hợp lệ tại thời điểm đặt.
- **Đối soát** định kỳ:
  - `SUM(ledger_entries.amount) = 0` toàn hệ thống
  - `accounts.balance = SUM(entries)` cho từng tài khoản
  - topup `SUCCEEDED` khớp với giao dịch của cổng thanh toán

## 3. Nhất quán giữa các service

- **Transactional outbox** — sự kiện ghi cùng transaction nghiệp vụ; relay đọc bằng `FOR UPDATE SKIP LOCKED` nên chạy song song nhiều instance được.
- **At-least-once + consumer idempotent** — stats-worker ghi `processed_events` trong cùng transaction với dữ liệu analytics.
- **Webhook thanh toán** — xác thực chữ ký, idempotent theo `provider_ref`, và có job đối soát cho webhook bị mất.

## 4. Hiệu năng PostgreSQL

- Index theo đúng truy vấn: `trips (route_id, departure_at)`, partial index cho hold active và outbox chưa publish.
- Đọc `EXPLAIN (ANALYZE, BUFFERS)` cho mọi truy vấn nóng.
- `available_seats` denormalized trên `trips` để trang tìm kiếm không phải đếm `trip_seats`.
- Partition bảng lớn: `ledger_entries`, `outbox_events`, fact tables.
- Connection pool: pgxpool + PgBouncer (transaction mode); giới hạn số connection theo CPU của PG.
- Read replica cho tìm chuyến / lịch sử; ghi luôn vào primary.
- Hiểu isolation level: dùng `READ COMMITTED` + cập nhật có điều kiện; thử nghiệm `SERIALIZABLE` và xử lý lỗi `40001` bằng retry.

## 5. Thống kê

- Tách OLTP / OLAP để truy vấn nặng không tranh tài nguyên với luồng đặt vé.
- Aggregate cập nhật tăng dần (upsert) thay vì quét lại toàn bộ.
- Window function, CTE, BRIN index cho dữ liệu time-series.

## 6. Frontend

- Sơ đồ ghế realtime qua SSE; cập nhật optimistic, rollback khi nhận `SEAT_UNAVAILABLE`.
- Sinh `Idempotency-Key` phía client một lần cho mỗi lần bấm, tái sử dụng khi retry.
- Kết quả tìm chuyến cache ngắn ở BFF và TanStack Query; dữ liệu ví/vé luôn invalidate sau mutation.
