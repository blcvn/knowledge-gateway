---
id: TDD-cognee-search
title: Technical Design — cognee-search
service: cognee-search
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
group: Cognee
---

# Technical Design — cognee-search

> **Group**: Cognee | **gRPC Port**: 9013 | **Origin**: Cognee L5

## 1. Service Overview

15 retrieval strategies over knowledge graph + vector store. RAG completion with LLM. Strategy pattern dispatcher routes queries to appropriate retriever implementations.

## 2. Search Strategies (from Cognee SearchType)

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

## 6. Observability

- Metrics: search latency per strategy, cache hit ratio
- Traces: OTel spans per retriever
- Logs: slog JSON with query_type, top_k, result_count

---

> **Next Steps**: Decompose into FEAT specs for each retriever strategy.
