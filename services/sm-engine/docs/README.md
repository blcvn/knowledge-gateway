# sm-engine — Supermemory Adaptive KG Engine Service

> **Service**: `sm-engine` | **gRPC Port**: 9071 | **Health**: 9116  
> **Origin**: Consolidated from sm-document + sm-memory + sm-profile  
> **Status**: Proposed | **Version**: 0.1.0

---

## Purpose

Unified engine service for the Supermemory adaptive knowledge graph. Handles document ingestion + chunking, memory creation with forgetting curve (Ebbinghaus decay), and dynamic user profile management. The document → memory → profile chain is now a single in-process workflow.

## Business Capability

- **Document Management**: CRUD, chunking, content extraction from web pages/articles
- **Memory Engine**: Memory creation, relationship tracking, forgetting curve decay
- **Profile Management**: Static preferences + dynamic traits derived from memory patterns
- **Forgetting Curve**: Ebbinghaus-based memory decay with configurable half-life
- **Content Extraction**: HTML parsing, metadata extraction, semantic chunking

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.23+ |
| RPC | gRPC (3 services: SmDocumentService + SmMemoryService + SmProfileService) |
| Database | PostgreSQL 17 + pgvector |
| Cache | Redis 7+ |
| Async | NATS JetStream |
| LLM | Bifrost (memory extraction, profile trait inference) |

## Quick Start

```bash
cd services/sm-engine
go run cmd/server/main.go
# gRPC: :9071 | Health: :9116
```

## Links

- [Architecture](./architecture.md)
- [Changelog](./changelog.md)

## Owner

Supermemory Engine Team
