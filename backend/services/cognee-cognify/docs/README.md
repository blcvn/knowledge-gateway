---
id: DOC-S01
service: cognee-cognify
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
owner: VNP Memory — Cognee Team
---

# cognee-cognify

> **Group**: Cognee (Semantic KG) | **gRPC Port**: 9012 | **Health Port**: 9092 | **Origin**: Cognee

## Purpose

Knowledge graph construction pipeline service responsible for classify → chunk → extract entities → extract relationships → deduplicate → build graph → embed → summarize communities. This is the core LLM-intensive processing service of the Cognee domain.

**Business Capability**: Transforms raw ingested data into a structured semantic knowledge graph (Neo4j) with vector embeddings (Qdrant) for downstream search and retrieval.

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC (internal), NATS JetStream (async events)
- **Database**: Neo4j/SurrealDB (knowledge graph), Qdrant/pgvector (vector embeddings), PostgreSQL (job status, pipeline state)
- **AI Integration**: Bifrost LLM Gateway (entity extraction, deduplication, summarization)
- **Architecture**: 4-layer Clean Architecture (Domain → Usecase → Adapter → Infra)
- **DI**: Google Wire

## Quick Start

```bash
# From monorepo root
make build-cognee-cognify
make run-cognee-cognify

# Or with Docker
docker compose up cognee-cognify

# Verify health
curl http://localhost:9092/healthz
```

## Key Features

- **Content Classification**: Auto-detect content type for optimal chunking strategy
- **Multi-Strategy Chunking**: Recursive, AST-based, paragraph, and sentence-level chunking
- **Entity Extraction**: NER + relationship extraction via LLM (GPT-4o)
- **Entity Deduplication**: Semantic entity resolution to prevent graph bloat
- **Knowledge Graph Construction**: Build nodes + edges in Neo4j with tenant isolation
- **Embedding Generation**: Vector embeddings for entities/chunks in Qdrant
- **Community Summarization**: Auto-generate summaries for graph communities
- **Pipeline State Machine**: Track job progress (PENDING → RUNNING → COMPLETED/FAILED)

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
