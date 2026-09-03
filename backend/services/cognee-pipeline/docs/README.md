---
id: DOC-S01
service: cognee-pipeline
version: 1.0.0
status: Active
created: 2026-05-10
updated: 2026-05-10
owner: VNP Memory — Cognee Team
---

# cognee-pipeline

> **Group**: Cognee (Semantic KG) | **Ports**: 9011(gRPC) 9091(Health) | **Origin**: Consolidated

## Purpose

Unified ingestion and knowledge graph construction service combining `cognee-ingestion` and `cognee-cognify` into a single binary. Handles document/text/URL ingestion, dataset management, and multi-stage cognify pipeline (classify → chunk → extract → deduplicate → build graph → embed → summarize communities).

### Business Capability

- **Data Ingestion**: File (PDF/DOCX/PPTX/CSV), text, URL ingestion with auto format detection
- **Dataset Management**: CRUD lifecycle with tenant isolation
- **7-Stage Cognify Pipeline**: Entity extraction, relationship extraction, deduplication, community detection, embedding generation
- **Local Pipeline Trigger**: Ingestion → cognify via local function call (no inter-service NATS hop)

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.23+ |
| Framework | gRPC (dual service: CogneeIngestionService + CogneeCognifyService) |
| Database | PostgreSQL 17 + pgvector |
| Graph DB | Neo4j 5+ |
| Object Storage | MinIO/S3 |
| Async Events | NATS JetStream |
| LLM | Bifrost multi-provider gateway |
| Architecture | 4-layer Clean Architecture |
| DI | Google Wire |

## Quick Start

```bash
# From monorepo root
make build-cognee-pipeline
make run-cognee-pipeline

# Or with Docker
docker compose up cognee-pipeline

# Verify health
curl http://localhost:9091/healthz

# Or directly
cd services/cognee-pipeline
go run cmd/server/main.go
```

## Key Features

- **Dual gRPC Services**: Single binary exposes both CogneeIngestionService and CogneeCognifyService
- **Streaming Upload**: gRPC streaming for large file uploads
- **Text Extraction**: PDF, DOCX, PPTX, CSV, HTML auto-detection
- **URL Scraping**: Web content extraction with sanitization
- **LLM Entity Extraction**: GPT-4o based NER + relationship extraction
- **Entity Deduplication**: Semantic entity resolution to prevent graph bloat
- **Community Detection**: Louvain algorithm via Neo4j GDS
- **Pipeline State Machine**: Job progress tracking with resume on failure
- **Tenant Isolation**: PostgreSQL RLS + Neo4j namespace + MinIO path

## Links

- [API Reference](./api.md)
- [Architecture](./architecture.md)
- [Data Model](./data-model.md)
- [Configuration](./configuration.md)
- [Runbook](./runbook.md)
- [Changelog](./changelog.md)
- [Specs](../specs/)

## Owner

- **Team**: VNP Memory — Cognee Engine
- **Contact**: TBD
