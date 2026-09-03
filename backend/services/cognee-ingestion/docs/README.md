---
id: DOC-S01
service: cognee-ingestion
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
owner: VNP Memory — Cognee Team
---

# cognee-ingestion

> **Group**: Cognee (Semantic KG) | **gRPC Port**: 9011 | **Health Port**: 9091 | **Origin**: Cognee

## Purpose

Multi-modal data ingestion pipeline service responsible for file upload, text extraction, URL scraping, and dataset management. Organizes data by dataset/namespace per tenant. This service is the entry point for all data flowing into the Cognee semantic knowledge graph pipeline.

**Business Capability**: Data acquisition, classification, extraction, and normalization for downstream knowledge graph construction (cognee-cognify) and semantic search (cognee-search).

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC (internal), NATS JetStream (async events)
- **Database**: PostgreSQL (dataset metadata, data items), MinIO/S3 (raw file storage), Redis (upload progress cache)
- **AI Integration**: Bifrost LLM Gateway (content classification)
- **Architecture**: 4-layer Clean Architecture (Domain → Usecase → Adapter → Infra)
- **DI**: Google Wire (compile-time dependency injection)
- **Observability**: OTel + Prometheus + slog JSON

## Quick Start

```bash
# From monorepo root
make build-cognee-ingestion
make run-cognee-ingestion

# Or with Docker
docker compose up cognee-ingestion

# Verify health
curl http://localhost:9091/healthz
```

## Key Features

- **File Upload**: PDF, DOCX, PPTX, CSV, TXT, Markdown extraction via streaming gRPC
- **Text Ingestion**: Direct text input with auto-classification
- **URL Scraping**: Web content extraction and normalization
- **Dataset Management**: CRUD operations per tenant with status tracking
- **Multi-Tenant Isolation**: PostgreSQL RLS + tenant_id scoping

## Links

- [API Reference](./api.md)
- [Architecture](./architecture.md)
- [Data Model](./data-model.md)
- [Configuration](./configuration.md)
- [Runbook](./runbook.md)
- [Changelog](./changelog.md)
- [Specs](../specs/)
- [Architecture Spec](../../../specs/architecture/02-cognee-services.md)

## Owner

- **Team**: VNP Memory — Cognee
- **Contact**: TBD
