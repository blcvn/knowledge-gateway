---
id: TASK-ING-002
title: Implement Usecase Layer — Saga Orchestrator with gRPC Delegation
service: graphiti-ingestion
type: task
status: done
priority: P0
created: 2026-05-11
dependencies: [TASK-ING-001]
estimated_time: 8h
linked_feat: FEAT-ING-002
---

## Objective
Implement saga orchestrator usecase cho graphiti-ingestion. Khác với graphiti-pipeline (local calls), ingestion standalone delegates tất cả extraction/resolution steps qua gRPC tới graphiti-knowledge service.

## Scope
- `internal/usecase/ingest_episode.go` — Single episode saga, gRPC delegation
- `internal/usecase/bulk_ingest.go` — Batch ingestion with dedup
- `internal/usecase/get_status.go` — Episode status query
- `internal/usecase/saga_orchestrator.go` — State machine + compensations
- Port interfaces: KnowledgeClient (gRPC), StoreClient (gRPC), SagaStateRepo, EventPublisher

## Key Difference from Pipeline
```
graphiti-pipeline: saga step → LOCAL function call → knowledge usecase
graphiti-ingestion: saga step → gRPC CALL → graphiti-knowledge:9023
```

## Acceptance Criteria
- [x] IngestEpisode delegates ExtractEntities to graphiti-knowledge via gRPC
- [x] Per-group serialization prevents concurrent saga execution
- [x] Compensation calls RollbackBulk on graphiti-store on failure
- [x] Circuit breaker protects both knowledge and store gRPC calls
- [x] Same IngestEpisodeRequest/Response as graphiti-pipeline (proto compat)

## Test Requirements
- Unit tests: Saga orchestrator with mock gRPC clients
- Minimum coverage: 80%
