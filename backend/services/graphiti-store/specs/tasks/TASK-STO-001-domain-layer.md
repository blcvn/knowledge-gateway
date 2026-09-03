---
id: TASK-STO-001
title: Domain Layer — Graph Entity Types
feature: FEAT-STO-001
status: Done
---

## Objective
Thực thi implement domain layer cho graphiti-store (graph entity types, edge types, value objects, GraphDriver interface, và domain errors) dựa trên FEAT-STO-001.

## Tasks
1. Tạo file `internal/domain/entity.go`:
   - Implement các models: `EntityNode`, `EpisodicNode`, `CommunityNode`, `SagaNode`.
   - Bổ sung JSON tags cho serialization.

2. Tạo file `internal/domain/edge.go`:
   - Implement các models: `EntityEdge` (bi-temporal), `EpisodicEdge`.
   - Viết method `Validate()` cho `EntityEdge` để kiểm tra: `valid_at` required, `invalid_at > valid_at`.

3. Tạo file `internal/domain/value_object.go`:
   - Định nghĩa các types: `NodeLabel`, `EdgeType`, `GroupID`, `UUID`, `EmbeddingVector`.
   - `EmbeddingVector` type cần hỗ trợ tính toán cosine distance.

4. Tạo file `internal/domain/index.go`:
   - Định nghĩa các models: `IndexDefinition`, `IndexType`.

5. Tạo file `internal/domain/driver.go`:
   - Định nghĩa composite interface `GraphDriver` kế thừa 7 repository interfaces (NodeRepository, EdgeRepository, CommunityRepository, SearchRepository, IndexRepository, BulkRepository, TransactionManager) và `io.Closer`.

6. Tạo file `internal/domain/search.go`:
   - Định nghĩa các structs: `SearchParams`, `SearchResult`, `SimilarityMetric`.

7. Tạo file `internal/domain/errors.go`:
   - Định nghĩa các sentinel errors (hỗ trợ `errors.Is()`): `ErrNodeNotFound`, `ErrEdgeNotFound`, `ErrDriverNotSupported`, `ErrTransactionFailed`.

8. Unit Tests:
   - Viết unit tests cho Entity validation, value object constructors, bi-temporal rules.
   - Đảm bảo domain layer compile với ZERO external imports.
   - Đảm bảo test coverage >= 90%.
