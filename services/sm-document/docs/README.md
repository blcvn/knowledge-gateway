---
id: DOC-S01
service: sm-document
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Supermemory Team
---

# sm-document

> **Group**: Supermemory (Adaptive KG Memory) | **gRPC Port**: 9071 | **Health Port**: 12071 | **Origin**: Supermemory

## Purpose

Document CRUD, chunking, and ingestion pipeline. Supports PDF, HTML, text, and image content extraction with format-aware chunking strategies.

### Business Capability

- **Document CRUD**: Create/read/delete documents with full content extraction
- **Chunking Pipeline**: Format-aware chunking (paragraph, section, page-based)
- **Content Extraction**: PDF, HTML, text, image content parsing
- **Chunk Retrieval**: Get document chunks for downstream search indexing

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server + NATS JetStream
- **Database**: PostgreSQL/SurrealDB (documents, chunks), pgvector (chunk embeddings)
- **Architecture**: 4-layer Clean Architecture (Domain → Usecase → Adapter → Infra)
- **DI**: Google Wire

## Quick Start

```bash
make build-sm-document
make run-sm-document
docker compose up sm-document
```

## API Surface

### gRPC Service

```protobuf
service SmDocumentService {
  rpc CreateDocument(CreateDocumentRequest) returns (Document);
  rpc GetDocument(GetDocumentRequest) returns (Document);
  rpc DeleteDocument(DeleteDocumentRequest) returns (Empty);
  rpc ListDocuments(ListDocumentsRequest) returns (ListDocumentsResponse);
  rpc GetChunks(GetChunksRequest) returns (ChunksResponse);
}
```

## NATS Events

Published: `sm.document.created` → sm-memory (extract facts), sm-search (index).
Published: `sm.document.deleted` → sm-memory (cleanup), sm-search (deindex).

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL/SurrealDB | SQL | Document + chunk persistence |
| NATS JetStream | Publisher | `sm.document.created/deleted` |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)
- [Architecture Spec](../../../specs/architecture/07-supermemory-services.md)

## Owner

- **Team**: VNP Memory — Supermemory
