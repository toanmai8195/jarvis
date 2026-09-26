# Prometheus + cấu hình của snaptix (P0-T02). Cấu hình được COPY vào image
# (không bind mount), xem deploy/docker-compose.yml.
FROM prom/prometheus:v3.15.0
COPY prometheus.yml /etc/prometheus/prometheus.yml
