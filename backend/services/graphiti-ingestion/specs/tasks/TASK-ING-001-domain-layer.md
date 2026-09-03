---
id: TASK-ING-001
title: Implement Domain Layer — Episode + Saga Types
service: graphiti-ingestion
type: task
status: done
priority: P0
created: 2026-05-11
dependencies: []
estimated_time: 4h
linked_feat: FEAT-ING-001
---

## Objective
Implement domain layer cho graphiti-ingestion standalone service: Episode, EpisodeType, Saga, SagaState, PipelineStep entities, value objects, domain events, và errors.

## Scope
### In Scope
- `internal/domain/entity.go` — Episode, EpisodeType, Saga, SagaState, PipelineStep
- `internal/domain/value_object.go` — GroupID, EpisodeID, ContentHash
- `internal/domain/event.go` — EpisodeIngested, EpisodeFailed, SagaStepCompleted
- `internal/domain/errors.go` — ErrDuplicateEpisode, ErrPipelineFailed, ErrGroupLocked

### Out of Scope
- Knowledge domain types (graphiti-knowledge service owns those)

## Technical Design Requirements
Required Structs:
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
- [x] Domain compiles with ZERO external imports
- [x] Episode.Validate() enforces required fields (name, body, group_id, reference_time)
- [x] ContentHash computed from SHA-256 of (name, group_id, reference_time)
- [x] SagaState.Transition() enforces valid state machine transitions
- [x] Domain events are immutable structs with timestamps

## Test Requirements
- Unit tests: Validation, hash computation, state transitions
- Minimum coverage: 90%
