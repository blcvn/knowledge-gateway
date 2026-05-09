---
id: DOC-S01
service: cognee-search
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
owner: VNP Memory — Cognee Team
---

# cognee-search

> **Group**: Cognee (Semantic KG) | **gRPC Port**: 9013 | **Health Port**: 9093 | **Origin**: Cognee

## Purpose

Semantic search service with 15 retrieval strategies over knowledge graph + vector store. Provides RAG completion, graph-based reasoning, lexical search, temporal queries, and code search capabilities.

**Business Capability**: Multi-strategy knowledge retrieval from the Cognee semantic knowledge graph, powering the `memory.recall()` pipeline for semantic results.

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC (internal), NATS JetStream
- **Database**: Qdrant (vector similarity), Neo4j (graph traversal), Redis (result cache)
- **AI Integration**: Bifrost LLM Gateway (RAG completion, graph reasoning)
- **Architecture**: 4-layer Clean Architecture
- **DI**: Google Wire

## Quick Start

```bash
make build-cognee-search
make run-cognee-search
docker compose up cognee-search
curl http://localhost:9093/healthz
```

## Search Strategies (15 Types)

| # | Strategy | Source | Description |
|---|----------|--------|-------------|
| 1 | `SIMILARITY` | Qdrant | Vector cosine similarity search |
| 2 | `GRAPH_COMPLETION` | Neo4j+LLM | Graph traverse + LLM reasoning (default) |
| 3 | `RAG_COMPLETION` | Qdrant+LLM | Traditional RAG with document chunks |
| 4 | `NATURAL_LANGUAGE` | Neo4j | NL → Cypher query translation |
| 5 | `CHUNKS` | Qdrant | Raw chunk retrieval by similarity |
| 6 | `CHUNKS_LEXICAL` | BM25/Jaccard | Token-based exact keyword matching |
| 7 | `SUMMARIES` | Neo4j | Pre-generated hierarchical summaries |
| 8 | `TRIPLET_COMPLETION` | Neo4j+LLM | Triplet-based graph completion |
| 9 | `GRAPH_COMPLETION_COT` | Neo4j+LLM | Chain-of-thought graph reasoning |
| 10 | `GRAPH_COMPLETION_DECOMPOSITION` | Neo4j+LLM | Query decomposition + parallel graph search |
| 11 | `GRAPH_COMPLETION_CONTEXT_EXTENSION` | Neo4j+LLM | Extended context window graph completion |
| 12 | `GRAPH_SUMMARY_COMPLETION` | Neo4j+LLM | Community summary-based completion |
| 13 | `CYPHER` | Neo4j | Direct Cypher query execution |
| 14 | `TEMPORAL` | Neo4j | Temporal-aware graph search |
| 15 | `FEELING_LUCKY` | Auto | Auto-select best retrieval strategy |

## Links

- [API Reference](./api.md)
- [Architecture](./architecture.md)
- [Data Model](./data-model.md)
- [Configuration](./configuration.md)
- [Runbook](./runbook.md)
- [Changelog](./changelog.md)

## Owner

- **Team**: VNP Memory — Cognee
