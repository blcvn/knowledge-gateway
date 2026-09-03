---
id: TDD-cognee-search
title: Technical Design — cognee-search
service: cognee-search
version: 2.0.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Cognee
linked_sol: SOL-001
---

# Technical Design — cognee-search

> **Group**: Cognee | **gRPC Port**: 9013 | **Health Port**: 9093 | **Origin**: Cognee L5

## 1. Service Overview

15 retrieval strategies over knowledge graph + vector store. 3-phase pipeline: retrieve → merge (RRF) → rerank. RAG completion with LLM. Strategy pattern dispatcher routes queries to appropriate retriever implementations.

## 2. Search Strategies

| Type | Source | LLM Required | Performance |
|------|--------|-------------|-------------|
| SIMILARITY | Qdrant | No | Fast |
| GRAPH_COMPLETION | Neo4j+LLM | Yes | Slow (intelligent) |
| RAG_COMPLETION | Qdrant+LLM | Yes | Medium |
| NATURAL_LANGUAGE | Neo4j | Yes (NL→Cypher) | Medium |
| CHUNKS | Qdrant | No | Fastest |
| CHUNKS_LEXICAL | BM25 | No | Fast |
| SUMMARIES | Neo4j | No | Fast |
| TRIPLET_COMPLETION | Neo4j+LLM | Yes | Medium |
| GRAPH_COMPLETION_COT | Neo4j+LLM | Yes | Slow |
| GRAPH_COMPLETION_DECOMPOSITION | Neo4j+LLM | Yes | Slow |
| GRAPH_COMPLETION_CONTEXT_EXTENSION | Neo4j+LLM | Yes | Medium |
| GRAPH_SUMMARY_COMPLETION | Neo4j+LLM | Yes | Medium |
| CYPHER | Neo4j | No | Fast |
| TEMPORAL | Neo4j | No | Medium |
| FEELING_LUCKY | Auto | Maybe | Variable |

## 3. gRPC API

```protobuf
service CogneeSearchService {
  rpc Search(SearchRequest) returns (SearchResponse);
  rpc RAGComplete(RAGRequest) returns (RAGResponse);
  rpc GetChunks(GetChunksRequest) returns (ChunksResponse);
}
```

## 4. NATS Events

| Direction | Subject | Peer |
|-----------|---------|------|
| Subscribe | `cognee.pipeline.completed` | cognee-cognify (reindex) |

## 5. Cross-Service Dependencies

| Target | Protocol | Purpose |
|--------|----------|---------|
| vnp-search-hub | gRPC (called by) | Cross-engine search fan-out |

## 6. Multi-Tenancy

Tenant isolation via Qdrant tenant_id filter + Neo4j namespace labels.

---

## Feature Specs Registry

| ID | Title | Status | Priority | Phase |
|----|-------|--------|----------|-------|
| [FEAT-SEA-001](./features/FEAT-SEA-001-domain-usecase-layer.md) | Domain + Usecase (15 Strategies, RRF) | Ready | P0 | Phase 1 |
| [FEAT-SEA-002](./features/FEAT-SEA-002-adapter-layer.md) | Adapter Layer (15 Retrievers + gRPC) | Ready | P0 | Phase 2 |
| [FEAT-SEA-003](./features/FEAT-SEA-003-infra-wire.md) | Infrastructure + Wire DI | Ready | P0 | Phase 3 |

## Architecture Specs Registry

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| — | _To be populated_ | — | — |

## Technical Specs Registry

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| — | _To be populated_ | — | — |

## Quality Specs Registry

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| — | _To be populated_ | — | — |

---

> **Linked**: [SOL-001](../../cognee-pipeline/specs/solutions/SOL-001-implement-cognee-pipeline-service.md) | [Architecture Spec](../../../services/cognee/specs/services/04-cognee-search.md)
