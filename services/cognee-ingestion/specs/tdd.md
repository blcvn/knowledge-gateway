---
id: TDD-cognee-ingestion
title: Technical Design — cognee-ingestion
service: cognee-ingestion
version: 2.0.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Cognee
linked_sol: SOL-001
---

# Technical Design — cognee-ingestion

> **Group**: Cognee (Semantic KG) | **gRPC Port**: 9011 | **Health Port**: 9091 | **Origin**: Cognee L2-L4

## 1. Service Overview

Multi-modal data ingestion pipeline: file upload, text extraction, URL scraping, dataset management. Entry point for all data flowing into the Cognee semantic knowledge graph pipeline. Organizes data by dataset/namespace per tenant.

**Core Use Cases**:
- `IngestFile` — Upload + extract text from PDF/DOCX/PPTX/CSV via streaming gRPC
- `IngestText` — Direct text input with auto-classification
- `IngestUrl` — Web scraping + content extraction
- `ManageDataset` — CRUD dataset lifecycle per tenant

## 2. Clean Architecture Layers

| Layer | Path | Responsibility |
|-------|------|---------------|
| Domain | `internal/domain/` | Dataset, DataItem, DataSource, events, errors |
| Usecase | `internal/usecase/` | IngestFile, IngestText, IngestUrl, ManageDataset |
| Adapter | `internal/adapter/` | gRPC handler, NATS publisher, repos, extractors |
| Infra | `internal/infra/` | Config, server, Wire, telemetry |

## 3. gRPC API

```protobuf
service CogneeIngestionService {
  rpc CreateDataset(CreateDatasetRequest) returns (Dataset);
  rpc DeleteDataset(DeleteDatasetRequest) returns (google.protobuf.Empty);
  rpc ListDatasets(ListDatasetsRequest) returns (ListDatasetsResponse);
  rpc GetDatasetStatus(GetDatasetStatusRequest) returns (DatasetStatusResponse);
  rpc AddData(stream AddDataRequest) returns (AddDataResponse);
  rpc AddText(AddTextRequest) returns (AddTextResponse);
  rpc AddUrl(AddUrlRequest) returns (AddUrlResponse);
}
```

## 4. NATS Events

| Direction | Subject | Payload | Peer |
|-----------|---------|---------|------|
| **Publish** | `cognee.data.ingested` | `{dataset_id, tenant_id, item_ids[]}` | cognee-cognify |
| **Subscribe** | `admin.tenant.deleted` | `{tenant_id}` | vnp-admin |

## 5. Cross-Service Dependencies

| Target Service | Protocol | Purpose |
|---------------|----------|---------|
| cognee-cognify | NATS (async) | Trigger KG pipeline after ingestion |
| vnp-admin | NATS (subscribe) | Handle tenant deletion cascade |

## 6. Multi-Tenancy

Tenant isolation via gRPC metadata `x-tenant-id` → PostgreSQL RLS + MinIO namespace.

---

## Feature Specs Registry

| ID | Title | Status | Priority | Phase |
|----|-------|--------|----------|-------|
| [FEAT-ING-001](./features/FEAT-ING-001-domain-usecase-layer.md) | Domain + Usecase Layer | Ready | P0 | Phase 1 |
| [FEAT-ING-002](./features/FEAT-ING-002-adapter-layer.md) | Adapter Layer (gRPC + NATS + Repos) | Ready | P0 | Phase 2 |
| [FEAT-ING-003](./features/FEAT-ING-003-infra-wire.md) | Infrastructure + Wire DI | Ready | P0 | Phase 3 |

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

> **Linked**: [SOL-001](../../cognee-pipeline/specs/solutions/SOL-001-implement-cognee-pipeline-service.md) | [Architecture Spec](../../../services/cognee/specs/services/02-cognee-ingestion.md)
