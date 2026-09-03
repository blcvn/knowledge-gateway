---
id: TASK-DOC-002
title: Usecases & Orchestration
service: sm-document
status: Done
priority: P0
created: 2026-05-11
---

# Usecases & Orchestration

## Objective
Implement the business logic orchestration and usecases.

## Specs Mapping
Please refer to the following content from `specs/tdd.md` to implement this task:

```markdown
---
id: TDD-sm-document
title: Technical Design — sm-document
service: sm-document
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Supermemory
---

# Technical Design — sm-document

> **Group**: Supermemory | **gRPC Port**: 9071 | **Health Port**: 12071

> **🚨 DEPRECATION NOTICE**: This specification is obsolete. The service has been merged into `sm-engine` (Ref: [ARCH-007-merge-sm-engine]).


## 1. Service Overview

Document CRUD, chunking, and ingestion pipeline. Supports PDF, HTML, text, and image content extraction with format-aware chunking strategies.

## 2. gRPC API

RPCs: CreateDocument, GetDocument, DeleteDocument, ListDocuments, GetChunks

```protobuf
service SmDocumentService {
  rpc CreateDocument(CreateDocumentRequest) returns (Document);
  rpc GetDocument(GetDocumentRequest) returns (Document);
  rpc DeleteDocument(DeleteDocumentRequest) returns (Empty);
  rpc ListDocuments(ListDocumentsRequest) returns (ListDocumentsResponse);
  rpc GetChunks(GetChunksRequest) returns (ChunksResponse);
}
```

## 3. NATS Events

Published: `sm.document.created` → sm-memory (extract facts), sm-search (index).
Published: `sm.document.deleted` → sm-memory (cleanup), sm-search (deindex).

## 4. Storage

PostgreSQL/SurrealDB (documents, chunks), pgvector (chunk embeddings)

## 5. Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL/SurrealDB | SQL | Document + chunk persistence |
| NATS JetStream | Publisher | `sm.document.created/deleted` |

## 6. Observability

- **Metrics**: Prometheus counters/histograms for all RPCs
- **Traces**: OTel spans for every usecase method
- **Logs**: Structured JSON via slog with request_id, tenant_id, org_id
- **Health**: gRPC health check + HTTP /healthz on port 12071

## 7. Multi-Tenancy

Organization isolation via `org_id`. Propagated via gRPC metadata.

---

> **Next Steps**: Decompose into FEAT specs in `specs/features/`.

```

## Acceptance Criteria
- [x] Usecases implemented fulfilling the service overview.
- [x] Usecases correctly coordinate domain logic and ports.
