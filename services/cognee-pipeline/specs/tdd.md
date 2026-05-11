---
id: TDD-cognee-pipeline
title: Technical Design — cognee-pipeline
service: cognee-pipeline
version: 2.0.0
status: Ready
created: 2026-05-10
updated: 2026-05-10
group: Cognee
linked_sol: SOL-001
---

# Technical Design — cognee-pipeline

> **Group**: Cognee (Semantic KG) | **gRPC Port**: 9011 | **Health Port**: 9091 | **Origin**: Consolidated (Ingestion + Cognify)

## 1. Service Overview

Consolidated service merging cognee-ingestion and cognee-cognify into a single binary. Exposes both CogneeIngestionService and CogneeCognifyService gRPC APIs from one process. Internal ingestion → cognify trigger uses local function call instead of NATS event.

**Key Difference**: In standalone deployment, ingestion publishes `cognee.data.ingested` → cognify subscribes. In pipeline deployment, ingestion triggers cognify directly via local call.

## 2. Dual gRPC Services

```protobuf
// Same port, dual service registration
service CogneeIngestionService {
  rpc CreateDataset(CreateDatasetRequest) returns (Dataset);
  rpc DeleteDataset(DeleteDatasetRequest) returns (google.protobuf.Empty);
  rpc ListDatasets(ListDatasetsRequest) returns (ListDatasetsResponse);
  rpc GetDatasetStatus(GetDatasetStatusRequest) returns (DatasetStatusResponse);
  rpc AddData(stream AddDataRequest) returns (AddDataResponse);
  rpc AddText(AddTextRequest) returns (AddTextResponse);
  rpc AddUrl(AddUrlRequest) returns (AddUrlResponse);
}

service CogneeCognifyService {
  rpc TriggerCognify(TriggerCognifyRequest) returns (CognifyJob);
  rpc GetJobStatus(GetJobStatusRequest) returns (CognifyJob);
  rpc CancelJob(CancelJobRequest) returns (google.protobuf.Empty);
}
```

## 3. Architecture

```
cognee-pipeline (single binary)
├── CogneeIngestionService → ingest usecases → local trigger
├── CogneeCognifyService → cognify usecases → 8-stage pipeline
├── Shared DB pools (PostgreSQL, Neo4j, pgvector)
└── NATS publisher: cognee.pipeline.completed → cognee-search
```

## 4. Cross-Service Dependencies

| Target | Protocol | Purpose |
|--------|----------|---------|
| cognee-search | NATS (async) | Notify graph/vector index update |

---

## Feature Specs Registry

| ID | Title | Status | Priority | Phase |
|----|-------|--------|----------|-------|
| [FEAT-PIP-001](./features/FEAT-PIP-001-consolidate-ingestion-cognify.md) | Consolidate Ingestion + Cognify | Ready | P1 | Phase 4 |

## Architecture Specs Registry

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| — | _To be populated_ | — | — |

## Technical Specs Registry

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| — | _To be populated_ | — | — |

---

> **Linked**: [SOL-001](./solutions/SOL-001-implement-cognee-pipeline-service.md)
