---
id: DOC-S01
service: sm-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Supermemory Team
---

# sm-search

> **Group**: Supermemory | **gRPC Port**: 9073 | **Health Port**: 9118 | **Origin**: Supermemory

## Purpose

Hybrid search engine combining **vector similarity** (pgvector HNSW) with **fulltext search**, supporting both document-level (v3 API) and memory-level (v4 API) search. Provides **RAG completion**, **query rewriting**, **cross-encoder reranking**, and **threshold-based filtering** with configurable sensitivity for both documents and chunks.

### Business Capability

- **Hybrid Search (v3)**: Vector + fulltext → RRF merge → optional rerank → chunk-level results with context window
- **Memory Search (v4)**: Direct memory entry search with relation context (parent/child chains)
- **Query Rewriting**: LLM-powered query expansion for better recall (~400ms added latency)
- **Reranking**: Cross-encoder reranking for precision optimization
- **Threshold Control**: Configurable document/chunk sensitivity (0=broad, 1=strict)
- **Container Tag Scoping**: Search within specific user/project containers
- **Metadata Filtering**: AND/OR expressions with string, numeric, boolean filters

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC, NATS JetStream
- **Database**: PostgreSQL + pgvector
- **Architecture**: 4-layer Clean Architecture
- **DI**: Google Wire

## Quick Start

```bash
make build-sm-search && make run-sm-search
docker compose up sm-search
```

## API Surface

```protobuf
service SmSearchService {
  // v3: Document chunk search
  rpc HybridSearch(SearchRequest) returns (SearchResponse);
  // v4: Memory entry search
  rpc MemorySearch(MemorySearchRequest) returns (MemorySearchResponse);
  // RAG completion
  rpc RAGComplete(RAGRequest) returns (RAGResponse);
}
```

### REST (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v3/search` | Document chunk search with filters |
| POST | `/v4/search` | Memory entry search with relation context |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL + pgvector | SQL | Vector + fulltext search |
| Bifrost | HTTP | Query embedding + rewriting + reranking |
| sm-document | NATS sub | `sm.document.created` → index chunks |
| sm-memory | NATS sub | `sm.memory.created` → index memory entries |

## Links

- [API](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md) · [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)

## Owner

- **Team**: VNP Memory — Supermemory
