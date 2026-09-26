package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5/middleware"
)

// CodeInternal là mã lỗi của 500 theo api.md.
const CodeInternal = "INTERNAL"

// errorBody là body lỗi chung của API: {"error":{"code","message","details"}}.
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// recoverer biến panic trong handler thành 500 INTERNAL, log ERROR kèm giá
// trị panic và stack; process vẫn sống. Body không chứa giá trị panic hay
// stack (có thể lộ dữ liệu nội bộ).
//
// panic(http.ErrAbortHandler) được panic lại: net/http cắt kết nối và không
// log stack — đúng ý người gọi muốn huỷ response.
func recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(v)
				}
				log.ErrorContext(r.Context(), "panic trong handler",
					"panic", v,
					"stack", string(debug.Stack()),
				)
				// Header đã gửi thì không ghi đè status được (WriteHeader lần
				// hai chỉ sinh cảnh báo "superfluous"); chỉ log.
				if ww.Status() != 0 {
					return
				}
				writeError(ww, http.StatusInternalServerError, CodeInternal, "lỗi hệ thống, vui lòng thử lại sau")
			}()
			next.ServeHTTP(ww, r)
		})
	}
}

func writeError(w http.ResponseWriter, code int, errCode, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errorBody{Error: errorDetail{Code: errCode, Message: msg}})
}
