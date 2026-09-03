---
id: TASK-007
title: Integration, Polish & Readme
service: apps-memory
status: Done
priority: P2
created: 2026-05-14
---

# TASK-007: Integration, Polish & Readme

## 1. Mục Tiêu
Hoàn thiện dự án với các bài test tích hợp (E2E), tổng hợp health check endpoint và documentation cho developer.

## 2. Các Bước Thực Thi

1. **Health Aggregation**:
   - Update observability server trên port `8083`.
   - Loop qua danh sách 35 services trong `InProcessRegistry` để trả về status `OK` cho `/healthz`.
2. **Graceful Shutdown**:
   - Đảm bảo `main.go` lắng nghe `SIGTERM`, `SIGINT`.
   - Drain NATS JetStream, Stop gRPC Server, Close DB Connection Pools theo thứ tự an toàn.
3. **E2E Smoke Tests (`tests/e2e_test.go`)**:
   - Viết 1 bài test đơn giản: start monolithic app (in-memory test db hoặc mocks nếu cần), gọi 1 REST API endpoint qua gateway (ví dụ: POST `/v1/memory/ingest`), và xác nhận HTTP 200 OK.
4. **`apps/memory/README.md`**:
   - Viết tài liệu hướng dẫn quickstart: cách chạy `make infra-up`, cách config `configs/config.yaml`, và `go run ./cmd/server`.

## 3. Acceptance Criteria
- [ ] Ctrl+C tắt server an toàn, không rò rỉ NATS/DB connection.
- [ ] `curl localhost:8083/healthz` list đầy đủ 35 services.
- [ ] README.md đầy đủ hướng dẫn setup.
