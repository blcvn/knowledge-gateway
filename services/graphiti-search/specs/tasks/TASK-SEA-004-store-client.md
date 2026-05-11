---
id: TASK-SEA-004
title: Implement graphiti-store Client Adapter
feature: FEAT-SEA-004
status: Done
---

## Objective
Thực thi implement gRPC client adapter kết nối `graphiti-store` dựa trên FEAT-SEA-004.

## Tasks
1. Tạo file `internal/adapter/client/store_client.go`
   - Implement interface `StoreSearchClient`.
   - Implement `CosineSimilaritySearch`.
   - Implement `FulltextSearch` (BM25).
   - Implement `BFSSearch`.

2. Resilience & Observability
   - Setup Circuit Breaker mở sau 5 lần failures liên tiếp.
   - Setup Deadline propagation từ context.
   - Thêm OTel trace injection vào outbound context.

3. Unit Tests
   - Test gRPC client calls, errors, và circuit breaker.
   - Coverage >= 80%.
