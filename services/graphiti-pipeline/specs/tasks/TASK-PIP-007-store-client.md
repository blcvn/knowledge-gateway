---
id: TASK-PIP-007
title: Implement graphiti-store gRPC Client
feature: FEAT-PIP-007
status: Done
---

## Objective
Thực thi implement gRPC client adapter kết nối graphiti-pipeline -> graphiti-store dựa trên FEAT-PIP-007.

## Tasks
1. Tạo file `internal/adapter/client/store_client.go`
   - Implement `StoreClient` port.

2. Implement Store Client methods
   - `SaveBulk`: gửi atomic nodes + edges + episode.
   - `RollbackBulk`: cleanup partial writes cho saga compensation.

3. Resiliency và Telemetry
   - Tích hợp circuit breaker (gobreaker) với cấu hình theo spec.
   - Tích hợp gRPC deadline propagation từ parent context.
   - Tích hợp OTel span injection vào outgoing gRPC metadata.

4. Tests
   - Viết unit tests cho StoreClient sử dụng mock gRPC server.
   - Test các circuit breaker state transitions (closed -> open -> half-open -> closed).
   - Đảm bảo coverage >= 80%.
