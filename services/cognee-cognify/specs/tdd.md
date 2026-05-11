---
id: TDD-cognee-cognify
title: Technical Design — cognee-cognify
service: cognee-cognify
version: 2.0.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Cognee
linked_sol: SOL-001
---

# Technical Design — cognee-cognify

> **Group**: Cognee (Semantic KG) | **gRPC Port**: 9012 | **Health Port**: 9092 | **Origin**: Cognee L3-L5

## 1. Service Overview

Knowledge graph construction pipeline: classify → chunk → extract entities → extract relationships → deduplicate → build graph → embed → summarize communities. Core LLM-intensive processing service.

## 2. Clean Architecture Layers

| Layer | Path | Responsibility |
|-------|------|---------------|
| Domain | `internal/domain/` | CognifyJob, Entity, Relationship, Community, events |
| Usecase | `internal/usecase/` | 8-stage pipeline orchestrator + individual stages |
| Adapter | `internal/adapter/` | gRPC handler, NATS sub/pub, Neo4j, Qdrant, Bifrost |
| Infra | `internal/infra/` | Config, server, Wire, telemetry, bulkhead |

## 3. Pipeline Stages

| Stage | Function | LLM | Output |
|-------|----------|-----|--------|
| classify | Content type detection | GPT-4o-mini | ChunkingStrategy |
| chunk | Text segmentation | No | []Chunk |
| extract_entities | NER | GPT-4o | []Entity |
| extract_relationships | Relation extraction | GPT-4o | []Relationship |
| deduplicate | Entity resolution | GPT-4o-mini | Merged entities |
| build_graph | Neo4j write | No | Graph nodes+edges |
| embed | Vector generation | Embedding model | Qdrant vectors |
| summarize | Community summaries | GPT-4o-mini | Community nodes |

## 4. gRPC API

```protobuf
service CogneeCognifyService {
  rpc TriggerCognify(TriggerCognifyRequest) returns (CognifyJob);
  rpc GetJobStatus(GetJobStatusRequest) returns (CognifyJob);
  rpc CancelJob(CancelJobRequest) returns (google.protobuf.Empty);
}
```

## 5. NATS Events

| Direction | Subject | Peer |
|-----------|---------|------|
| Subscribe | `cognee.data.ingested` | cognee-ingestion |
| Publish | `cognee.pipeline.completed` | cognee-search |

## 6. Cross-Service Dependencies

| Target | Protocol | Purpose |
|--------|----------|---------|
| cognee-ingestion | gRPC | Fetch raw data items |
| cognee-search | NATS (async) | Notify search index update |

## 7. Multi-Tenancy

Tenant isolation via Neo4j namespace labels + Qdrant tenant_id filter.

---

## Feature Specs Registry

| ID | Title | Status | Priority | Phase |
|----|-------|--------|----------|-------|
| [FEAT-COG-001](./features/FEAT-COG-001-domain-usecase-layer.md) | Domain + Usecase (8-Stage Pipeline) | Ready | P0 | Phase 1 |
| [FEAT-COG-002](./features/FEAT-COG-002-adapter-layer.md) | Adapter Layer (gRPC + NATS + Neo4j + Qdrant) | Ready | P0 | Phase 2 |
| [FEAT-COG-003](./features/FEAT-COG-003-infra-wire.md) | Infrastructure + Wire DI | Ready | P0 | Phase 3 |

## Architecture Specs Registry

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| — | _To be populated_ | — | — |

## Technical Specs Registry

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| — | _To be populated_ | — | — |

## Quality Specs Registry

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| — | _To be populated_ | — | — |

---

> **Linked**: [SOL-001](../../cognee-pipeline/specs/solutions/SOL-001-implement-cognee-pipeline-service.md) | [Architecture Spec](../../../services/cognee/specs/services/03-cognee-cognify.md)
