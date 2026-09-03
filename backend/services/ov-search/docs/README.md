---
id: DOC-S01
service: ov-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
owner: VNP Memory — OpenViking Team
---

# ov-search

> **Group**: OpenViking | **gRPC Port**: 9052 | **Health Port**: 9105 | **Origin**: OpenViking

## Purpose

Hierarchical retrieval service with **score propagation** (child → parent directory), **hotness scoring** (recency-weighted boosting), **convergence detection** (stop when quality plateaus), and **tiered loading** (L0 → L1 → L2 on demand). Replaces Python `openviking/retrieve/hierarchical_retriever.py`.

### Business Capability

- **Hierarchical Search**: Query → embedding → vector search → score propagation across directory tree
- **Hotness Scoring**: Boost recently accessed/modified files; session commit triggers hotness updates
- **Reranking**: Cross-encoder, RRF (Reciprocal Rank Fusion), MMR (Maximal Marginal Relevance)
- **Convergence Detection**: Stop retrieval when marginal quality gain < threshold
- **Tiered Loading**: Progressive detail — L0 summaries first, upgrade to L1/L2 on demand
- **Embedding Management**: Upsert/delete embeddings for file content indexing

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server
- **Vector DB**: Qdrant / pgvector (hybrid dense + sparse search)
- **Database**: PostgreSQL (hotness scores, search metadata)
- **Architecture**: 4-layer Clean Architecture
- **DI**: Google Wire

## Quick Start

```bash
make build-ov-search
make run-ov-search
docker compose up ov-search postgresql qdrant nats
```

## API Surface

### gRPC Service

```protobuf
service OvSearchService {
  rpc HierarchicalSearch(SearchRequest) returns (SearchResponse);
  rpc RetrieveContext(ContextRequest) returns (ContextResponse);
  rpc GetHotness(HotnessRequest) returns (HotnessResponse);
  rpc UpsertEmbedding(UpsertRequest) returns (google.protobuf.Empty);
  rpc DeleteEmbedding(DeleteRequest) returns (google.protobuf.Empty);
}
```

### Search Pipeline

```
1. Query → embedding (via Bifrost LLM Gateway)
2. Vector search (dense + sparse hybrid via Qdrant)
3. Score propagation (child → parent directory)
4. Hotness boost (recently accessed/modified)
5. Reranking (cross-encoder or RRF/MMR)
6. Convergence detection (stop when quality plateaus)
7. Tiered loading: L0 → L1 → L2 on demand
```

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| Qdrant / pgvector | Native | Vector similarity search |
| Bifrost (LLM Gateway) | gRPC | Query embedding generation |
| ov-fs | gRPC | Retrieve file content for tiered loading |
| PostgreSQL | SQL | Hotness scores, search metadata |

## NATS Events

| Event | Direction | Description |
|-------|-----------|-------------|
| `ov.content.written` | Subscribe | Generate embedding + upsert to vector DB |
| `ov.content.deleted` | Subscribe | Remove embedding from vector DB |
| `ov.resource.ingested` | Subscribe | Index newly ingested resources |
| `ov.session.committed` | Subscribe | Update hotness scores for referenced files |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)
- [Architecture Spec](../../../specs/architecture/05-openviking-services.md)

## Owner

- **Team**: VNP Memory — OpenViking
