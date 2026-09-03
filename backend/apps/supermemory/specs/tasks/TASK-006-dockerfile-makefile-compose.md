---
id: TASK-006
title: Dockerfile + Makefile + docker-compose
app: apps/supermemory
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Chuẩn bị sẵn sàng các công cụ cho việc build, run và container hóa cho dự án monolith Supermemory.

## Bối Cảnh Nghiệp Vụ
Dự án cần có Makefile để gõ lệnh build/test nhanh gọn. Dockerfile để đóng gói thành duy nhất một container image nhỏ gọn, và docker-compose để khởi chạy ứng dụng cùng với các external dependencies (PostgreSQL, Redis, NATS) ở local environment.

## Scope
### In Scope
- Tạo `Makefile`.
- Tạo `Dockerfile` multi-stage build.
- Tạo `docker-compose.yml`.

### Out of Scope
- CI/CD pipeline (GitHub Actions).

## Thiết Kế Kỹ Thuật
- **Dockerfile**: Sử dụng Golang image cho builder stage và alpine (hoặc distroless) cho runtime stage.
- **docker-compose.yml**:
  - `postgres` (kèm pgvector)
  - `redis`
  - `nats`
  - `supermemory-app`

## Acceptance Criteria
- [ ] Lệnh `make build` và `make run` hoạt động
- [ ] Build thành công Docker image
- [ ] `docker-compose up -d` khởi động full stack ở môi trường dev

## Test Requirements
- Kiểm tra việc build image và run container thành công.
