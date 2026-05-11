---
id: TASK-SEA-001
title: Phase 1 - Domain & Usecase Layer Implementation
service: cognee-search
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-11
updated: 2026-05-11
linked_features: [FEAT-SEA-001]
---

# Kế Hoạch Triển Khai: Phase 1 - Domain & Usecase Layer

### Task 1.1: Thiết lập Domain Entities, Value Objects và Errors
- **File(s)**: `internal/domain/entity.go`, `internal/domain/value_object.go`, `internal/domain/errors.go`
- **Chi tiết**:
  - Implement Domain Entities: `SearchResult`, `RetrieverConfig`, `RerankScore`, `SearchSession`.
  - Implement Domain Value Objects: `SearchStrategy` (15 hằng số), `ResultType`, `SearchScope`.
  - Implement Errors: `StrategyNotFoundError`, `EmptyQueryError`.

### Task 1.2: Định nghĩa UseCase Ports (Interfaces)
- **File(s)**: `internal/usecase/port/output.go`, `internal/usecase/port/input.go`
- **Chi tiết**:
  - Định nghĩa Output Ports: `Retriever` interface, `VectorSearcher`, `GraphSearcher`, `Reranker`, `LLMClient`, `CacheStore`.
  - Định nghĩa Input Ports: `SearchUseCase`, `RAGCompleteUseCase`, `ExploreGraphUseCase`.

### Task 1.3: Định nghĩa UseCase DTOs
- **File(s)**: `internal/usecase/dto/request.go`, `internal/usecase/dto/response.go`
- **Chi tiết**:
  - Request DTOs: `SearchRequest`, `RAGRequest`, `ExploreRequest`.
  - Response DTOs: `SearchResponse`, `RAGResponse`, `ExploreResponse`.

### Task 1.4: Implement SearchUseCase (3-Phase Pipeline)
- **File(s)**: `internal/usecase/search.go`, `internal/usecase/merge.go`
- **Chi tiết**:
  - Implement hàm `Execute` như là một orchestrator với errgroup:
    - Phase 1 (Retrieve): Chạy song song các strategies thông qua registry mapping.
    - Phase 2 (Merge): Gọi hàm logic RRF (Reciprocal Rank Fusion).
    - Phase 3 (Rerank): Gọi Reranker port nếu `rerank` là true.
  - Trả về `SearchResponse`.

### Task 1.5: Implement RAGCompleteUseCase và ExploreGraphUseCase
- **File(s)**: `internal/usecase/rag_complete.go`, `internal/usecase/explore_graph.go`
- **Chi tiết**:
  - Implement logic cho `RAGCompleteUseCase` và `ExploreGraphUseCase`.

### Task 1.6: Unit Tests cho Phase 1
- **File(s)**: `internal/domain/*_test.go`, `internal/usecase/*_test.go`
- **Chi tiết**:
  - Test Orchestrator, thuật toán RRF scoring, đảm bảo Code Coverage >= 80%.
