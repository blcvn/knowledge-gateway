---
id: TASK-PIP-001
title: Implement Domain Layer
feature: FEAT-PIP-001
status: Done
---

## Objective
Thực thi implement domain layer cho graphiti-pipeline bao gồm cả ingestion và knowledge sub-domains dựa trên FEAT-PIP-001.

## Tasks
1. Tạo file `internal/domain/ingestion/entity.go`
   - Định nghĩa `Episode`, `EpisodeType` enum (`message`, `json`, `text`, `fact_triple`).
   - Định nghĩa `Saga`, `SagaState`, `PipelineStep` enum.

2. Tạo file `internal/domain/ingestion/value_object.go`
   - Định nghĩa `GroupID`, `EpisodeID`, `ContentHash`.
   - Implement `String()` và `Validate()`.

3. Tạo file `internal/domain/ingestion/event.go`
   - Định nghĩa domain events: `EpisodeIngested`, `EpisodeFailed`.

4. Tạo file `internal/domain/ingestion/errors.go`
   - Định nghĩa sentinel errors (e.g., `ErrDuplicateEpisode`, `ErrPipelineFailed`).

5. Tạo file `internal/domain/knowledge/entity.go`
   - Định nghĩa `ExtractedEntity`, `ExtractedEdge`, `Resolution`.

6. Tạo file `internal/domain/knowledge/value_object.go`
   - Định nghĩa `PromptTemplate`, `TokenUsage`, `ModelConfig`.

7. Tạo file `internal/domain/knowledge/embedding.go`
   - Định nghĩa `EmbeddingVector`, `EmbeddingRequest`.

8. Tạo file `internal/domain/knowledge/community.go`
   - Định nghĩa `CommunityNode`, `CommunityEdge`, `CommunityLevel`.

9. Tạo file `internal/domain/knowledge/errors.go`
   - Định nghĩa sentinel errors (e.g., `ErrLLMTimeout`, `ErrPromptTooLong`).

10. Unit Tests
   - Viết unit tests cho validation methods, value object constructors, và edge bi-temporal rules.
   - Đảm bảo coverage >= 90%.
   - Đảm bảo domain layer ZERO external imports (chỉ Go stdlib).
