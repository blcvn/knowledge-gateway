# 04 — Search Service

> **gRPC**: 9003 | **Health**: 9083

---

## 1. Purpose

Hybrid search engine: RAG (document chunks) + Memory search (extracted facts), vector similarity (pgvector), query rewriting, reranking, metadata filtering, và context assembly.

---

## 2. Clean Architecture

```
services/search-service/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # SearchResult, ChunkResult, MemoryResult
│   │   ├── value_object.go     # SearchMode, FilterOperator, SortOrder
│   │   └── errors.go           # ErrEmptyQuery, ErrInvalidThreshold
│   ├── usecase/
│   │   ├── hybrid_search.go    # V3: RAG + Memory combined search
│   │   ├── memory_search.go    # V4: Memory-only search with version context
│   │   ├── rewrite_query.go    # AI query optimization
│   │   ├── rerank.go           # AI result reranking
│   │   ├── port/
│   │   │   ├── input.go        # HybridSearchUC, MemorySearchUC
│   │   │   └── output.go       # VectorSearcher, MetadataFilterer,
│   │   │                       # QueryRewriter, Reranker, EmbeddingGenerator
│   │   └── dto/
│   │       ├── search.go       # HybridSearchInput, MemorySearchInput
│   │       └── result.go       # SearchOutput with timing
│   ├── adapter/
│   │   ├── grpc/handler.go     # SearchServiceServer implementation
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── chunk_search.go    # pgvector similarity on chunks
│   │   │       ├── memory_search.go   # pgvector similarity on memories
│   │   │       ├── document_search.go # Document-level similarity
│   │   │       └── metadata_filter.go # Dynamic filter builder (AND/OR)
│   │   ├── embedding/
│   │   │   └── provider.go     # Query embedding generation
│   │   ├── llm/
│   │   │   ├── query_rewriter.go   # AI query rewriting (+~400ms)
│   │   │   └── reranker.go         # AI result reranking
│   │   └── event/
│   │       └── subscriber.go  # memory.created → update search index
│   └── infra/
│       ├── config/config.go
│       └── wire/wire.go
├── migrations/
│   ├── 001_create_pgvector_extension.up.sql
│   └── 002_create_search_indexes.up.sql
└── Dockerfile
```

---

## 3. Search Pipeline

```
SearchRequest(q, containerTags, limit, thresholds, filters, options)
         │
         ▼
┌── Phase 1: PREPARE ─────────────────────────────────────┐
│  1. Optional: QueryRewriter.Rewrite(q) → optimized_q    │
│  2. EmbeddingGenerator.Generate(q) → query_vector        │
│  3. Build metadata filter SQL (AND/OR operators)         │
└────────────┬────────────────────────────────────────────┘
             ▼
┌── Phase 2: SEARCH (Parallel) ───────────────────────────┐
│  ┌─ Goroutine 1 ──────────────────────────────────────┐ │
│  │  Vector search on chunks (pgvector cosine)          │ │
│  │  WHERE similarity > chunkThreshold                  │ │
│  │    AND org_id = $1 AND container_tags && $2         │ │
│  └─────────────────────────────────────────────────────┘ │
│  ┌─ Goroutine 2 ──────────────────────────────────────┐ │
│  │  Vector search on memory_entries (pgvector cosine)  │ │
│  │  WHERE similarity > threshold AND is_latest = true  │ │
│  │    AND is_forgotten = false                         │ │
│  └─────────────────────────────────────────────────────┘ │
│  ┌─ Goroutine 3 ──────────────────────────────────────┐ │
│  │  Document-level similarity on summary_embedding     │ │
│  │  WHERE similarity > documentThreshold               │ │
│  └─────────────────────────────────────────────────────┘ │
└────────────┬────────────────────────────────────────────┘
             ▼
┌── Phase 3: MERGE + ENRICH ──────────────────────────────┐
│  1. Deduplicate by result ID                             │
│  2. Apply metadata filters                               │
│  3. Optional: Reranker.Rerank(results, query)            │
│  4. Context assembly:                                    │
│     ├── Adjacent chunks (prev/next position)             │
│     ├── Related memories (parents/children chain)        │
│     ├── Document summaries (if includeSummary)           │
│     └── Full documents (if includeFullDocs)              │
│  5. Apply final limit + pagination                       │
└────────────┬────────────────────────────────────────────┘
             ▼
SearchResponse(results[], total, timing_ms)
```

---

## 4. Metadata Filter Engine

```go
// Dynamic SQL filter builder
type FilterGroup struct {
    Operator  LogicalOp  // AND | OR
    Conditions []FilterCondition
}

type FilterCondition struct {
    Field    string       // metadata JSONB field
    Operator CompareOp   // eq, ne, gt, gte, lt, lte, contains, in
    Value    any
}

// Generated SQL:
// WHERE metadata->>'category' = 'tech'
//   AND (metadata->>'priority')::int >= 3
//   AND metadata->>'tags' ? 'important'
```

---

## 5. gRPC Interface

```protobuf
service SearchService {
  rpc HybridSearch(HybridSearchRequest) returns (HybridSearchResponse);
  rpc MemorySearch(MemorySearchRequest) returns (MemorySearchResponse);
}

message HybridSearchRequest {
  string query = 1;
  repeated string container_tags = 2;
  int32  limit = 3;                    // 1-100, default 10
  double chunk_threshold = 4;          // 0.0-1.0
  double document_threshold = 5;       // 0.0-1.0
  FilterGroup filters = 6;
  bool include_full_docs = 7;
  bool include_summary = 8;
  bool only_matching_chunks = 9;
  bool rerank = 10;
  bool rewrite_query = 11;
}

message MemorySearchRequest {
  string query = 1;
  string container_tag = 2;
  double threshold = 3;               // default 0.6
  int32  limit = 4;
  MemorySearchIncludes includes = 5;  // documents, summaries, relatedMemories
  bool rerank = 6;
}
```
