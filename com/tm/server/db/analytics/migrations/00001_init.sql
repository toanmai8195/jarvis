-- Migration khởi đầu của PG analytics (P0-T03): rỗng, chỉ để goose tạo bảng
-- goose_db_version và xác nhận đường chạy migration. Không tạo/sửa object nào;
-- schema nghiệp vụ thêm ở các migration sau (database.md).
-- Quy ước: chỉ thêm migration mới, không sửa migration đã merge.

-- +goose Up
SELECT 1;

-- +goose Down
SELECT 1;
