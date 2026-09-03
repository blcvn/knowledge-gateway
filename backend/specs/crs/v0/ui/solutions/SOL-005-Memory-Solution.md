# SOL-005 — Solution: Memory Explorer API (CR-004)

| Field | Value |
|---|---|
| **Solution ID** | SOL-005 |
| **CR** | [CR-004 — Memory Explorer](../CR-004-MEMORY.md) |
| **Architecture ref** | §4.1 Engine Routes · §4.2 FEAT-007 · §6.3 Cross-Engine Recall Flow · §2.2 Services Inventory |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Implemented** | 2026-06-17 |

---

## 1. Phân tích kiến trúc

Memory search là use case cốt lõi nhất — sử dụng `vnp-search-hub` (§2.2 Platform services):

```
§6.3 Cross-Engine Recall Flow:
Client → POST /v1/memory/recall
    → vnp-search-hub
        → [parallel gRPC via bufconn]
            cognee-search + graphiti-search + memobase-context
            ov-search + zep-search + sm-search
        → Merge + rerank → Response
```

Console Memory Explorer (`FEAT-007`, §4.2) reuse `vnp-search-hub` nhưng thêm:
- Console-specific metadata (policyTags, versionChain, score breakdown)
- Facets aggregation (byEngine, byType)
- Navigation: neighbors, versions

---

## 2. Giải pháp Backend

### 2.1 Handler (`console_memory_handler.go`)

```go
type ConsoleMemoryHandler struct {
    searchHub  VNPSearchHubClient  // gRPC → vnp-search-hub (parallel fan-out)
    graphiti   GraphitiStoreClient // gRPC → graphiti-store (entity/edge detail)
    smMemory   SMMemoryClient      // gRPC → sm-memory (version chain)
    cognee     CogneeSearchClient  // gRPC → cognee-search (neighbors)
}
```

### 2.2 Memory Search — via vnp-search-hub

```go
// POST /v1/console/memory/search
func (h *ConsoleMemoryHandler) Search(w http.ResponseWriter, r *http.Request) {
    var req MemorySearchRequest
    json.NewDecoder(r.Body).Decode(&req)
    tenantID := authctx.TenantID(r.Context())

    // Gọi vnp-search-hub (đã có parallel fan-out logic §6.3)
    hubResp, err := h.searchHub.CrossEngineSearch(r.Context(), &searchhub.SearchRequest{
        TenantID:  tenantID,
        Query:     req.Query,
        Mode:      req.Mode,           // semantic|bm25|hybrid|graph
        Engines:   req.Engines,        // filter engines
        Limit:     int32(req.Limit),
        Offset:    int32(req.Offset),
        Reranking: req.Reranking,
    })

    // Map hub response → MemorySearchResult (frontend type)
    results := make([]MemoryItem, len(hubResp.Results))
    for i, r := range hubResp.Results {
        results[i] = MemoryItem{
            ID:               fmt.Sprintf("%s:%s", r.Engine, r.LocalID),
            Engine:           r.Engine,
            MemoryType:       r.MemoryType,
            Title:            r.Title,
            Summary:          r.Summary,
            Content:          r.Content,
            Score:            r.Score,
            Entities:         r.Entities,
            SourceSessions:   r.SourceSessions,
            TemporalValidity: TemporalValidity{From: r.ValidFrom, To: r.ValidTo},
            PolicyTags:       r.PolicyTags,
            VersionChain:     r.VersionChain,
            Metadata:         r.Metadata,
        }
    }

    // Compute facets
    facets := computeFacets(results)

    httputil.JSON(w, 200, MemorySearchResult{
        Results:   results,
        Total:     int(hubResp.Total),
        Facets:    facets,
        LatencyMs: hubResp.LatencyMs,
    })
}
```

**JSON camelCase** (phải khớp `MemoryItem` TypeScript):
```go
type MemoryItem struct {
    ID               string            `json:"id"`
    Engine           string            `json:"engine"`
    MemoryType       string            `json:"memoryType"`
    Title            string            `json:"title"`
    Summary          string            `json:"summary"`
    Content          string            `json:"content"`
    Score            float64           `json:"score"`
    Entities         []string          `json:"entities"`
    SourceSessions   []string          `json:"sourceSessions"`
    TemporalValidity TemporalValidity  `json:"temporalValidity"`
    PolicyTags       []string          `json:"policyTags"`
    VersionChain     *string           `json:"versionChain"`
    Metadata         map[string]any    `json:"metadata"`
}
```

### 2.3 Memory Detail — Engine routing by ID prefix

Memory ID format: `{engine}:{local_id}` (e.g. `graphiti:ep_abc123`, `memobase:prof_xyz`)

```go
// GET /v1/console/memory/{id}
// Path: /v1/console/memory/graphiti:ep_abc123
func (h *ConsoleMemoryHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
    rawID := r.PathValue("id")  // URL-decoded by stdlib
    parts := strings.SplitN(rawID, ":", 2)
    engine, localID := parts[0], parts[1]
    tenantID := authctx.TenantID(r.Context())

    var item *MemoryItem
    switch engine {
    case "graphiti":
        item, _ = h.fetchFromGraphiti(r.Context(), localID, tenantID)
    case "memobase":
        item, _ = h.fetchFromMemobase(r.Context(), localID, tenantID)
    case "sm", "supermemory":
        item, _ = h.fetchFromSupermemory(r.Context(), localID, tenantID)
    case "zep":
        item, _ = h.fetchFromZep(r.Context(), localID, tenantID)
    case "cognee":
        item, _ = h.fetchFromCognee(r.Context(), localID, tenantID)
    case "ov", "openviking":
        item, _ = h.fetchFromOpenViking(r.Context(), localID, tenantID)
    default:
        httputil.NotFound(w); return
    }

    httputil.JSON(w, 200, item)
}
```

Mỗi engine route đến service tương ứng qua gRPC bufconn:
- `graphiti:` → `graphiti-store.GetEpisode()`
- `memobase:` → `memobase-engine.GetProfile()`
- `sm:` → `sm-memory.GetMemory()`
- `cognee:` → `cognee-search.GetNode()`

### 2.4 Memory Neighbors — Semantic/Graph/Temporal

```go
// GET /v1/console/memory/{id}/neighbors?strategy=semantic&limit=10
func (h *ConsoleMemoryHandler) GetNeighbors(w http.ResponseWriter, r *http.Request) {
    rawID := r.PathValue("id")
    strategy := r.URL.Query().Get("strategy") // semantic | graph | temporal
    limit := queryInt(r, "limit", 10)
    engine, localID := parseMemoryID(rawID)

    switch strategy {
    case "graph":
        // Neo4j 1-hop neighbors qua graphiti-store
        neighbors, _ := h.graphiti.GetNeighbors(r.Context(), localID, int32(limit))
        httputil.JSON(w, 200, mapToSearchResult(neighbors))

    case "temporal":
        // Memories trong cùng time window từ vnp-event
        // ...

    default: // "semantic"
        // Vector similarity từ vnp-search-hub với embedding của item hiện tại
        item, _ := h.fetchItem(r.Context(), engine, localID, ...)
        neighbors, _ := h.searchHub.SimilarByEmbedding(r.Context(), item.Embedding, limit)
        httputil.JSON(w, 200, mapToSearchResult(neighbors))
    }
}
```

### 2.5 Memory Versions — sm-memory version chain

```go
// GET /v1/console/memory/{id}/versions
// Chỉ có Supermemory (sm:*) hỗ trợ versioning
func (h *ConsoleMemoryHandler) GetVersions(w http.ResponseWriter, r *http.Request) {
    rawID := r.PathValue("id")
    engine, localID := parseMemoryID(rawID)

    if engine != "sm" && engine != "supermemory" {
        // Các engine khác không có version chain
        httputil.JSON(w, 200, []MemoryVersion{}); return
    }

    // sm-memory service có GetVersionChain
    chain, err := h.smMemory.GetVersionChain(r.Context(), &sm.GetVersionChainRequest{
        MemoryId: localID,
        TenantID: authctx.TenantID(r.Context()),
    })

    versions := make([]MemoryVersion, len(chain.Versions))
    for i, v := range chain.Versions {
        versions[i] = MemoryVersion{
            ID:            v.Id,
            MemoryID:      rawID,
            Content:       v.Content,
            VersionNumber: int(v.VersionNumber),
            IsLatest:      v.IsLatest,
            Diff:          v.Diff,
            CreatedAt:     v.CreatedAt,
        }
    }
    httputil.JSON(w, 200, versions)
}
```

---

## 3. Đặc điểm kỹ thuật quan trọng

### 3.1 URL encoding

Memory ID có format `engine:local_id` phải được URL-encode khi dùng trong path:
```
/v1/console/memory/graphiti%3Aep_abc123
```

Frontend phải dùng `encodeURIComponent(id)` khi tạo URL.

### 3.2 Timeout per engine

Cross-engine search qua vnp-search-hub: mỗi engine có timeout riêng 2s, kết quả từ engine nào respond kịp thì merge vào:

```go
ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
defer cancel()
// vnp-search-hub internal: per-engine timeout 2s
```

### 3.3 Tenant isolation

Mọi query đều inject `tenantID` vào gRPC request để đảm bảo isolation (§3.1 Gateway Domain: TenantID trong AuthContext).
