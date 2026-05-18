---
id: TASK-005
title: Build and Deployment Artifacts
package: apps/OpenViking
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Hoàn thiện hạ tầng triển khai cục bộ và production cho OpenViking Monolith, bao gồm Dockerfile, Makefile và cấu hình docker-compose.

## Scope
### In Scope
- Viết `Makefile` tại `apps/OpenViking/Makefile`.
- Viết `Dockerfile` (Multi-stage build) tại `apps/OpenViking/Dockerfile`.
- Viết cấu hình `docker-compose.yml` để cung cấp môi trường thử nghiệm độc lập cùng với DB, Redis, NATS.

### Out of Scope
- Triển khai trực tiếp lên cloud/server.

## Thiết Kế Kỹ Thuật
- **Dockerfile:** Sử dụng builder container (ví dụ `golang:1.23-alpine`) để biên dịch. Lưu ý rằng binary sẽ build cho entry point `apps/OpenViking/cmd/openviking/main.go`. Binary tạo ra chứa toàn bộ code của monolith. Output là minimal image (ví dụ `alpine` hoặc `scratch`).
- **Makefile:** Cung cấp các lệnh: `build`, `run`, `test`, `docker-build`, `clean`.
- **Docker Compose:** Định nghĩa 1 service `openviking` và các dependency bao gồm Redis, NATS, VectorDB(nếu cần, hoặc embedded tuỳ thiết kế).

## Acceptance Criteria
- [x] AC-1: `make build` biên dịch thành công file binary tại `apps/OpenViking/bin/openviking`.
- [x] AC-2: `docker build` tạo ra image hợp lệ và chạy thành công mà không gặp lỗi runtime liên quan đến thiếu dependency OS.
- [x] AC-3: Lệnh `docker-compose up` dựng toàn bộ stack monolith lên thành công.

## Test Requirements
- Tự động chạy build test trong CI.
