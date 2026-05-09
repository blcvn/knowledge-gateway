---
id: TDD-cognee-cognify
title: Technical Design — cognee-cognify
service: cognee-cognify
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
group: Cognee
---

# Technical Design — cognee-cognify

> **Group**: Cognee (Semantic KG) | **gRPC Port**: 9012 | **Origin**: Cognee L3-L5

## 1. Service Overview

Knowledge graph construction pipeline: classify → chunk → extract entities → extract relationships → deduplicate → build graph → embed → summarize communities. Core LLM-intensive processing service.

## 2. Clean Architecture Layers

### 2.1 Domain Layer

```go
type CognifyJob struct {
    ID, DatasetID, TenantID  string
    Status                    JobStatus  // PENDING/RUNNING/COMPLETED/FAILED/CANCELLED
    CurrentStage              string
    ProgressPercent           float64
    Metrics                   PipelineMetrics
}
```

### 2.2 Usecase Layer — Pipeline Stages

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

### 2.3 Adapter Layer

- gRPC handler: TriggerCognify, GetJobStatus, CancelJob
- NATS subscriber: cognee.data.ingested
- NATS publisher: cognee.pipeline.completed

### 2.4 Infrastructure Layer

- PostgreSQL: job state persistence
- Neo4j: graph operations
- Qdrant: vector storage
- Bifrost: LLM + embedding calls

## 3. gRPC API

```protobuf
service CogneeCognifyService {
  rpc TriggerCognify(TriggerCognifyRequest) returns (CognifyJob);
  rpc GetJobStatus(GetJobStatusRequest) returns (CognifyJob);
  rpc CancelJob(CancelJobRequest) returns (Empty);
}
```

## 4. NATS Events

| Direction | Subject | Peer |
|-----------|---------|------|
| Subscribe | `cognee.data.ingested` | cognee-ingestion |
| Publish | `cognee.pipeline.completed` | cognee-search |

## 5. Cross-Service Dependencies

| Target | Protocol | Purpose |
|--------|----------|---------|
| cognee-ingestion | gRPC | Fetch raw data items for processing |
| cognee-search | NATS (async) | Notify search index update |

## 6. Observability

- Metrics: Pipeline duration, stage duration, LLM call counts
- Traces: OTel spans per pipeline stage
- Logs: slog JSON with job_id, dataset_id, stage

## 7. Multi-Tenancy

Tenant isolation via Neo4j namespace labels + Qdrant tenant_id filter.

---

> **Next Steps**: Decompose into FEAT specs in `specs/features/`.
