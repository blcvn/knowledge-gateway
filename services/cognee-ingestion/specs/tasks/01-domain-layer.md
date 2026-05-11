---
id: TASK-ING-01
title: Implement Domain Layer
service: cognee-ingestion
feature: FEAT-ING-001
status: Done
---

## Objective
Implement Domain entities, value objects, events, and errors for the cognee-ingestion service.

## Files to Create/Update
- `internal/domain/entity.go`: Define `Dataset` and `DataItem` structs.
- `internal/domain/value_object.go`: Define `DatasetStatus`, `MimeType`, `DataSource`.
- `internal/domain/event.go`: Define `DataIngestedEvent`.
- `internal/domain/errors.go`: Define domain-specific errors (`DatasetNotFoundError`, `DuplicateDatasetError`, `ExtractionFailedError`).
- `internal/domain/entity_test.go`: Unit tests for domain logic.

## Acceptance Criteria
- All domain structures exactly match the specifications in FEAT-ING-001.
- Unit tests cover entity validation and value object behavior with >= 80% coverage.
- ZERO external imports (no gRPC, DB, or framework) in the domain layer.
