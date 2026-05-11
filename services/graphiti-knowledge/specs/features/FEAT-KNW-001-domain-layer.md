---
id: FEAT-KNW-001
title: Domain Layer — Extraction Types, Prompts, Embeddings
service: graphiti-knowledge
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement domain layer cho graphiti-knowledge: extraction/resolution entity types, prompt template definitions, embedding types, community types, reranking types, và domain errors.

## Scope

### In Scope
- `internal/domain/entity.go` — ExtractedEntity, ExtractedEdge, Resolution, DuplicateDecision
- `internal/domain/value_object.go` — PromptTemplate, TokenUsage, ModelConfig, EmbeddingDimension
- `internal/domain/embedding.go` — EmbeddingVector, EmbeddingRequest, EmbeddingResult
- `internal/domain/community.go` — CommunityNode, CommunityMember, CommunityLevel
- `internal/domain/rerank.go` — RerankRequest, RerankResult, CrossEncoderScore
- `internal/domain/errors.go` — ErrLLMTimeout, ErrPromptTooLong, ErrProviderUnavailable, ErrMalformedLLMResponse

### Key Types

```go
type ExtractedEntity struct {
    Name    string `json:"name"`
    Label   string `json:"label"`
    Summary string `json:"summary"`
}

type Resolution struct {
    ExistingEntityID string           `json:"existing_entity_id"`
    ExtractedEntity  ExtractedEntity  `json:"extracted_entity"`
    Decision         DuplicateDecision `json:"decision"`
    Confidence       float64          `json:"confidence"`
}

type DuplicateDecision string
const (
    DecisionMerge  DuplicateDecision = "merge"   // same entity
    DecisionCreate DuplicateDecision = "create"  // new entity
    DecisionSkip   DuplicateDecision = "skip"    // already exists
)
```

## Acceptance Criteria

- [ ] AC-1: Domain compiles with ZERO external imports
- [ ] AC-2: ExtractedEntity and ExtractedEdge have JSON tags
- [ ] AC-3: DuplicateDecision enum has exhaustive validation
- [ ] AC-4: TokenUsage tracks prompt_tokens + completion_tokens + model
- [ ] AC-5: EmbeddingVector validates dimension matches configured value

## Test Requirements
- **Unit tests**: Validation, type constructors
- **Minimum coverage**: 90%
