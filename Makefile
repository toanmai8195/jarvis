# Makefile — lệnh dev local của snaptix (P0-T05). Hướng dẫn: com/tm/docs/technical/local-setup.md
#
#   make up        hạ tầng local: docker compose -f deploy/docker-compose.yml up -d --wait
#   make migrate   goose up cho PG core + PG analytics (scripts/migrate.sh)
#   make test      test không cần stack Docker: scripts, server (go vet, go test -race,
#                  bazel test //...), app (pnpm) — scripts/test-all.sh
#
# Tương thích GNU Make 3.81 (bản có sẵn trên macOS): không dùng .ONESHELL, !=, ::=...
# Đường dẫn tính theo vị trí Makefile, nên `make -C <repo> <target>` chạy được từ thư mục khác.

SHELL := /bin/bash

# Thư mục chứa Makefile (gốc repo), có `/` ở cuối.
ROOT := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

COMPOSE := docker compose -f $(ROOT)deploy/docker-compose.yml

# `make` không đối số chỉ in hướng dẫn: không bật/tắt stack, không chạy migration.
.DEFAULT_GOAL := help

.PHONY: help up migrate test

help:
	@echo "snaptix — lệnh dev local (chi tiết: com/tm/docs/technical/local-setup.md)"
	@echo "  make up       bật hạ tầng local (PG core/analytics, MongoDB, Redis, observability), chờ healthy"
	@echo "  make migrate  goose up cho PG core + analytics (CORE_DATABASE_URL, ANALYTICS_DATABASE_URL)"
	@echo "  make test     scripts/*_test.sh, check-structure, check-compose, go vet, go test -race,"
	@echo "                bazel test //..., pnpm lint/typecheck/test/build (không cần stack Docker)"
	@echo "Tắt stack: docker compose -f deploy/docker-compose.yml down   (thêm -v để xoá dữ liệu)"

up:
	$(COMPOSE) up -d --wait

migrate:
	bash $(ROOT)scripts/migrate.sh

test:
	bash $(ROOT)scripts/test-all.sh
