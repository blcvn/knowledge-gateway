---
id: TASK-KNW-001
title: Implement Domain Layer
feature: FEAT-KNW-001
status: Done
---

## Objective
Thực thi implement domain layer cho graphiti-knowledge dựa trên FEAT-KNW-001.

## Tasks
1. Tạo file `internal/domain/entity.go`
   - Định nghĩa `ExtractedEntity` với các fields: `Name`, `Label`, `Summary` cùng với JSON tags.
   - Định nghĩa `ExtractedEdge`.
   - Định nghĩa `Resolution` với `ExistingEntityID`, `ExtractedEntity`, `Decision`, `Confidence`.
   - Định nghĩa `DuplicateDecision` enum (`merge`, `create`, `skip`) và validate.

2. Tạo file `internal/domain/value_object.go`
   - Định nghĩa `PromptTemplate`, `ModelConfig`, `EmbeddingDimension`.
   - Định nghĩa `TokenUsage` (track prompt_tokens, completion_tokens, model).

3. Tạo file `internal/domain/embedding.go`
   - Định nghĩa `EmbeddingVector` (validate dimension matches configured value).
   - Định nghĩa `EmbeddingRequest`, `EmbeddingResult`.

4. Tạo file `internal/domain/community.go`
   - Định nghĩa `CommunityNode`, `CommunityMember`, `CommunityLevel`.

5. Tạo file `internal/domain/rerank.go`
   - Định nghĩa `RerankRequest`, `RerankResult`, `CrossEncoderScore`.

6. Tạo file `internal/domain/errors.go`
   - Định nghĩa `ErrLLMTimeout`, `ErrPromptTooLong`, `ErrProviderUnavailable`, `ErrMalformedLLMResponse`.

7. Unit Tests
   - Viết unit tests cho validation và type constructors.
   - Đảm bảo coverage >= 90%.
   - Đảm bảo domain layer không có dependency bên ngoài (ZERO external imports).
