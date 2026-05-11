---
id: TASK-SEA-007
title: Implement Infrastructure Layer
feature: FEAT-SEA-007
status: Done
---

## Objective
Thực thi infrastructure layer cho graphiti-search dựa trên FEAT-SEA-007.

## Tasks
1. Tạo file `internal/infra/config/config.go`
   - Load cấu hình `STORE_ADDR`, `REDIS_URL`, `NATS_URL`, cache TTL, reranker weights từ env.
   - Add unit test cho validation config.

2. Tạo file `internal/infra/server/grpc.go`
   - Khởi tạo gRPC Server trên `:9022` và Health check `:9095`.
   - Setup graceful shutdown close connections (Redis, NATS, gRPC).

3. Tạo file `internal/infra/wire/wire.go`
   - Setup Google Wire providers cho adapters, rerankers factory và usecase.

4. Tạo file `cmd/server/main.go`
   - Bootstrap service với wire.
   - Kích hoạt NATS subscriber lúc startup.
   - Initialize OTel provider.

5. Docker & Scripts
   - Viết Dockerfile cho `graphiti-search`.

6. Code Coverage
   - Cập nhật test để coverage layer >= 70%.
