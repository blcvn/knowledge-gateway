---
id: FEAT-ING-001
title: Domain Layer — Episode + Saga Types
service: graphiti-ingestion
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement domain layer cho graphiti-ingestion standalone service: Episode, EpisodeType, Saga, SagaState, PipelineStep entities, value objects, domain events, và errors.

## Bối Cảnh Nghiệp Vụ

graphiti-ingestion là standalone saga orchestrator, coordinates pipeline across graphiti-knowledge (gRPC) và graphiti-store (gRPC). Domain types tương đồng với graphiti-pipeline/domain/ingestion/ nhưng focus vào gRPC delegation thay vì local calls.

## Scope

### In Scope
- `internal/domain/entity.go` — Episode, EpisodeType, Saga, SagaState, PipelineStep
- `internal/domain/value_object.go` — GroupID, EpisodeID, ContentHash
- `internal/domain/event.go` — EpisodeIngested, EpisodeFailed, SagaStepCompleted
- `internal/domain/errors.go` — ErrDuplicateEpisode, ErrPipelineFailed, ErrGroupLocked

### Out of Scope
- Knowledge domain types (graphiti-knowledge service owns those)

## Thiết Kế Kỹ Thuật

```go
type Episode struct {
    UUID          string            `json:"uuid"`
    Name          string            `json:"name"`
    GroupID       string            `json:"group_id"`
    Body          string            `json:"body"`
    Source        EpisodeType       `json:"source"`
    ReferenceTime time.Time         `json:"reference_time"`
    ContentHash   string            `json:"content_hash"`
    SagaID        *string           `json:"saga_id,omitempty"`
    EntityTypes   map[string]string `json:"entity_types,omitempty"`
    EdgeTypes     map[string]string `json:"edge_types,omitempty"`
    CreatedAt     time.Time         `json:"created_at"`
}

type SagaState struct {
    ID           string       `json:"id"`
    EpisodeID    string       `json:"episode_id"`
    GroupID      string       `json:"group_id"`
    CurrentStep  PipelineStep `json:"current_step"`
    Status       SagaStatus   `json:"status"`
    StepHistory  []StepEntry  `json:"step_history"`
    RetryCount   int          `json:"retry_count"`
    ErrorMessage string       `json:"error_message,omitempty"`
    StartedAt    time.Time    `json:"started_at"`
    CompletedAt  *time.Time   `json:"completed_at,omitempty"`
}
```

## Acceptance Criteria

- [ ] AC-1: Domain compiles with ZERO external imports
- [ ] AC-2: Episode.Validate() enforces required fields (name, body, group_id, reference_time)
- [ ] AC-3: ContentHash computed from SHA-256 of (name, group_id, reference_time)
- [ ] AC-4: SagaState.Transition() enforces valid state machine transitions
- [ ] AC-5: Domain events are immutable structs with timestamps

## Test Requirements
- **Unit tests**: Validation, hash computation, state transitions
- **Minimum coverage**: 90%
