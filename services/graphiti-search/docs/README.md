---
id: DOC-S01
service: graphiti-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
owner: VNP Memory — Graphiti Team
---

# graphiti-search

> **Group**: Graphiti (Episodic KG) | **gRPC Port**: 9022 | **Health Port**: 9095 | **Origin**: Graphiti

## Purpose

Hybrid search over the Graphiti temporal knowledge graph combining **vector similarity** (cosine), **full-text search** (BM25), **BFS graph traversal**, and **reranking** (RRF/MMR/Cross-Encoder). Supports search across nodes, edges, episodes, and communities with temporal and property filtering.

### Business Capability

- **HybridSearch**: Multi-strategy search combining cosine similarity + BM25 + BFS with configurable reranking
- **NodeSearch**: Entity node search by name embedding similarity or full-text
- **EdgeSearch**: Fact/relationship search with temporal validity filtering
- **CommunitySearch**: Community-level semantic search over aggregated knowledge
- **Configurable Reranking**: RRF (fast default), MMR (diversity-focused), Cross-Encoder (highest quality), Node Distance, Episode Mentions

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server
- **Search**: Vector similarity (cosine), BM25 full-text, BFS traversal
- **Reranking**: RRF, MMR, Cross-Encoder (via graphiti-knowledge), Node Distance, Episode Mentions
- **Architecture**: 4-layer Clean Architecture
- **DI**: Google Wire

## Quick Start

```bash
make build-graphiti-search
make run-graphiti-search
docker compose up graphiti-search
```

## Search Pipeline

```
1. Generate embedding (→ graphiti-knowledge.GenerateEmbedding)
2. Parallel fan-out:
   ├── graphiti-store.CosineSimilaritySearch(embedding)
   ├── graphiti-store.FulltextSearch(query)
   └── graphiti-store.BFSSearch(matched_nodes)
3. Merge + Rerank (RRF/MMR/Cross-Encoder)
4. Apply temporal + property filters
5. Return ranked results
```

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| `graphiti-knowledge` | gRPC (9023) | Embedding generation, cross-encoder reranking |
| `graphiti-store` | gRPC (9024) | Cosine similarity, full-text, BFS search |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)

## Owner

- **Team**: VNP Memory — Graphiti
