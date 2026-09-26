package httpx

import (
	"context"
	"net/http"
	"uuid"
)

// HeaderRequestID là header mang request ID giữa client, BFF và core.
// Go chuẩn hoá tên header thành "X-Request-Id"; HTTP không phân biệt hoa thường.
const HeaderRequestID = "X-Request-ID"

// maxRequestIDLen giới hạn độ dài request ID nhận từ client.
const maxRequestIDLen = 128

type requestIDKey struct{}

// RequestIDFrom trả request ID của request đang xử lý, hoặc "" nếu ctx không
// thuộc request nào (ví dụ log lúc khởi động).
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// withRequestID gắn request ID vào ctx.
func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// requestID nhận X-Request-ID hợp lệ của client hoặc sinh UUID v7 mới, trả
// lại trong header response và đặt vào ctx cho các middleware/handler sau.
// Giá trị không hợp lệ bị thay (không trả 400): request ID chỉ để truy vết,
// không được làm hỏng request, và không để client đưa chuỗi tuỳ ý vào log.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderRequestID)
		if !validRequestID(id) {
			id = uuid.NewV7().String()
		}
		w.Header().Set(HeaderRequestID, id)
		next.ServeHTTP(w, r.WithContext(withRequestID(r.Context(), id)))
	})
}

// validRequestID kiểm id khớp ^[A-Za-z0-9._:-]{1,128}$. Viết tay thay vì
// regexp: chạy trên mọi request, và tập ký tự đủ đơn giản.
func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > maxRequestIDLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
		case c == '.', c == '_', c == ':', c == '-':
		default:
			return false
		}
	}
	return true
}
