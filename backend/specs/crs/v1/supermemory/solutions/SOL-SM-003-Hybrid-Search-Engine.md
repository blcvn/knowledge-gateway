# Solution: SOL-SM-003 — Hybrid Search Engine (RAG + Memory)

**CR ID:** CR-SM-003  
**Solution ID:** SOL-SM-003  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Nâng cấp `services/search-service/` thành Hybrid Search Engine kết hợp 3 nguồn dữ liệu song song: **document chunks** (RAG), **memory entries** (Facts), và **document summaries**. Sử dụng pgvector HNSW index đã có sẵn trong PostgreSQL infrastructure.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| `search-service/` | `services/search-service/` | Có: multi-strategy search cơ bản |
| `sm-search` gRPC service | `apps/memory/internal/bootstrap/` | Có: vector search đơn giản |
| `vnp-search-hub` | `apps/memory/internal/bootstrap/` | Cross-engine search (hiện search tất cả engines) |
| pgvector | PostgreSQL infra | Đã cài, cần thêm HNSW indexes |

### Gap phân tích

- Hiện chỉ search trên document chunks, chưa hợp nhất với `memory_entries`
- Thiếu 3-goroutine parallel search pattern
- Thiếu query rewriting (AI-assisted)
- Thiếu metadata filtering (AND/OR operators trên JSONB)
- Thiếu adjacent chunk context assembly
- Thiếu `/api/v4/search` (memory-only với similarity scores)

---

## 3. Thiết kế Giải pháp

### 3.1. Cấu trúc Search Service (Nâng cấp)

```
services/search-service/
├── internal/
│   ├── domain/
│   │   ├── search.go          # SearchRequest, SearchResult, FilterGroup types
│   │   ├── filter.go          # FilterEngine: SQL builder cho metadata
│   │   └── repository.go      # ChunkSearcher, MemorySearcher, DocSearcher ports
│   ├── usecase/
│   │   ├── hybrid_search.go   # 3-goroutine parallel search orchestrator
│   │   ├── memory_search.go   # V4: memory-only search với scores
│   │   └── query_rewriter.go  # LLM query rewriting
│   └── infra/
│       ├── postgres/
│       │   ├── chunk_searcher.go    # pgvector cosine search trên chunks
│       │   ├── memory_searcher.go   # pgvector cosine search trên memory_entries
│       │   └── doc_searcher.go      # pgvector cosine search trên doc summaries
│       └── reranker/
│           └── llm_reranker.go      # AI rerank qua Bifrost
```

### 3.2. Domain Model

```go
// services/search-service/internal/domain/search.go

type SearchRequest struct {
    Query              string
    SpaceID            string     // Container tag filter
    OrgID              string
    Limit              int        // Default 10
    ChunkThreshold     float64    // Default 0.7
    DocumentThreshold  float64    // Default 0.6
    MemoryThreshold    float64    // Default 0.75
    Rerank             bool       // AI reranking
    RewriteQuery       bool       // AI query rewriting (+400ms)
    IncludeFullDocs    bool       // Include full document content
    IncludeSummary     bool       // Include document summaries
    Filters            *FilterGroup
    Mode               SearchMode // hybrid | memories-only | documents-only
}

type SearchMode string

const (
    SearchModeHybrid      SearchMode = "hybrid"
    SearchModeMemoryOnly  SearchMode = "memories-only"
    SearchModeDocumentOnly SearchMode = "documents-only"
)

type SearchResult struct {
    ID             string
    Type           ResultType  // "chunk" | "memory" | "document"
    Content        string
    Score          float64     // Cosine similarity score
    DocumentID     *string
    DocumentTitle  *string
    ChunkPosition  *int
    MemoryVersion  *int
    IsLatest       *bool
    Metadata       map[string]any

    // Context assembly
    AdjacentChunks []string    // Prev/next chunk content
    RelatedMemories []string   // Parent/child memory chain
}

type ResultType string

const (
    ResultChunk    ResultType = "chunk"
    ResultMemory   ResultType = "memory"
    ResultDocument ResultType = "document"
)
```

### 3.3. Metadata Filter Engine

```go
// services/search-service/internal/domain/filter.go

type LogicalOp string
const (
    LogicalAND LogicalOp = "AND"
    LogicalOR  LogicalOp = "OR"
)

type CompareOp string
const (
    CompareEq       CompareOp = "eq"
    CompareNe       CompareOp = "ne"
    CompareGt       CompareOp = "gt"
    CompareGte      CompareOp = "gte"
    CompareLt       CompareOp = "lt"
    CompareLte      CompareOp = "lte"
    CompareContains CompareOp = "contains"
    CompareIn       CompareOp = "in"
)

type FilterGroup struct {
    Operator   LogicalOp
    Conditions []FilterCondition
}

type FilterCondition struct {
    Field    string
    Operator CompareOp
    Value    any
}

// BuildSQL generates safe parameterized SQL for JSONB metadata filtering
func (f *FilterGroup) BuildSQL(paramOffset int) (string, []any) {
    parts := make([]string, 0, len(f.Conditions))
    args := make([]any, 0)
    idx := paramOffset

    for _, c := range f.Conditions {
        part, arg := buildConditionSQL(c, idx)
        parts = append(parts, part)
        args = append(args, arg...)
        idx += len(arg)
    }

    joiner := " AND "
    if f.Operator == LogicalOR {
        joiner = " OR "
    }
    return "(" + strings.Join(parts, joiner) + ")", args
}

func buildConditionSQL(c FilterCondition, idx int) (string, []any) {
    // JSONB access: metadata->>'field'
    // Type casting: (metadata->>'field')::int
    field := fmt.Sprintf("metadata->>'%s'", c.Field)
    switch c.Operator {
    case CompareEq:
        return fmt.Sprintf("%s = $%d", field, idx), []any{fmt.Sprint(c.Value)}
    case CompareGte:
        return fmt.Sprintf("(%s)::numeric >= $%d", field, idx), []any{c.Value}
    case CompareIn:
        vals := c.Value.([]any)
        // Generate $1, $2, ... for IN clause
        placeholders := make([]string, len(vals))
        for i := range vals { placeholders[i] = fmt.Sprintf("$%d", idx+i) }
        return fmt.Sprintf("%s = ANY(ARRAY[%s])", field, strings.Join(placeholders, ",")), vals
    // ... other operators
    default:
        return "true", nil
    }
}
```

### 3.4. Hybrid Search Orchestrator (3-Goroutine Pattern)

```go
// services/search-service/internal/usecase/hybrid_search.go

type HybridSearchUseCase struct {
    embedder      EmbedderPort
    chunkSearcher ChunkSearcher
    memSearcher   MemorySearcher
    docSearcher   DocSearcher
    reranker      RerankerPort
    queryRewriter QueryRewriterPort
    chunkRepo     ChunkRepository  // Để fetch adjacent chunks
    memRepo       MemoryRepository // Để fetch related memories
}

func (uc *HybridSearchUseCase) Execute(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
    // === Phase 1: PREPARE ===

    query := req.Query
    // Optional: AI Query Rewrite (+400ms)
    if req.RewriteQuery {
        rewritten, err := uc.queryRewriter.Rewrite(ctx, query)
        if err == nil { query = rewritten }
    }

    // Generate query embedding
    embedding, err := uc.embedder.Embed(ctx, query)
    if err != nil { return nil, err }

    // Build metadata filter SQL
    var filterSQL string
    var filterArgs []any
    if req.Filters != nil {
        filterSQL, filterArgs = req.Filters.BuildSQL(1)
    }

    // === Phase 2: PARALLEL SEARCH (3 goroutines) ===

    type searchResult struct {
        results []SearchResult
        err     error
    }

    chunkCh := make(chan searchResult, 1)
    memoryCh := make(chan searchResult, 1)
    docCh := make(chan searchResult, 1)

    // G1: pgvector cosine search on chunks
    go func() {
        if req.Mode == SearchModeMemoryOnly {
            chunkCh <- searchResult{nil, nil}
            return
        }
        results, err := uc.chunkSearcher.Search(ctx, ChunkSearchParams{
            OrgID:     req.OrgID,
            SpaceID:   req.SpaceID,
            Embedding: embedding,
            Threshold: req.ChunkThreshold,
            Limit:     req.Limit * 2, // Over-fetch for dedup
            FilterSQL: filterSQL,
            FilterArgs: filterArgs,
        })
        chunkCh <- searchResult{results, err}
    }()

    // G2: pgvector cosine search on memory_entries (is_latest=true, is_forgotten=false)
    go func() {
        if req.Mode == SearchModeDocumentOnly {
            memoryCh <- searchResult{nil, nil}
            return
        }
        results, err := uc.memSearcher.Search(ctx, MemorySearchParams{
            OrgID:     req.OrgID,
            SpaceID:   req.SpaceID,
            Embedding: embedding,
            Threshold: req.MemoryThreshold,
            Limit:     req.Limit * 2,
        })
        memoryCh <- searchResult{results, err}
    }()

    // G3: pgvector cosine search on document summaries
    go func() {
        if req.Mode == SearchModeMemoryOnly {
            docCh <- searchResult{nil, nil}
            return
        }
        results, err := uc.docSearcher.SearchSummaries(ctx, DocSearchParams{
            OrgID:     req.OrgID,
            SpaceID:   req.SpaceID,
            Embedding: embedding,
            Threshold: req.DocumentThreshold,
            Limit:     req.Limit,
        })
        docCh <- searchResult{results, err}
    }()

    chunkR := <-chunkCh
    memR := <-memoryCh
    docR := <-docCh

    // === Phase 3: MERGE + ENRICH ===

    allResults := deduplicateByID(
        append(chunkR.results, append(memR.results, docR.results...)...),
    )

    // Context assembly
    if len(allResults) > 0 {
        uc.enrichWithContext(ctx, allResults, req)
    }

    // Apply limit + threshold
    filtered := filterByThreshold(allResults, minThreshold(req))
    sorted := sortByScore(filtered)

    // Optional: AI Rerank
    if req.Rerank && len(sorted) > 0 {
        sorted, _ = uc.reranker.Rerank(ctx, query, sorted)
    }

    limit := req.Limit
    if len(sorted) < limit { limit = len(sorted) }
    return sorted[:limit], nil
}

// enrichWithContext adds adjacent chunks and related memories
func (uc *HybridSearchUseCase) enrichWithContext(ctx context.Context, results []SearchResult, req SearchRequest) {
    for i := range results {
        switch results[i].Type {
        case ResultChunk:
            // Fetch adjacent chunks (prev/next position)
            adjacent, _ := uc.chunkRepo.GetAdjacent(ctx, *results[i].DocumentID, *results[i].ChunkPosition)
            results[i].AdjacentChunks = extractContent(adjacent)

        case ResultMemory:
            // Fetch parent/child chain
            chain, _ := uc.memRepo.GetChain(ctx, results[i].ID)
            results[i].RelatedMemories = extractContent(chain)
        }
    }
}
```

### 3.5. V4 Memory Search API (Memory-Only với Scores)

```go
// services/search-service/internal/usecase/memory_search.go

type MemorySearchV4UseCase struct {
    embedder    EmbedderPort
    memSearcher MemorySearcher
}

type MemorySearchV4Result struct {
    ID              string
    Memory          string
    Score           float64      // Explicit similarity score
    IsStatic        bool
    Version         int
    SpaceID         string
    Metadata        map[string]any
}

func (uc *MemorySearchV4UseCase) Execute(ctx context.Context, req SearchRequest) ([]MemorySearchV4Result, error) {
    embedding, _ := uc.embedder.Embed(ctx, req.Query)
    
    memories, _ := uc.memSearcher.SearchWithScores(ctx, MemorySearchParams{
        OrgID:     req.OrgID,
        SpaceID:   req.SpaceID,
        Embedding: embedding,
        Threshold: 0.5, // Lower threshold cho V4
        Limit:     req.Limit,
    })

    results := make([]MemorySearchV4Result, 0, len(memories))
    for _, m := range memories {
        results = append(results, MemorySearchV4Result{
            ID: m.ID, Memory: m.Content,
            Score: m.Score, IsStatic: m.IsStatic,
            Version: m.Version, SpaceID: m.SpaceID,
        })
    }
    return results, nil
}
```

### 3.6. pgvector SQL Queries

```sql
-- G1: Chunk search với metadata filter
SELECT 
    c.id, c.document_id, c.content, c.position,
    1 - (c.embedding <=> $1) AS score
FROM chunks c
JOIN documents d ON c.document_id = d.id
WHERE d.org_id = $2
  AND d.container_tags @> $3::text[]   -- spaceID filter
  AND 1 - (c.embedding <=> $1) > $4    -- threshold
  AND {filterSQL}                       -- dynamic metadata filter
ORDER BY c.embedding <=> $1
LIMIT $5;

-- G2: Memory search (is_latest=true, is_forgotten=false)
SELECT 
    id, content, is_static, version, space_id,
    1 - (memory_embedding <=> $1) AS score
FROM memory_entries
WHERE org_id = $2
  AND space_id = $3
  AND is_latest = true
  AND is_forgotten = false
  AND 1 - (memory_embedding <=> $1) > $4
ORDER BY memory_embedding <=> $1
LIMIT $5;

-- G3: Document summary search
SELECT 
    id, title, summary,
    1 - (summary_embedding <=> $1) AS score
FROM documents
WHERE org_id = $2
  AND container_tags @> $3::text[]
  AND status = 'done'
  AND 1 - (summary_embedding <=> $1) > $4
ORDER BY summary_embedding <=> $1
LIMIT $5;
```

---

## 4. API Endpoints (Gateway)

```go
// gateway/adapter/handler/search_handler.go

// POST /api/v1/search  — Hybrid search
// POST /api/v4/search  — Memory-only search với explicit scores

func (h *SearchHandler) Register(mux *http.ServeMux) {
    mux.HandleFunc("POST /api/v1/search", h.HybridSearch)
    mux.HandleFunc("POST /api/v4/search", h.MemorySearchV4)
}
```

**Response (Hybrid):**
```json
{
  "results": [
    {
      "id": "chunk_abc",
      "type": "chunk",
      "content": "Go's goroutines provide lightweight concurrency...",
      "score": 0.892,
      "documentId": "doc_123",
      "documentTitle": "Go Performance Guide",
      "chunkPosition": 5,
      "adjacentChunks": ["...prev chunk...", "...next chunk..."]
    },
    {
      "id": "mem_xyz",
      "type": "memory",
      "content": "User prefers Go over Python for performance",
      "score": 0.841,
      "isLatest": true,
      "version": 2
    }
  ],
  "query": "Go performance optimization",
  "rewrittenQuery": "Go language performance tuning best practices",
  "totalResults": 2,
  "latencyMs": 245
}
```

---

## 5. Performance Target

| Metric | Target | Cách đạt |
|--------|--------|---------|
| p95 latency (no rewrite) | < 500ms | 3 parallel goroutines + HNSW index |
| p95 latency (with rewrite) | < 900ms | ~400ms LLM rewrite + 500ms search |
| Index build time (1M chunks) | < 30 min | `ef_construction=128, m=16` HNSW |
| Recall@10 (pgvector HNSW) | > 95% | `ef_search=64` at query time |

```sql
-- HNSW index creation với tuning parameters
CREATE INDEX ON chunks USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 128);

CREATE INDEX ON memory_entries USING hnsw (memory_embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 128);

-- Query time setting
SET hnsw.ef_search = 64;
```

---

## 6. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | Domain model + Filter Engine (SQL builder) | 1 ngày |
| **P2** | Chunk searcher + Memory searcher (pgvector) | 2 ngày |
| **P3** | 3-goroutine parallel search orchestrator | 2 ngày |
| **P4** | Context assembly (adjacent chunks + memory chain) | 1 ngày |
| **P5** | V4 Memory-only search API | 1 ngày |
| **P6** | Query rewriter (Bifrost LLM) | 1 ngày |
| **P7** | AI Reranker (Bifrost LLM) | 1 ngày |
| **P8** | Gateway integration + REST handlers | 1 ngày |
| **P9** | HNSW index creation + pgvector tuning | 1 ngày |
| **P10** | Tests + Performance benchmarks | 2 ngày |

**Tổng:** ~13 ngày (Wave 3)

---

## 7. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| "Python" → chunks VÀ memory entries | 3-goroutine parallel: G1 (chunks) + G2 (memories) |
| rewriteQuery=true → relevant + latency+400ms | LLM rewriter qua Bifrost |
| Metadata filter lang=vi | FilterGroup → dynamic SQL WHERE clause |
| Không trả về isForgotten=true memories | `AND is_forgotten = false` trong G2 SQL |
| V4 trả về similarity_score | MemorySearchV4Result.Score field |
| p95 < 500ms (no rewrite) | HNSW index + parallel goroutines |
