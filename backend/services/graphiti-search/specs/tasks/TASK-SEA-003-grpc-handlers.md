---
id: TASK-SEA-003
title: Implement gRPC Handlers — Search Service
feature: FEAT-SEA-003
status: Done
---

## Objective
Thực thi implement gRPC handlers cho GraphitiSearchService dựa trên FEAT-SEA-003.

## Tasks
1. Tạo file `internal/adapter/grpc/handler.go`
   - Định nghĩa server struct struct map với protobuf interface của Search Service.
   - Implement `HybridSearch` rpc.
   - Implement `NodeSearch` rpc.
   - Implement `EdgeSearch` rpc.
   - Implement `CommunitySearch` rpc.

2. Tạo file `internal/adapter/grpc/mapper.go`
   - Implement các hàm mapping giữa Protobuf generated types và Domain types.
   - Đảm bảo mapping Error Codes từ Domain Errors sang gRPC status.

3. Context & Middleware Integration
   - Extract tenant thông tin (`x-tenant-id` thành `group_id`).
   - Đảm bảo OTel span propagation.

4. Unit Tests
   - Test mapping logic và RPC handlers với mock usecases.
   - Coverage >= 80%.
