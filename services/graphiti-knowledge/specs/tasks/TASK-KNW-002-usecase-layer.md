---
id: TASK-KNW-002
title: Implement Usecase Layer
feature: FEAT-KNW-002
status: Done
---

## Objective
Thực thi implement usecase layer cho graphiti-knowledge dựa trên FEAT-KNW-002.

## Tasks
1. Tạo các file port interfaces:
   - `internal/usecase/port/input.go` (ExtractUseCase, ResolveUseCase, EmbedUseCase, RerankUseCase)
   - `internal/usecase/port/output.go` (LLMClient, EmbedderClient, GraphReader, EventPublisher)

2. Tạo các DTO files:
   - `internal/usecase/dto/request.go`
   - `internal/usecase/dto/response.go`

3. Implement `internal/usecase/extract_entities.go`:
   - Build prompt từ template.
   - Call LLMClient.Complete().
   - Parse JSON, validate entities, track token usage.

4. Implement `internal/usecase/resolve_entities.go`:
   - Search similar entities (threshold 0.85).
   - Nếu giống: call LLM để so sánh và quyết định merge/create.
   - Nếu không giống: quyết định create.

5. Implement `internal/usecase/extract_edges.go`:
   - Gọi LLM để lấy temporal fact triples.

6. Implement `internal/usecase/resolve_edges.go`:
   - Tìm mâu thuẫn, invalidate old edges.

7. Implement `internal/usecase/generate_embedding.go`:
   - Text -> embedder -> vector (dimension).

8. Implement `internal/usecase/update_community.go`:
   - Label propagation, LLM summarization.

9. Implement `internal/usecase/rerank.go`:
   - Cross-encoder neural reranking.

10. Unit Tests:
    - Viết unit test cho từng usecase với mock LLM client và mock store reader.
    - Đảm bảo coverage >= 80%.
    - Đảm bảo usecases chỉ phụ thuộc vào port interfaces.
