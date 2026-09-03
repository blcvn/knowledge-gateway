---
id: TASK-PIP-002
title: Implement Usecase Layer
feature: FEAT-PIP-002
status: Done
---

## Objective
Thực thi implement usecase layer cho saga orchestration và knowledge processing dựa trên FEAT-PIP-002.

## Tasks
1. Định nghĩa Port Interfaces
   - Tạo file `internal/usecase/port/` chứa interfaces (input + output) cho tất cả adapter dependencies.
   - Tạo DTO structs cho request/response mapping.

2. Saga Orchestrator logic
   - Implement `SagaOrchestrator` với 6-step pipeline.
   - Quản lý Saga state machine (QUEUED → PROCESSING → COMPLETED/FAILED) và compensating actions (RollbackBulk).
   - Implement `GroupLock` interface.

3. Ingestion Usecases
   - Implement `IngestEpisode` usecase: dedup → queue → saga pipeline.
   - Implement `BulkIngest` usecase: streaming batch processing with cross-episode dedup.

4. Knowledge Usecases
   - Implement `ExtractEntities` usecase: content → LLM → parsed entities.
   - Implement `ResolveEntities` usecase: search similar → LLM compare → merge/create.
   - Implement `ExtractEdges` usecase: episode + entities → LLM → temporal fact triples.
   - Implement `ResolveEdges` usecase: find contradictions → invalidate old edges.
   - Implement `GenerateEmbedding` usecase: text → vector via embedder client.
   - Implement `UpdateCommunity` usecase: label propagation → LLM summarization.

5. Unit Tests
   - Viết unit tests với mocked adapters.
   - Test Saga state machine transitions, dedup logic, group lock, LLM response parsing.
   - Đảm bảo coverage >= 80%.
   - Đảm bảo tất cả usecase methods chỉ phụ thuộc vào port interfaces.
