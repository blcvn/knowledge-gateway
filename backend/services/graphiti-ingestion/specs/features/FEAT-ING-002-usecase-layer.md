---
id: FEAT-ING-002
title: Usecase Layer — Saga Orchestrator with gRPC Delegation
service: graphiti-ingestion
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement saga orchestrator usecase cho graphiti-ingestion. Khác với graphiti-pipeline (local calls), ingestion standalone delegates tất cả extraction/resolution steps qua gRPC tới graphiti-knowledge service.

## Scope

### In Scope
- `internal/usecase/ingest_episode.go` — Single episode saga, gRPC delegation
- `internal/usecase/bulk_ingest.go` — Batch ingestion with dedup
- `internal/usecase/get_status.go` — Episode status query
- `internal/usecase/saga_orchestrator.go` — State machine + compensations
- Port interfaces: KnowledgeClient (gRPC), StoreClient (gRPC), SagaStateRepo, EventPublisher

### Key Difference from Pipeline

```
graphiti-pipeline: saga step → LOCAL function call → knowledge usecase
graphiti-ingestion: saga step → gRPC CALL → graphiti-knowledge:9023
```

## Acceptance Criteria

- [ ] AC-1: IngestEpisode delegates ExtractEntities to graphiti-knowledge via gRPC
- [ ] AC-2: Per-group serialization prevents concurrent saga execution
- [ ] AC-3: Compensation calls RollbackBulk on graphiti-store on failure
- [ ] AC-4: Circuit breaker protects both knowledge and store gRPC calls
- [ ] AC-5: Same IngestEpisodeRequest/Response as graphiti-pipeline (proto compat)

## Test Requirements
- **Unit tests**: Saga orchestrator with mock gRPC clients
- **Minimum coverage**: 80%
