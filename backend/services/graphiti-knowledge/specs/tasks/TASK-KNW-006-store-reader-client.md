---
id: TASK-KNW-006
title: Implement Store Reader Client
feature: FEAT-KNW-006
status: Done
---

## Objective
Thực thi implement gRPC client (read-only) giao tiếp với graphiti-store dựa trên FEAT-KNW-006.

## Tasks
1. Tạo file `internal/adapter/client/store_client.go`:
   - Implement `GraphReader` port.
   - Implement `FindSimilarEntities`: gọi graphiti-store CosineSimilaritySearch qua gRPC trên `name_embedding`.
   - Implement `FindSimilarEdges`: gọi cosine search trên `fact_embedding`.
   - Implement `GetEntityByName`: tra cứu chính xác trong phạm vi `group_id`.

2. Resilience & Observability:
   - Circuit breaker: mở sau 5 lần lỗi.
   - Deadline propagation.
   - OTel trace spans cho các outgoing calls.

3. Unit Tests:
   - Dùng mock gRPC server.
   - Đảm bảo coverage >= 80%.
