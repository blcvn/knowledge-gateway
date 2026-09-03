---
id: ARCH-003
title: Merge graphiti-ingestion + graphiti-knowledge → graphiti-pipeline
service: graphiti-pipeline
version: 1.0.0
status: Ready
priority: P1
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
linked_adr: ADR-0001
behavior_change: false
---

## Vấn Đề Kiến Trúc Hiện Tại

`graphiti-ingestion` orchestrates a saga pipeline that makes 6 sequential gRPC calls to `graphiti-knowledge` (ExtractEntities → ResolveEntities → ExtractEdges → ResolveEdges → GenerateEmbedding → UpdateCommunity) and 1 call to `graphiti-store`. Cross-service gRPC for tightly sequential saga steps adds latency and failure modes.

## Kiến Trúc Mới

```
services/graphiti-pipeline/
├── internal/
│   ├── domain/
│   │   ├── ingestion/      # Episode, IngestionJob, SagaState
│   │   └── knowledge/      # EntityExtraction, EdgeExtraction, Community
│   ├── usecase/
│   │   ├── ingest/         # IngestEpisode, BulkIngest (saga orchestrator)
│   │   └── knowledge/      # ExtractEntities, ResolveEntities, ExtractEdges, etc.
│   ├── adapter/grpc/
│   │   ├── ingestion_handler.go    # GraphitiIngestionService
│   │   └── knowledge_handler.go    # GraphitiKnowledgeService
│   └── infra/
```

**Key**: Saga steps become local function calls. `graphiti-store` remains separate (it's the DB abstraction layer shared with graphiti-search).

## Acceptance Criteria

- [ ] AC-1: Saga pipeline executes locally (no cross-service gRPC for knowledge RPCs)
- [ ] AC-2: Compensating actions still functional for each saga step
- [ ] AC-3: `graphiti.episode.completed` emitted after full saga
- [ ] AC-4: graphiti-store gRPC calls preserved (separate service)
- [ ] AC-5: Per-group_id serialization maintained
