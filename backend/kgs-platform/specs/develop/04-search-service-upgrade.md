# search-service Upgrade — KGS Search + Query Intelligence

> **Strategy:** 🔄 UPGRADE existing `services/search-service/`  
> **Absorbs:** search-service (KGS hybrid) + query-intel-service  
> **Effort:** 5 ngày  
> **Priority:** P1

---

## 1. Tại Sao search-service Là Đúng Chỗ

`search-service` đã có:
- `SearchOrchestrator` với **fan-out pattern** + **RRF merge** → gần như giống hệt KGS hybrid search
- `rrfMerge()` function → tái sử dụng trực tiếp
- **Graceful degradation** per engine → pattern phù hợp với KGS multi-source search
- **KG client** (`adapter/kg/client.go`) → đã có bridge sang kg-service
- MCP tool support → sẽ extend cho KGS search tools

---

## 2. Cấu Trúc Sau Upgrade

```
services/search-service/
├── cmd/server/
│   └── main.go              [MODIFY] Wire KGS deps
│
├── internal/
│   ├── usecase/
│   │   ├── orchestrator/    [UNCHANGED] cross-engine search (graphiti, cognee, memobase, sm)
│   │   ├── connector/       [UNCHANGED]
│   │   ├── mcp/             [UNCHANGED]
│   │   └── kgs/             [NEW PACKAGE]
│   │       ├── hybrid.go    ← KGS hybrid search (Qdrant + PG fulltext + Neo4j centrality)
│   │       ├── query_intel.go ← Traversal: Context, Impact, Coverage, Subgraph
│   │       ├── analytics.go ← Coverage report, Traceability matrix
│   │       └── view.go      ← View definitions + Projection engine
│   │
│   ├── domain/
│   │   ├── search/          [UNCHANGED]
│   │   └── kgs/             [NEW]
│   │       ├── search.go    ← KGS search request/response types
│   │       └── query.go     ← Query intel types
│   │
│   ├── adapter/
│   │   └── grpc/
│   │       ├── router.go         [MODIFY] Thêm KGS routes
│   │       ├── kgs_search.go     [NEW] /v1/kgs/search/** + /v1/search (KGS)
│   │       └── kgs_query.go      [NEW] /v1/graph/nodes/*/ + /v1/analytics/** + /v1/views/**
│   │
│   └── infra/
│       ├── kg/              [EXISTING] HTTP client gọi kg-service
│       ├── memory/          [EXISTING]
│       ├── storage/         [EXISTING]
│       ├── qdrant/          [NEW] Vector search (từ data/qdrant.go)
│       ├── neo4j/           [NEW] Traversal + Centrality (từ data/graph_query.go)
│       └── redis/           [NEW] Embedding cache
```

---

## 3. Routes Mới Thêm Vào router.go

```go
// internal/adapter/grpc/router.go — THÊM VÀO (không xóa routes cũ)

func RegisterRoutes(router *forward.Router, h *SearchHandler) {
    // ── Legacy routes (UNCHANGED) ──
    router.Handle("POST", "/v1/search", ...)         // cross-engine search
    router.Handle("POST", "/v1/search/rag", ...)     // RAG
    router.Handle("POST", "/v1/search/agents", ...)  // agent search
    // ... tất cả routes cũ

    // ── KGS Hybrid Search (NEW) ──
    // NOTE: /v1/search được dùng bởi cả legacy và KGS
    // Gateway phân biệt qua API key type:
    //   - JWT token → route đến legacy /v1/search (cross-engine)
    //   - kgs_ API key → route đến /v1/kgs/search
    router.Handle("POST", "/v1/kgs/search",          h.adapt(h.KGSHybridSearch))
    router.Handle("POST", "/v1/kgs/search/vector",   h.adapt(h.KGSVectorSearch))
    router.Handle("POST", "/v1/kgs/search/text",     h.adapt(h.KGSTextSearch))
    router.Handle("POST", "/v1/kgs/search/similar",  h.adapt(h.KGSSimilarNodes))

    // ── KGS Query Intelligence (NEW) ──
    router.Handle("GET",  "/v1/graph/nodes/{id}/context",  h.adapt(h.KGSGetContext))
    router.Handle("GET",  "/v1/graph/nodes/{id}/impact",   h.adapt(h.KGSGetImpact))
    router.Handle("GET",  "/v1/graph/nodes/{id}/coverage", h.adapt(h.KGSGetCoverage))
    router.Handle("POST", "/v1/graph/subgraph",             h.adapt(h.KGSGetSubgraph))
    router.Handle("POST", "/v1/query",                      h.adapt(h.KGSExecuteQuery))

    // ── KGS Analytics (NEW) ──
    router.Handle("GET",  "/v1/analytics/coverage-report",     h.adapt(h.KGSCoverageReport))
    router.Handle("GET",  "/v1/analytics/traceability-matrix", h.adapt(h.KGSTraceabilityMatrix))
    router.Handle("POST", "/v1/analytics/cluster",             h.adapt(h.KGSClusterAnalysis))

    // ── KGS View + Projection (NEW) ──
    router.Handle("POST", "/v1/views",          h.adapt(h.KGSCreateView))
    router.Handle("GET",  "/v1/views",          h.adapt(h.KGSListViews))
    router.Handle("GET",  "/v1/views/{id}",     h.adapt(h.KGSGetView))
    router.Handle("GET",  "/v1/views/{id}/query", h.adapt(h.KGSResolveView))
    router.Handle("DELETE", "/v1/views/{id}",   h.adapt(h.KGSDeleteView))
}
```

---

## 4. KGS Hybrid Search Implementation

```go
// internal/usecase/kgs/hybrid.go

type KGSHybridSearch struct {
    qdrant   QdrantClient
    pgSearch PGFullTextSearcher
    neo4j    CentralityScorer
    embedder EmbeddingProvider  // từ pkg/vectorstore/
    redis    *redis.Client       // embedding cache (1h TTL)
    defaults SearchDefaults
}

type SearchDefaults struct {
    TopK          int     // 10
    Alpha         float64 // 0.7 (vector weight)
    Beta          float64 // 0.2 (centrality weight)
    MinConfidence float64 // 0.3
}

func (s *KGSHybridSearch) Search(ctx context.Context, q *KGSSearchQuery) (*KGSSearchResult, error) {
    // Step 1: Get embedding (Redis cache-first)
    vector, err := s.getEmbedding(ctx, q.QueryText)
    if err != nil {
        return nil, fmt.Errorf("embed query: %w", err)
    }

    // Step 2: Parallel retrieval — tái sử dụng fan-out pattern từ SearchOrchestrator
    type rawResult struct {
        source string
        items  []SearchItem
        err    error
    }
    ch := make(chan rawResult, 2)
    
    go func() {
        items, err := s.qdrant.Search(ctx, q.AppID, vector, q.TopK*3, q.EntityTypes)
        ch <- rawResult{"vector", items, err}
    }()
    go func() {
        items, err := s.pgSearch.FullTextSearch(ctx, q.AppID, q.QueryText, q.TopK*3)
        ch <- rawResult{"text", items, err}
    }()

    vectorResults := (<-ch).items
    textResults   := (<-ch).items

    // Step 3: RRF Blend — tái sử dụng rrfMerge() từ orchestrator/search.go
    alpha := q.Alpha
    if alpha == 0 { alpha = s.defaults.Alpha }
    blended := weightedRRFMerge(vectorResults, textResults, alpha)

    // Step 4: Centrality re-ranking (Neo4j degree)
    if q.Beta > 0 {
        nodeIDs := extractIDs(blended)
        centrality := s.neo4j.GetDegrees(ctx, nodeIDs, q.AppID)
        blended = rerankWithCentrality(blended, centrality, q.Beta)
    }

    // Step 5: Filter + return
    return applyFiltersAndLimit(blended, q), nil
}

// weightedRRFMerge — extension của rrfMerge từ orchestrator/search.go
func weightedRRFMerge(vectorResults, textResults []SearchItem, alpha float64) []SearchItem {
    scores := make(map[string]float64)
    k := 60.0
    
    for rank, r := range vectorResults {
        scores[r.ID] += alpha * (1.0 / (k + float64(rank+1)))
    }
    for rank, r := range textResults {
        scores[r.ID] += (1-alpha) * (1.0 / (k + float64(rank+1)))
    }
    
    return sortByScore(scores, mergeItems(vectorResults, textResults))
}
```

---

## 5. Query Intelligence Implementation

```go
// internal/usecase/kgs/query_intel.go
// Tái sử dụng từ kgs-platform/internal/biz/query_planner.go + graph.go (read methods)

type QueryIntelUsecase struct {
    neo4j    Neo4jQuerier     // direct Neo4j traversal
    planner  *QueryPlanner    // Cypher query builder
    guardrails *Guardrails    // max depth, max nodes
}

func (uc *QueryIntelUsecase) GetContext(ctx context.Context, appID, tenantID, nodeID string, depth int, direction string) (map[string]any, error) {
    if err := uc.guardrails.ValidateDepth(depth); err != nil {
        return nil, err
    }
    cypher := uc.planner.BuildContextQuery(nodeID, direction)
    params := map[string]any{
        "app_id":    appID,
        "tenant_id": tenantID,
        "node_id":   nodeID,
        "depth":     depth,
    }
    return uc.neo4j.ExecuteQuery(ctx, cypher, params)
}

func (uc *QueryIntelUsecase) GetImpact(ctx context.Context, appID, tenantID, nodeID string, maxDepth int) (map[string]any, error) {
    if err := uc.guardrails.ValidateDepth(maxDepth); err != nil {
        return nil, err
    }
    cypher := uc.planner.BuildImpactQuery(nodeID, maxDepth)
    params := map[string]any{
        "app_id":    appID,
        "tenant_id": tenantID,
        "node_id":   nodeID,
    }
    return uc.neo4j.ExecuteQuery(ctx, cypher, params)
}
```

---

## 6. Embedding Cache (Redis)

```go
// internal/infra/redis/embed_cache.go

func (c *EmbedCache) Get(ctx context.Context, queryText string) ([]float32, bool) {
    key := fmt.Sprintf("embed:%x", sha256.Sum256([]byte(queryText)))
    val, err := c.redis.Get(ctx, key).Bytes()
    if err != nil {
        return nil, false
    }
    var vec []float32
    json.Unmarshal(val, &vec)
    return vec, true
}

func (c *EmbedCache) Set(ctx context.Context, queryText string, vec []float32) {
    key := fmt.Sprintf("embed:%x", sha256.Sum256([]byte(queryText)))
    data, _ := json.Marshal(vec)
    c.redis.Set(ctx, key, data, 1*time.Hour)
}
```

---

## 7. Effort Breakdown

| Task | Ngày |
|------|------|
| Qdrant + Neo4j + Redis infra clients | 1 |
| KGS hybrid search (tái sử dụng RRF từ orchestrator) | 1 |
| Query intelligence (copy từ biz/query_planner.go) | 0.5 |
| Analytics (copy từ internal/analytics/) | 0.5 |
| View/Projection engine (copy từ internal/projection/) | 0.5 |
| HTTP handlers + routing | 1 |
| main.go wiring | 0.25 |
| Tests | 0.25 |
| **Total** | **5 ngày** |
