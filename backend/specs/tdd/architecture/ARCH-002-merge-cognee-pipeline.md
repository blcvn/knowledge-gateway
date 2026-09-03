---
id: ARCH-002
title: Merge cognee-ingestion + cognee-cognify → cognee-pipeline
service: cognee-pipeline
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

`cognee-ingestion` và `cognee-cognify` luôn hoạt động sequential: ingestion hoàn thành → emit NATS `cognee.data.ingested` → cognify subscribes và bắt đầu pipeline. Cả hai cùng depend on PostgreSQL, Neo4j, Qdrant. NATS hop giữa chúng là unnecessary latency.

## Kiến Trúc Mới Đề Xuất

```
services/cognee-pipeline/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── ingestion/      # Dataset, DataItem, DataSource
│   │   └── cognify/        # CognifyJob, PipelineStage, Ontology
│   ├── usecase/
│   │   ├── ingest/         # IngestFile, IngestText, IngestUrl, ManageDataset
│   │   └── cognify/        # TriggerCognify (calls ingest locally), pipeline stages
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── ingestion_handler.go   # CogneeIngestionService
│   │   │   └── cognify_handler.go     # CogneeCognifyService
│   │   ├── repository/
│   │   │   ├── postgres/   # Dataset, DataItem, Job tables
│   │   │   ├── neo4j/      # Knowledge graph CRUD
│   │   │   └── qdrant/     # Entity/chunk embeddings (→ pgvector in Phase 4)
│   │   └── event/nats/     # Emit: cognee.pipeline.completed
│   └── infra/
├── docs/                   # DOC-S01 through DOC-S07
└── specs/
```

**Key change**: `cognee.data.ingested` NATS event becomes local function call. Only `cognee.pipeline.completed` is emitted externally (for cognee-search reindex).

## Phạm Vi Refactor

### Files cần tạo mới
- `services/cognee-pipeline/` — merged service

### Files cần xóa (sau migration)
- `services/cognee-ingestion/`
- `services/cognee-cognify/`

## Invariants

- [ ] CogneeIngestionService proto methods giữ nguyên
- [ ] CogneeCognifyService proto methods giữ nguyên
- [ ] cognee-search vẫn subscribe `cognee.pipeline.completed`
- [ ] Neo4j knowledge graph schema giữ nguyên

## Acceptance Criteria

- [ ] AC-1: `cognee-pipeline` registers both CogneeIngestionService + CogneeCognifyService
- [ ] AC-2: IngestFile → automatic cognify trigger (local call, no NATS)
- [ ] AC-3: `cognee.pipeline.completed` emitted after cognify completes
- [ ] AC-4: cognee-search receives and processes pipeline.completed events
