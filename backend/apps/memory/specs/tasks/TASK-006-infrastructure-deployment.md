---
id: TASK-006
title: Deployment & Infrastructure Setup
service: apps-memory
status: Done
priority: P2
created: 2026-05-14
---

# TASK-006: Deployment & Infrastructure Setup

## 1. Mục Tiêu
Chuẩn hóa quy trình build và deploy cho monolith application bằng Docker, Docker Compose, và Makefile.
**Tối ưu token:** Nhóm tất cả các file cấu hình dev/ops vào 1 task.

## 2. Các Bước Thực Thi

1. **`apps/memory/Dockerfile`**:
   - Tạo Multi-stage Dockerfile sử dụng `golang:1.25-alpine` để build và `alpine:3.21` làm runtime image.
   - Copy `go.work`, `go.work.sum`, cùng tất cả thư mục service để build thành công.
2. **`apps/memory/docker-compose.yml`**:
   - Khởi tạo file compose chứa container `vnp-memory` (port 8080, 8082, 8083) và 5 infra containers (PostgreSQL, Neo4j, Qdrant, Redis, MinIO). NATS không cần thiết vì đã embedded.
3. **`apps/memory/docker-compose.infra.yml`**:
   - File riêng biệt chỉ chạy 5 infra containers để developer chạy `go run local`.
4. **`apps/memory/Makefile`**:
   - Các lệnh chuẩn: `build`, `run`, `dev`, `docker`, `infra-up`, `infra-down`.

## 3. Acceptance Criteria
- [ ] `make infra-up` khởi chạy đủ 5 database/cache containers.
- [ ] `docker build -t vnp-memory .` thành công (Go workspace module được resolve chính xác).
- [ ] `docker compose up` khởi động toàn hệ thống với duy nhất 6 containers tổng cộng.
