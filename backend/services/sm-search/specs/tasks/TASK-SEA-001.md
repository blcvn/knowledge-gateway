---
id: TASK-SEA-001
title: Domain Models & Core Algorithms
service: sm-search
status: Done
priority: P0
created: 2026-05-11
---

# Domain Models & Core Algorithms

## Objective
Implement the core domain entities, value objects, and algorithms.

## Specs Mapping
Please refer to the following content from `specs/tdd.md` to implement this task:

```markdown
---
id: TDD-sm-search
title: Technical Design — sm-search
service: sm-search
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Supermemory
---

# Technical Design — sm-search

> **Group**: Supermemory | **gRPC Port**: 9073 | **Health Port**: 9118

## 1. Service Overview

Hybrid search: vector similarity (pgvector HNSW) + fulltext → RRF merge → optional cross-encoder rerank. Dual API: v3 (document chunks) and v4 (memory entries with version context). Supports query rewriting, threshold-based filtering, and metadata AND/OR expressions.

## 2. Clean Architecture Layers

### Domain Layer
- **SearchRequest**: q, limit, container_tags, filters, thresholds (document/chunk 0-1), rerank, rewrite_query, include_full_docs, include_summary, only_matching_chunks
- **SearchResult**: document_id, title, type, score, metadata, chunks[{content, score, is_relevant}], summary, created_at
- **MemorySearchResult**: id, memory, similarity, version, metadata, context{parents[], children[]}, documents[]
- **FilterExpression**: AND/OR with key-value comparisons (string, numeric with operators, boolean, negate)

### Usecase Layer
- **HybridSearchUseCase**: embed query → parallel(vector search, fulltext search) → RRF merge → optional rerank → threshold filter → return
- **MemorySearchUseCase**: embed query → vector search memory_entries → fetch version context (parents/children) → optional related documents
- **RAGCompleteUseCase**: search → assemble context → LLM completion
- **QueryRewriteUseCase**: LLM rewrite query for better recall (~400ms)

### Adapter Layer
- **gRPC handler**: HybridSearch, MemorySearch, RAGComplete
- **PostgreSQL repos**: ChunkSearchRepository (pgvector), MemorySearchRepository
- **NATS subscriber**: `sm.document.created`, `sm.memory.created` for index updates

### Infrastructure Layer
- Config, Server, Wire, Telemetry, Bifrost client

## 3. Search Pipeline

```
SearchRequest{q, container_tags, filters, thresholds}
  │
  ├─ [Optional] Query Rewrite (LLM) → expanded_query (+400ms)
  │
  ├─ Embed query → VECTOR(1536)
  │
  ├─ Parallel:
  │   ├─ Vector search (pgvector HNSW, cosine similarity)
  │   └─ Fulltext search (PostgreSQL tsvector)
  │
  ├─ RRF Merge (Reciprocal Rank Fusion)
  │
  ├─ Apply thresholds:
  │   ├─ documentThreshold (0=all docs, 1=exact match only)
  │   └─ chunkThreshold (0=all chunks, 1=exact match only)
  │
  ├─ [Optional] Cross-encoder rerank (Bifrost)
  │
  ├─ Metadata filter (AND/OR expressions)
  │
  ├─ Add context chunks (prev/next) unless onlyMatchingChunks=true
  │
  └─ Return: {results[], timing, total}
```

## 4. gRPC API

```protobuf
service SmSearchService {
  rpc HybridSearch(SearchRequest) returns (SearchResponse);
  rpc MemorySearch(MemorySearchRequest) returns (MemorySearchResponse);
  rpc RAGComplete(RAGRequest) returns (RAGResponse);
}
```

## 5. NATS Events

| Direction | Subject | Purpose |
|-----------|---------|---------|
| Subscribe | `sm.document.created` | Index new document chunks |
| Subscribe | `sm.document.deleted` | Remove document from index |
| Subscribe | `sm.memory.created` | Index new memory entry |
| Subscribe | `sm.memory.forgotten` | Remove forgotten memory from index |

## 6. Data Model

Uses `documents`, `chunks`, `memory_entries` tables from sm-document and sm-memory (read replicas). Search-specific indexes:
- `idx_chunk_embedding` HNSW (lists=100, m=16) on chunks.embedding
- `idx_chunk_fulltext` GIN on chunks.content (tsvector)
- `idx_memory_embedding` HNSW on memory_entries.memory_embedding
- `idx_doc_summary_emb` HNSW on documents.summary_embedding

## 7. Observability

- **Metrics**: search_total, search_latency_seconds (histogram), rerank_latency, rewrite_latency, results_count
- **Traces**: OTel spans for embed, vector_search, fulltext_search, rrf_merge, rerank
- **Health**: gRPC + HTTP /healthz on port 9118

## 8. SLA Targets

| Metric | Target |
|--------|--------|
| Search latency (p95) | < 300ms (without rerank/rewrite) |
| Search latency (p95) | < 800ms (with rerank + rewrite) |
| RAG completion (p95) | < 2s |

---

> **Next Steps**: FEAT-001 (Hybrid Search Pipeline), FEAT-002 (Memory Search v4), FEAT-003 (Query Rewriting), FEAT-004 (RAG Completion)

## Task Specs Registry

_To be populated during implementation._

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| TASK-SEA-001 | Implement Domain Models | Pending | P0 |
| TASK-SEA-002 | Implement Usecases | Pending | P0 |
| TASK-SEA-003 | Implement Adapters and Repositories | Pending | P0 |
| TASK-SEA-004 | Infrastructure and Telemetry setup | Pending | P1 |

```

## Acceptance Criteria
- [x] Domain models compile and have no external dependencies.
- [x] Core algorithms are fully implemented and unit tested.
