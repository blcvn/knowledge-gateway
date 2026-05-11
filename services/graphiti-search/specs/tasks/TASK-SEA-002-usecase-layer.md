---
id: TASK-SEA-002
title: Implement Usecase Layer — Hybrid Search Orchestrator
feature: FEAT-SEA-002
status: Done
---

## Objective
Thực thi implement usecase layer cho hybrid search dựa trên FEAT-SEA-002.

## Tasks
1. Port Interfaces
   - Định nghĩa `StoreSearchClient`, `EmbedderClient`, `CacheRepo`, `Reranker` ports trong `internal/usecase`.

2. Tạo file `internal/usecase/hybrid_search.go`
   - Implement `HybridSearchUseCase` struct với các ports tương ứng.
   - Implement hàm `Execute(ctx, query)` theo workflow (cache -> embedding -> parallel search -> merge dedup -> rerank -> cache -> return).

3. Tạo file `internal/usecase/node_search.go`
   - Implement entity-specific search usecase.

4. Tạo file `internal/usecase/edge_search.go`
   - Implement edge search với temporal filter.

5. Tạo file `internal/usecase/community_search.go`
   - Implement community-level search.

6. Unit Tests
   - Viết tests cho orchestrator workflow sử dụng mock ports.
   - Đảm bảo logic parallel execution, dedup và sequential reranking hoạt động đúng.
   - Coverage >= 80%.
