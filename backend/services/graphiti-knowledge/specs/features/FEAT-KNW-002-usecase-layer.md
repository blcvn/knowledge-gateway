---
id: FEAT-KNW-002
title: Usecase Layer — Extract, Resolve, Embed, Community
service: graphiti-knowledge
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement usecase layer cho graphiti-knowledge: 7 core usecases (extract entities, resolve entities, extract edges, resolve edges, generate embedding, update community, rerank) và port interfaces.

## Scope

### In Scope
- `internal/usecase/extract_entities.go` — Content → LLM prompt → parse → validate entities
- `internal/usecase/resolve_entities.go` — Search similar → LLM compare → merge/create decision
- `internal/usecase/extract_edges.go` — Episode + entities → LLM → temporal fact triples
- `internal/usecase/resolve_edges.go` — Find contradictions → invalidate old edges
- `internal/usecase/generate_embedding.go` — Text → embedder → vector
- `internal/usecase/update_community.go` — Label propagation → LLM summarization
- `internal/usecase/rerank.go` — Cross-encoder neural reranking
- Port interfaces: LLMClient, EmbedderClient, GraphReader (store gRPC read-only), EventPublisher

### Entity Extraction Flow

```go
func (uc *ExtractEntitiesUseCase) Execute(ctx context.Context, req ExtractEntitiesRequest) ([]ExtractedEntity, TokenUsage, error) {
    // 1. Build prompt from template (inject content, previous episodes, entity types)
    // 2. Call LLMClient.Complete() via Bifrost
    // 3. Parse JSON from LLM response (handle markdown fences)
    // 4. Validate extracted entities (non-empty name, valid label)
    // 5. Track token usage
    // 6. Return validated entities + usage
}
```

### Entity Resolution Flow

```go
func (uc *ResolveEntitiesUseCase) Execute(ctx context.Context, req ResolveEntitiesRequest) ([]Resolution, error) {
    // For each extracted entity:
    // 1. Generate name embedding
    // 2. Search similar entities in graphiti-store (cosine, threshold 0.85)
    // 3. If similar found: call LLM to compare → merge/create decision
    // 4. If no similar: create new entity
    // 5. Return resolution decisions
}
```

## Acceptance Criteria

- [ ] AC-1: ExtractEntities returns parsed entities from LLM with >95% parse success rate
- [ ] AC-2: ResolveEntities merges duplicates with configurable similarity threshold (default 0.85)
- [ ] AC-3: ExtractEdges produces bi-temporal fact triples with valid_at from episode context
- [ ] AC-4: ResolveEdges detects contradictions and invalidates superseded edges
- [ ] AC-5: GenerateEmbedding produces vectors of configured dimension
- [ ] AC-6: UpdateCommunity runs label propagation then LLM summarization for each cluster
- [ ] AC-7: All usecases depend only on port interfaces

## Test Requirements
- **Unit tests**: Each usecase with mock LLM client + mock store reader
- **Minimum coverage**: 80%
