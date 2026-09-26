# Grafana + provisioning datasource Prometheus/Tempo (P0-T02) và dashboard (P0-T06).
# Cấu hình COPY vào image, không bind mount (Docker Desktop treo khi mount ~/Documents).
FROM grafana/grafana:13.2.2

# Grafana 13 tách plugin datasource prometheus/tempo khỏi core: mặc định chúng được
# tải ngầm (background installer) vài giây SAU khi /api/health đã OK, nên ngay sau
# `up --wait` mọi query trả 404 "plugin not registered". Cài sẵn lúc build (bản pin)
# để container healthy là dashboard dùng được ngay, không cần mạng lúc chạy.
RUN grafana cli plugins install prometheus 13.2.1 \
 && grafana cli plugins install tempo 13.2.2

COPY provisioning /etc/grafana/provisioning
COPY dashboards /etc/grafana/dashboards
