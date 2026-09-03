---
id: DOC-S01
service: graphiti-knowledge
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
owner: VNP Memory — Graphiti Team
---

# graphiti-knowledge

> **Group**: Graphiti (Episodic KG) | **gRPC Port**: 9023 | **Health Port**: 9096 | **Origin**: Graphiti

## Purpose

LLM-intensive processing engine for the Graphiti temporal knowledge graph. Handles all AI-powered operations: **entity extraction**, **edge/relationship extraction**, **entity/edge resolution** (deduplication), **community detection/update**, **embedding generation**, and **cross-encoder reranking**.

### Business Capability

- **Entity Extraction**: LLM-based extraction of named entities from episodic content with structured JSON output (`{name, type, summary}[]`)
- **Entity Resolution**: Deduplication of extracted entities against existing graph entities using LLM comparison
- **Edge Extraction**: Relationship/fact extraction with temporal validity markers (`{source, target, type, fact, valid_at, invalid_at}[]`)
- **Edge Resolution**: Deduplication and invalidation of extracted edges against existing graph
- **Embedding Generation**: Vector embedding creation for entity names and edge facts
- **Reranking**: Cross-encoder reranking for search result quality enhancement
- **Community Update**: Louvain-based community detection and summary regeneration

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server
- **LLM Gateway**: Bifrost (GPT-4o for extraction, GPT-4o-mini for resolution)
- **Embedder**: OpenAI text-embedding-3-large (via Bifrost)
- **Cross-Encoder**: OpenAI Reranker (via Bifrost)
- **Architecture**: 4-layer Clean Architecture
- **DI**: Google Wire

## LLM Integration

| Operation | Model | Structured Output | Avg Latency |
|-----------|-------|-------------------|-------------|
| ExtractEntities | GPT-4o | `{name, type, summary}[]` | ~2-5s |
| ResolveEntities | GPT-4o-mini | `{source_id, target_id, is_duplicate}[]` | ~1-3s |
| ExtractEdges | GPT-4o | `{source, target, type, fact, temporal}[]` | ~3-8s |
| ResolveEdges | GPT-4o-mini | `{edge_id, action: keep\|merge\|invalidate}[]` | ~1-3s |
| GenerateEmbedding | text-embedding-3-large | `float64[]` | ~100-300ms |
| Rerank | Cross-Encoder | `{relevance_score}[]` | ~500ms-2s |
| UpdateCommunity | GPT-4o-mini | `{community_name, summary}` | ~1-3s |

## Quick Start

```bash
make build-graphiti-knowledge
make run-graphiti-knowledge
docker compose up graphiti-knowledge
```

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)

## Owner

- **Team**: VNP Memory — Graphiti
