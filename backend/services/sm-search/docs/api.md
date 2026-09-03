---
id: DOC-S02
service: sm-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-search — API Reference

> **Protocol**: gRPC (internal) | **Port**: 9073

## gRPC Service

```protobuf
service SmSearchService {
  rpc HybridSearch(SearchRequest) returns (SearchResponse);
  rpc MemorySearch(MemorySearchRequest) returns (MemorySearchResponse);
  rpc RAGComplete(RAGRequest) returns (RAGResponse);
}
```

## HybridSearch (v3)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| q | string | Yes | — | Search query (min 1 char) |
| limit | int | No | 10 | Max results (1-100) |
| container_tags | string[] | No | — | Scope to containers |
| filters | FilterExpr | No | — | AND/OR metadata filters |
| document_threshold | float | No | 0 | 0=broad, 1=strict doc filter |
| chunk_threshold | float | No | 0 | 0=broad, 1=strict chunk filter |
| rerank | bool | No | false | Cross-encoder reranking |
| rewrite_query | bool | No | false | LLM query rewriting (+400ms) |
| include_full_docs | bool | No | false | Return full document content |
| include_summary | bool | No | false | Return document summary |
| only_matching_chunks | bool | No | true | Skip context chunks |

**Response**: `{ results: SearchResult[], timing: float, total: int }`

Each SearchResult: `{ documentId, title, type, score, metadata, chunks[{content, score, isRelevant}], summary?, content?, createdAt, updatedAt }`

## MemorySearch (v4)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| q | string | Yes | — | Search query |
| container_tag | string | No | — | Single container scope |
| threshold | float | No | 0.6 | Similarity threshold |
| limit | int | No | 10 | Max results (1-100) |
| rerank | bool | No | false | Cross-encoder rerank |
| rewrite_query | bool | No | false | LLM query rewriting |
| include.documents | bool | No | false | Include source documents |
| include.summaries | bool | No | false | Include doc summaries |
| include.relatedMemories | bool | No | false | Include related memories |

**Response**: `{ results: MemorySearchResult[], timing: float, total: int }`

Each MemorySearchResult: `{ id, memory, similarity, version?, metadata, context?{parents[], children[]}, documents[]? }`

## Error Codes

| gRPC Code | Description |
|-----------|-------------|
| `INVALID_ARGUMENT` | Empty query, invalid threshold/limit |
| `RESOURCE_EXHAUSTED` | Rate limit exceeded |
| `INTERNAL` | Embedding/reranking failure |
