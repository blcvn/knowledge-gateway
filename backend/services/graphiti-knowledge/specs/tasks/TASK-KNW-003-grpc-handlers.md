---
id: TASK-KNW-003
title: Implement gRPC Handlers
feature: FEAT-KNW-003
status: Done
---

## Objective
Thực thi implement gRPC handlers cho GraphitiKnowledgeService dựa trên FEAT-KNW-003.

## Tasks
1. Tạo file `internal/adapter/grpc/handler.go`:
   - Implement 9 RPCs: ExtractEntities, ResolveEntities, ExtractEdges, ResolveEdges, GenerateEmbedding, GenerateEmbeddingBulk, Rerank, UpdateCommunity, GetTokenUsage.
   - Delegate requests cho usecase layer.
   - Return TokenUsage trong response.

2. Tạo file `internal/adapter/grpc/mapper.go`:
   - Proto <-> Domain bidirectional mapping.

3. Cross-cutting concerns:
   - Trích xuất `x-tenant-id` để gán vào `group_id`.
   - Setup OTel span cho mỗi RPC, lưu attribute model và template.

4. Unit Tests:
   - Viết unit test cho handler cùng mock usecases.
   - Đảm bảo cùng proto interface với graphiti-pipeline/knowledge.
   - Đảm bảo coverage >= 80%.
