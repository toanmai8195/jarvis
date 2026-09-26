# Ghi chú execute-all

Việc ngoài phạm vi phát hiện trong lúc chạy execute-all, để báo cáo cuối.

## P0-T01

- Gợi ý cho P0-T04 (CI) / P0-T05 (Makefile): gọi `bash scripts/check-structure_test.sh` và `bash scripts/check-structure.sh` để luật cấu trúc được kiểm tự động. Không làm trong P0-T01.
- TC08 của P0-T01 (không có `MODULE.bazel`, `go.mod`, `package.json`, `docker-compose.yml`, `Makefile`, `.github/workflows`, `com/tm/server/db`) chỉ đúng tại commit của P0-T01; không chạy lại khi đóng phase.
