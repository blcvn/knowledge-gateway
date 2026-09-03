---
id: TASK-SEA-001
title: Implement Domain Layer — Search Types + Reranker Interfaces
feature: FEAT-SEA-001
status: Done
---

## Objective
Thực thi implement domain layer cho graphiti-search dựa trên FEAT-SEA-001.

## Tasks
1. Tạo file `internal/domain/entity.go`
   - Định nghĩa `SearchQuery`.
   - Định nghĩa `SearchResult`.
   - Định nghĩa `RankedResult`.

2. Tạo file `internal/domain/value_object.go`
   - Định nghĩa type `SearchMethod` enum (`MethodCosine`, `MethodBM25`, `MethodBFS`).
   - Định nghĩa type `RerankerType` enum (`RerankerRRF`, `RerankerMMR`, `RerankerCrossEncoder`, `RerankerNodeDistance`, `RerankerEpisodeMentions`).
   - Định nghĩa `ScoreWeight`.
   - Định nghĩa `TemporalWindow` với validation (`from < to`).

3. Tạo file `internal/domain/config.go`
   - Định nghĩa `SearchConfig`.
   - Định nghĩa `RerankerConfig`.
   - Định nghĩa `CacheConfig`.

4. Tạo file `internal/domain/errors.go`
   - Định nghĩa `ErrNoResults`, `ErrInvalidQuery`, `ErrCacheUnavailable`.

5. Validation & Constraints
   - Implement `SearchQuery.Validate()`: required fields, >0 limits, at least 1 method.
   - Implement `SearchMethod` và `RerankerType` validation.

6. Unit Tests
   - Viết unit tests cho validation, constructors.
   - Đảm bảo ZERO external imports.
   - Coverage >= 90%.
