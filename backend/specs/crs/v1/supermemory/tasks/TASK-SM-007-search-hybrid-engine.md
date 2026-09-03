# TASK-SM-007 — services/search-service: Hybrid Search (RAG + Memory)

**Task ID:** TASK-SM-007  
**Wave:** 3 (Intelligence)  
**Solution:** [SOL-SM-003](../solutions/SOL-SM-003-Hybrid-Search-Engine.md)  
**Depends on:** TASK-SM-006 (memory_entries with HNSW index)  
**Ước tính:** 5h  
**Priority:** High

---

## Mục tiêu

Nâng cấp `services/search-service/` thành Hybrid Search Engine:
1. **3-goroutine parallel search**: G1 (chunks) + G2 (memory entries) + G3 (doc summaries)
2. **Metadata Filter Engine**: JSONB SQL builder (AND/OR operators)
3. **Context assembly**: adjacent chunks + related memory chains
4. **V4 Memory-only API** với explicit similarity scores
5. **Query rewriter** (LLM via Bifrost, +400ms)
6. **AI Reranker** (LLM via Bifrost, optional)

---

## Công việc cụ thể

### 1. Tạo Domain Types

**`services/search-service/internal/domain/search.go`**

```go
type SearchRequest struct {
    Query             string
    SpaceID           string
    OrgID             string
    Limit             int          // default 10
    ChunkThreshold    float64      // default 0.7
    DocumentThreshold float64      // default 0.6
    MemoryThreshold   float64      // default 0.75
    Rerank            bool
    RewriteQuery      bool         // +400ms
    IncludeFullDocs   bool
    IncludeSummary    bool
    Filters           *FilterGroup
    Mode              SearchMode   // "hybrid" | "memories-only" | "documents-only"
}

type SearchResult struct {
    ID              string
    Type            ResultType    // "chunk" | "memory" | "document"
    Content         string
    Score           float64
    DocumentID      *string
    DocumentTitle   *string
    ChunkPosition   *int
    MemoryVersion   *int
    IsLatest        *bool
    Metadata        map[string]any
    AdjacentChunks  []string     // prev/next chunk content
    RelatedMemories []string     // parent/child memory chain
}
```

### 2. Implement Metadata Filter Engine

**`services/search-service/internal/domain/filter.go`**

```go
type FilterGroup struct {
    Operator   LogicalOp        // AND | OR
    Conditions []FilterCondition
}

type FilterCondition struct {
    Field    string
    Operator CompareOp  // eq|ne|gt|gte|lt|lte|contains|in
    Value    any
}

// BuildSQL generates safe parameterized JSONB SQL
// metadata->>'field' = $N
// (metadata->>'field')::numeric >= $N
// metadata->>'field' = ANY(ARRAY[$N, $N+1, ...])
func (f *FilterGroup) BuildSQL(paramOffset int) (string, []any)
```

### 3. Implement 3-Goroutine Parallel Search Orchestrator

**`services/search-service/internal/usecase/hybrid_search.go`**

```go
// G1: chunks.embedding <=> $query_vec (SpaceID via container_tags)
// G2: memory_entries.memory_embedding (is_latest=true, is_forgotten=false)
// G3: documents.summary_embedding (status='done')
// All 3 run concurrently → wait for all → merge + dedup by ID
// enrichWithContext: G1 result → adjacent chunks; G2 result → memory chain
// Optional: AI rerank via Bifrost LLM
func (uc *HybridSearchUseCase) Execute(ctx, req SearchRequest) ([]SearchResult, error)
```

### 4. pgvector SQL Queries

**`services/search-service/internal/infra/postgres/`**

```sql
-- chunk_searcher.go
SELECT c.id, c.document_id, c.content, c.position,
       1 - (c.embedding <=> $1) AS score
FROM chunks c JOIN documents d ON c.document_id = d.id
WHERE d.org_id = $2 AND d.container_tags @> $3::text[]
  AND 1 - (c.embedding <=> $1) > $4
  AND {filterSQL}
ORDER BY c.embedding <=> $1 LIMIT $5;

-- memory_searcher.go
SELECT id, content, is_static, version, space_id,
       1 - (memory_embedding <=> $1) AS score
FROM memory_entries
WHERE org_id = $2 AND space_id = $3
  AND is_latest = true AND is_forgotten = false
  AND 1 - (memory_embedding <=> $1) > $4
ORDER BY memory_embedding <=> $1 LIMIT $5;

-- doc_searcher.go
SELECT id, title, summary,
       1 - (summary_embedding <=> $1) AS score
FROM documents
WHERE org_id = $2 AND container_tags @> $3::text[] AND status = 'done'
  AND 1 - (summary_embedding <=> $1) > $4
ORDER BY summary_embedding <=> $1 LIMIT $5;
```

### 5. Implement V4 Memory-Only Search

**`services/search-service/internal/usecase/memory_search.go`**

```go
// POST /api/v4/search
// Returns explicit similarity scores per memory
// threshold: 0.5 (lower than hybrid)
type MemorySearchV4Result struct {
    ID       string
    Memory   string
    Score    float64   // cosine similarity
    IsStatic bool
    Version  int
    SpaceID  string
}
```

### 6. Query Rewriter & AI Reranker

**`services/search-service/internal/usecase/query_rewriter.go`**:
```go
// Bifrost LLM call: "Rewrite this search query to be more specific..."
// Only called when RewriteQuery=true
// Expected latency: ~400ms additional
```

**`services/search-service/internal/infra/reranker/llm_reranker.go`**:
```go
// Bifrost LLM: "Given these results, rank them from most to least relevant..."
// Only called when Rerank=true
```

### 7. REST Endpoints

```
POST /api/v1/search  → HybridSearch
POST /api/v4/search  → MemorySearchV4
```

### 8. Tests

- `TestHybridSearch_3GoroutinesRunConcurrently`: verify via trace/mock that all 3 run concurrently
- `TestHybridSearch_MemoryOnlyMode`: G1 + G3 return empty; only G2 results
- `TestHybridSearch_IsForgottenFiltered`: forgotten memories NOT in results
- `TestHybridSearch_MetadataFilter_LangVi`: filter lang=vi → only matching results
- `TestFilterGroup_BuildSQL_AND`: 2 conditions → "... AND ..."
- `TestFilterGroup_BuildSQL_OR`: 2 conditions → "... OR ..."
- `TestFilterGroup_BuildSQL_InOperator`: in:[1,2,3] → ANY(ARRAY[$1,$2,$3])
- `TestMemorySearchV4_ReturnsScores`: results have Score field
- `TestHybridSearch_DeduplicateByID`: same result in G1 + G2 → appears once

---

## Acceptance Criteria

- [ ] `go build ./services/search-service/...` không lỗi
- [ ] POST /api/v1/search → results include both chunk + memory types
- [ ] is_forgotten=true memories NOT in any results
- [ ] Metadata filter `{"field": "lang", "op": "eq", "value": "vi"}` → SQL WHERE clause
- [ ] POST /api/v4/search → each result has `score` field (0.0-1.0)
- [ ] RewriteQuery=true → `rewrittenQuery` in response (different from original)
- [ ] p95 latency (no rewrite) < 500ms (benchmark test)
- [ ] `go test ./services/search-service/...` pass

---

## Files tạo/sửa

```
services/search-service/
├── internal/
│   ├── domain/
│   │   ├── search.go              (NEW/MODIFY)
│   │   └── filter.go              (NEW)
│   ├── usecase/
│   │   ├── hybrid_search.go       (NEW)
│   │   ├── hybrid_search_test.go  (NEW)
│   │   ├── memory_search.go       (NEW)
│   │   └── query_rewriter.go      (NEW)
│   └── infra/
│       ├── postgres/
│       │   ├── chunk_searcher.go  (NEW)
│       │   ├── memory_searcher.go (NEW)
│       │   └── doc_searcher.go    (NEW)
│       └── reranker/
│           └── llm_reranker.go    (NEW)

gateway/adapter/handler/
└── search_handler.go  (MODIFY: thêm v4 endpoint)
```

## Sau khi hoàn thành

Chạy: `go build ./services/search-service/... && go test ./services/search-service/...`
