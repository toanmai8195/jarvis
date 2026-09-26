# deploy

Hạ tầng local dùng chung cho `com/tm/server` và `com/tm/app`: Docker Compose (PostgreSQL core/analytics, MongoDB, Redis) và observability (otel-collector, Prometheus, Tempo, Grafana — cấu hình trong `deploy/observability/`).

Nội dung được thêm từ task P0-T02. Biến môi trường mẫu: `.env.example` (commit); `.env` là cấu hình cá nhân, bị `.gitignore`.

Xem [project-structure](../com/tm/docs/technical/project-structure.md) và [local-setup](../com/tm/docs/technical/local-setup.md).
