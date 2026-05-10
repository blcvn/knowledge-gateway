# cognee-pipeline — Semantic KG Ingestion + Cognify Service

> **Service**: `cognee-pipeline` | **gRPC Port**: 9011 | **Health**: 9091  
> **Origin**: Consolidated from cognee-ingestion + cognee-cognify  
> **Status**: Proposed | **Version**: 0.1.0

## Purpose

Unified ingestion and knowledge graph construction service for the Cognee semantic KG engine. Handles document/text/URL ingestion, dataset management, and multi-stage cognify pipeline (entity extraction, relationship extraction, deduplication, community detection, embedding generation).

## Business Capability

- **Data Ingestion**: File, text, URL ingestion with automatic format detection
- **Dataset Management**: Create, list, delete datasets with data items
- **Cognify Pipeline**: 7-stage pipeline for KG construction (extract entities → resolve → extract edges → resolve → generate embeddings → update communities → post-process)
- **15 Retrieval Strategies**: Base support for cognee-search retrieval

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.23+ |
| RPC | gRPC (2 service definitions: CogneeIngestionService + CogneeCognifyService) |
| Database | PostgreSQL 17 + pgvector |
| Graph DB | Neo4j 5+ |
| Object Storage | MinIO/S3 |
| Async | NATS JetStream |
| LLM | Bifrost multi-provider |

## Quick Start

```bash
cd services/cognee-pipeline
go run cmd/server/main.go
# gRPC: :9011 | Health: :9091
```

## Links

- [Architecture](./architecture.md)
- [Changelog](./changelog.md)

## Owner

Cognee Engine Team
