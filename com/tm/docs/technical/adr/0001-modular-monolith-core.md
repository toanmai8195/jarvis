# ADR-0001: Core là modular monolith trong monorepo

- **Trạng thái**: Chấp nhận
- **Ngày**: 2026-09-26

## Bối cảnh

- snaptix là dự án học tập, một người phát triển. Mục tiêu chính là đào sâu **Go và PostgreSQL**.
- Luồng đặt vé gồm giữ ghế → trừ ví → phát vé → ghi outbox. Tính đúng đắn (không bán trùng, không sai tiền) là yêu cầu số một.
- Người phát triển đã có kinh nghiệm production với hệ phân tán (Kafka, Pub/Sub, gRPC, K8s); kinh nghiệm transaction quan hệ còn hạn chế.

## Các phương án

1. **Microservice ngay từ đầu** — `catalog`, `booking`, `wallet` là service riêng, mỗi service một DB.
   - Ưu: học tách service, deploy độc lập.
   - Nhược: luồng đặt vé thành saga + eventual consistency — vùng đã quen; mất bài toán transaction PG; tốn thời gian vào hạ tầng thay vì Go/PG.
2. **Monolith không ranh giới** — một package lớn.
   - Ưu: nhanh lúc đầu.
   - Nhược: không có đường tách sau này; dễ thành "big ball of mud".
3. **Modular monolith** — một service `core`, chia module theo nghiệp vụ với ranh giới chặt.
   - Ưu: luồng đặt vé là **một transaction ACID**; ranh giới rõ, tách được sau.
   - Nhược: cần kỷ luật giữ ranh giới; scale theo cả khối.

## Quyết định

Chọn **phương án 3**, trong **monorepo** (`com/tm/server` dùng Bazel, `com/tm/app` dùng pnpm).

Ranh giới giữa các service **có lý do** được giữ: web (tĩnh), bff (Node, nhiều kết nối), core (nguồn sự thật), stats-worker (tải analytics, cô lập lỗi).

Luật trong core:
- Module chỉ lộ `Service` và kiểu công khai; gọi nhau qua interface khai báo ở phía dùng.
- Module sở hữu bảng của mình, không truy vấn bảng module khác.
- Ngoại lệ có chủ đích: truyền `pgx.Tx` qua interface để dùng chung transaction.
- Ranh giới được kiểm tra tự động (Bazel visibility / test import).

## Hệ quả

- Dễ hơn: tính đúng đắn của đặt vé dựa vào transaction PG; debug một process; deploy đơn giản.
- Khó hơn: phải giữ kỷ luật ranh giới; không scale riêng từng module.
- Theo dõi: [Phase 13](../planning/phase-13-split-wallet/) (tuỳ chọn) tách `wallet` thành service riêng và đo so sánh với bản monolith để kiểm chứng quyết định này bằng số liệu.
