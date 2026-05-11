---
id: FEAT-PIP-001
title: Domain Layer — Ingestion + Knowledge Entities
service: graphiti-pipeline
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement domain layer cho graphiti-pipeline với 2 sub-domains: `ingestion` (episode lifecycle, saga state) và `knowledge` (extracted entities, edges, embeddings, communities). Layer này là foundation — ZERO external imports, chỉ chứa pure Go types và business rules.

## Bối Cảnh Nghiệp Vụ

Domain layer định nghĩa ngôn ngữ chung (ubiquitous language) cho episodic knowledge graph:
- **Episode**: A unit of information (message, JSON, text) ingested into the graph
- **Entity**: A named concept extracted from episodes (person, place, concept)
- **Edge**: A temporal fact triple connecting two entities with validity windows
- **Community**: A cluster of related entities with LLM-generated summary

## Scope

### In Scope
- `internal/domain/ingestion/`: Episode, EpisodeType, Saga, SagaState, PipelineStep
- `internal/domain/ingestion/`: GroupID, EpisodeID, ContentHash value objects
- `internal/domain/ingestion/`: EpisodeIngested, EpisodeFailed domain events
- `internal/domain/ingestion/`: Domain errors (ErrDuplicateEpisode, ErrPipelineFailed, etc.)
- `internal/domain/knowledge/`: ExtractedEntity, ExtractedEdge, Resolution, DuplicateDecision
- `internal/domain/knowledge/`: PromptTemplate, TokenUsage, ModelConfig, EmbeddingVector
- `internal/domain/knowledge/`: CommunityNode, CommunityEdge, CommunityLevel
- `internal/domain/knowledge/`: Domain errors (ErrLLMTimeout, ErrPromptTooLong, etc.)

### Out of Scope
- Port interfaces (FEAT-PIP-002)
- Adapter implementations (FEAT-PIP-003..007)
- Infrastructure (FEAT-PIP-008)

## Thiết Kế Kỹ Thuật

### Business Logic

**Episode Types:**
```go
type EpisodeType string
const (
    EpisodeTypeMessage    EpisodeType = "message"
    EpisodeTypeJSON       EpisodeType = "json"
    EpisodeTypeText       EpisodeType = "text"
    EpisodeTypeFactTriple EpisodeType = "fact_triple"
)
```

**Saga Pipeline Steps:**
```go
type PipelineStep string
const (
    StepExtractEntities    PipelineStep = "EXTRACT_ENTITIES"
    StepResolveEntities    PipelineStep = "RESOLVE_ENTITIES"
    StepExtractEdges       PipelineStep = "EXTRACT_EDGES"
    StepResolveEdges       PipelineStep = "RESOLVE_EDGES"
    StepGenerateEmbeddings PipelineStep = "GENERATE_EMBEDDINGS"
    StepSaveBulk           PipelineStep = "SAVE_BULK"
    StepUpdateCommunity    PipelineStep = "UPDATE_COMMUNITY"
)
```

**Bi-temporal Validation Rules:**
- `valid_at` is REQUIRED and must be before `invalid_at` (if set)
- `invalid_at` is optional — NULL means "still valid"
- `expired_at` is set by edge resolution when a newer edge supersedes
- `created_at` is auto-set by the system

### Internal Architecture

Files to create:
1. `internal/domain/ingestion/entity.go` — Episode, Saga, SagaState structs
2. `internal/domain/ingestion/value_object.go` — GroupID, EpisodeID, ContentHash types
3. `internal/domain/ingestion/event.go` — Domain event structs
4. `internal/domain/ingestion/errors.go` — Sentinel errors
5. `internal/domain/knowledge/entity.go` — ExtractedEntity, ExtractedEdge, Resolution
6. `internal/domain/knowledge/value_object.go` — PromptTemplate, TokenUsage, ModelConfig
7. `internal/domain/knowledge/embedding.go` — EmbeddingVector, EmbeddingRequest
8. `internal/domain/knowledge/community.go` — CommunityNode, CommunityEdge
9. `internal/domain/knowledge/errors.go` — LLM-related errors

## Acceptance Criteria

- [ ] AC-1: `internal/domain/` compiles with ZERO external imports (only Go stdlib)
- [ ] AC-2: All entity types have proper JSON tags for serialization
- [ ] AC-3: Value objects implement `String()` and `Validate()` methods
- [ ] AC-4: Domain errors are sentinel errors with `errors.Is()` support
- [ ] AC-5: Bi-temporal validation rule enforced in `EntityEdge.Validate()`
- [ ] AC-6: `EpisodeType` enum has exhaustive switch coverage

## Test Requirements

- **Unit tests**: Validation methods, value object constructors, edge bi-temporal rules
- **Minimum coverage**: 90% (domain layer is critical)

## Definition of Done

- [ ] Code compiles with no external imports in domain/
- [ ] Unit tests pass, coverage ≥ 90%
- [ ] Linter clean (go vet, golangci-lint)
- [ ] No hardcoded strings for enum values
