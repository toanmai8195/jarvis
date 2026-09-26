# Handbook

Ghi chú kỹ thuật **theo task**: kỹ thuật đã áp dụng, lý do, bẫy gặp phải, kèm link tới đoạn code thật. Agent viết sau khi task qua bước 5 (xem [`CLAUDE.md`](../../../../../CLAUDE.md#handbook)).

| | Handbook | Lessons learned |
|---|---|---|
| Cấp độ | Task | Phase |
| Người viết | Agent | Người dùng |
| Nội dung | Kỹ thuật cụ thể + link code | Nhìn lại quá trình, số liệu, điều làm khác |

## Cấu trúc

Mỗi task một file, nhóm theo phase:

```
handbook/
├── README.md              # file này: chỉ mục theo task và theo chủ đề
├── phase-0/
│   └── P0-T01.md
├── phase-1/
│   └── ...
└── phase-9/
```

## Theo task

| Task | Tên | File |
|---|---|---|
| P0-T01 | Khởi tạo cấu trúc `com/tm/{server,app,docs}`, `deploy/`, `loadtest/` | [phase-0/P0-T01.md](phase-0/P0-T01.md) |

## Chỉ mục theo chủ đề

Tag gợi ý: `go/channel` · `go/errgroup` · `go/context` · `go/generics` · `go/pprof` · `go/testing` · `pg/lock` · `pg/isolation` · `pg/index` · `pg/mvcc` · `pg/partition` · `node/event-loop` · `node/stream` · `node/fastify` · `mongo/index` · `react/state` · `react/memo` · `react/query` · `bazel` · ...

| Chủ đề | Task | Bài học | Link |
|---|---|---|---|
| `git` | P0-T01 | Git không track thư mục rỗng: `.gitkeep` hoặc README | [P0-T01.md](phase-0/P0-T01.md#git-không-track-thư-mục-rỗng-gitkeep-hoặc-readme) |
| `bash`, `testing` | P0-T01 | Script kiểm tra cấu trúc nhận ROOT làm tham số | [P0-T01.md](phase-0/P0-T01.md#script-kiểm-tra-cấu-trúc-nhận-root-làm-tham-số) |

## Mẫu một file

```markdown
# Handbook — P4-T03: Use case hold ghế

## Update có điều kiện thay cho SELECT ... FOR UPDATE
- **Chủ đề**: `pg/lock`, `pg/isolation`
- **Bối cảnh**: 500 request cùng giữ ghế A05.
- **Cách làm & lý do**: `UPDATE ... WHERE status = 'AVAILABLE' RETURNING`, so số dòng trả về; không cần khoá tường minh vì ...
- **Bẫy / lưu ý**: phải sắp `seat_id` trước khi update nhiều ghế, nếu không sẽ deadlock (tái hiện ở P4-AT14).
- **Code**: [booking/store.go#L30-L55 — holdSeats](../../../../server/services/core/internal/booking/store.go#L30-L55)
- **Tham khảo**: PostgreSQL docs — Explicit Locking
```
