# otel-collector cho local (P0-T02).
# Image gốc distroless: không có shell/wget, CLI không có lệnh health.
# Chép binary collector sang alpine đã pin để healthcheck gọi được
# `wget http://127.0.0.1:13133/` (extension health_check).
FROM otel/opentelemetry-collector-contrib:0.161.0 AS otelcol

FROM alpine:3.24.2
COPY --from=otelcol /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=otelcol /otelcol-contrib /otelcol-contrib
COPY config.yaml /etc/otelcol-contrib/config.yaml
USER 10001:10001
EXPOSE 4317 4318 8888 13133
ENTRYPOINT ["/otelcol-contrib"]
CMD ["--config", "/etc/otelcol-contrib/config.yaml"]
