# ADR-0002: Đăng nhập Google dùng mock OIDC provider ở local và test

- **Trạng thái**: Chấp nhận
- **Ngày**: 2026-09-26

## Bối cảnh

- Phase 2 yêu cầu đăng nhập Google (P2-FR1, P2-T02, challenge N3); test nghiệm thu P2-AT01..AT05 cần chạy trọn luồng OAuth.
- Dự án chạy tự động bằng chế độ execute-all: agent không tạo được OAuth client trên Google Cloud Console, không đăng nhập được tài khoản Google thật, và test E2E không được phụ thuộc mạng ngoài.

## Các phương án

1. **Google thật** — client ID/secret thật, test E2E đăng nhập tài khoản thật.
   - Ưu: giống production. Nhược: cần người thao tác, test không tái lập được, lộ credential vào CI.
2. **Stub route trong BFF** (bỏ qua OAuth, set session trực tiếp) — nhược: không học/không kiểm chứng được `state`, ID token, JWKS; mất ý nghĩa challenge N3.
3. **Mock OIDC provider chạy trong docker compose** (vd `ghcr.io/navikt/mock-oauth2-server`) — BFF vẫn chạy đủ luồng authorization code: redirect, `state`, đổi code, xác thực ID token qua JWKS.
   - Ưu: luồng OAuth thật, tái lập được, không cần credential. Nhược: khác Google ở chi tiết (claim `hd`, issuer).

## Quyết định

Chọn phương án 3. Provider cấu hình bằng biến môi trường (`OIDC_ISSUER`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`); local/CI trỏ tới mock, production trỏ tới `https://accounts.google.com`. Trường `google_sub` giữ nguyên tên, lấy từ claim `sub`.

## Hệ quả

- P2-AT01..AT05 chạy được tự động với mock; "Đăng nhập Google" trong mốc demo phase 2 hiểu là đăng nhập qua mock provider.
- Code BFF không được hard-code endpoint Google; phải đọc discovery document từ issuer.
- Chuyển sang Google thật chỉ cần đổi biến môi trường + tạo OAuth client — cần kiểm tra thủ công một lần trước khi deploy thật.
