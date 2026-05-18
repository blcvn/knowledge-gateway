---
id: TASK-005
title: Create Deployment Artifacts
service: zep
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Hoàn thiện cơ sở hạ tầng triển khai cho ứng dụng Zep monolith, bao gồm cấu hình container và công cụ build tự động hóa.

## Scope
### In Scope
- Tạo tệp `Dockerfile` đa giai đoạn (multi-stage build) cho thư mục `apps/zep`.
- Viết tệp `Makefile` để hỗ trợ build, test và run ứng dụng monolith.
- Xây dựng tệp `docker-compose.yml` (hoặc cập nhật file hiện tại) để khởi chạy toàn bộ system bao gồm Zep monolith và các external infra dependencies (PostgreSQL, Neo4j, Redis, NATS).

### Out of Scope
- Triển khai lên cụm Kubernetes.

## Business Logic / Technical Design
1. **Dockerfile**:
   - Sử dụng `golang:1.23-alpine` làm builder.
   - Build file nhị phân tĩnh (static binary) cho `apps/zep/cmd/main.go`.
   - Sử dụng scratch hoặc alpine làm base image cho production để giảm kích thước.
2. **Makefile**:
   - Cung cấp các lệnh: `make build`, `make run`, `make test`, `make docker-build`.
3. **docker-compose.yml**:
   - Khởi tạo 1 service chung `zep` từ image vừa build.
   - Định nghĩa các service: `postgres`, `neo4j`, `redis`, `nats`.
   - Cung cấp biến môi trường phù hợp.

## Acceptance Criteria
- [ ] AC-1: `make build` tạo ra executable file hoạt động tốt trên máy local.
- [ ] AC-2: `docker build` tạo thành công container image kích thước tối ưu.
- [ ] AC-3: `docker-compose up` khởi chạy toàn bộ monolith stack cùng với các database liên quan mà không bị exit error.

## Definition of Done
- [x] Code implement đủ Acceptance Criteria
- [x] Thử nghiệm `docker-compose up -d` hoạt động thành công cục bộ
