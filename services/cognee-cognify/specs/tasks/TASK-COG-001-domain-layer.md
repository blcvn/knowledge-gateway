---
id: TASK-COG-001
title: Implement Domain Layer
feature: FEAT-COG-001
status: Done
---
# Task: Implement Domain Layer

## Objective
Implement the core domain entities, value objects, events, and errors for the `cognee-cognify` service.

## Files to Create/Modify
- `internal/domain/entity.go`
- `internal/domain/value_object.go`
- `internal/domain/event.go`
- `internal/domain/errors.go`

## Requirements
- `entity.go`: Define `CognifyJob`, `Chunk`, `Entity`, `Relationship`, `Community`, and `Ontology` structures.
- `value_object.go`: Define enums and value types: `JobStatus` (PENDING, RUNNING, COMPLETED, FAILED), `ChunkingStrategy`, `EntityType`, and `StageType`.
- `event.go`: Define domain events `PipelineCompletedEvent` and `StageAdvancedEvent`.
- `errors.go`: Define domain-specific errors like `JobNotFoundError`, `PipelineFailedError`, and `LLMTimeoutError`.
- **State Machine**: Implement methods on `CognifyJob` to manage its state transition (e.g., `AdvanceStage`, `Fail`, `Complete`).
- **Constraint**: Strict zero external dependencies in the domain layer.
