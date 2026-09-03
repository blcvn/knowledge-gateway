# TASK-BE-006 — Console Memory Handler: search / get / neighbors / versions

| Field | Value |
|---|---|
| **Task ID** | TASK-BE-006 |
| **Layer** | Backend — Go |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-005](../solutions/SOL-005-Memory-Solution.md) |
| **Priority** | 🔴 P0 |
| **Depends On** | — |
| **Estimated** | 3h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `gateway/internal/adapter/handler/console_memory_handler.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

```go
package handler

type ConsoleMemoryHandler struct {
    searchHub VNPSearchHubClient  // gRPC bufconn → vnp-search-hub
    graphiti  GraphitiClient
    memobase  MemobaseEngineClient
    zep       ZepMemoryClient
    cognee    CogneeClient
    sm        SMMemoryClient      // Supermemory
    ov        OpenVikingClient
}

// POST /v1/console/memory/search
func (h *ConsoleMemoryHandler) Search(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Query   string   `json:"query"`
        Mode    string   `json:"mode"`    // hybrid | semantic | keyword
        Engines []string `json:"engines"`
        Limit   int      `json:"limit"`
        Offset  int      `json:"offset"`
        Reranking string `json:"reranking"` // cross_encoder | rrf | none
    }
    json.NewDecoder(r.Body).Decode(&req)

    tenantID := authctx.TenantID(r.Context())

    // Fan-out to vnp-search-hub (internal router dispatches to each engine)
    hubResp, err := h.searchHub.CrossEngineSearch(r.Context(), &searchhub.SearchRequest{
        TenantID: tenantID,
        Query:    req.Query,
        Mode:     req.Mode,
        Engines:  req.Engines,
        Limit:    int32(req.Limit),
        Reranking: req.Reranking,
    })
    if err != nil {
        httputil.Error(w, "Search failed", "SEARCH_ERROR", 500)
        return
    }

    // Map gRPC results + compute facets
    results := []map[string]any{}
    engineCount := map[string]int{}
    typeCount   := map[string]int{}
    for _, r := range hubResp.Results {
        results = append(results, map[string]any{
            "id":      r.ID,
            "content": r.Content,
            "engine":  r.Engine,
            "type":    r.Type,
            "score":   r.Score,
        })
        engineCount[r.Engine]++
        typeCount[r.Type]++
    }

    httputil.JSON(w, 200, map[string]any{
        "results":   results,
        "total":     hubResp.Total,
        "latencyMs": hubResp.LatencyMs,
        "facets": map[string]any{
            "byEngine": engineCount,
            "byType":   typeCount,
        },
    })
}

// GET /v1/console/memory/{id}
// ID format: "graphiti:ep_abc123", "sm:mem_xyz", "memobase:prof_123"
func (h *ConsoleMemoryHandler) Get(w http.ResponseWriter, r *http.Request) {
    id, _ := url.PathUnescape(r.PathValue("id"))  // URL-decode "graphiti%3Aep_abc"

    parts := strings.SplitN(id, ":", 2)
    if len(parts) != 2 {
        httputil.Error(w, "Invalid memory ID format", "INVALID_ID", 400)
        return
    }
    engine, localID := parts[0], parts[1]
    tenantID := authctx.TenantID(r.Context())

    var item any
    var err error
    switch engine {
    case "graphiti":
        item, err = h.graphiti.GetEpisode(r.Context(), localID, tenantID)
    case "sm", "supermemory":
        item, err = h.sm.GetMemory(r.Context(), localID, tenantID)
    case "memobase":
        item, err = h.memobase.GetProfile(r.Context(), localID, tenantID)
    case "zep":
        item, err = h.zep.GetFact(r.Context(), localID, tenantID)
    case "cognee":
        item, err = h.cognee.GetNode(r.Context(), localID, tenantID)
    default:
        httputil.Error(w, "Unknown engine: "+engine, "UNKNOWN_ENGINE", 404)
        return
    }
    if err != nil {
        httputil.Error(w, "Memory not found", "NOT_FOUND", 404)
        return
    }
    httputil.JSON(w, 200, item)
}

// GET /v1/console/memory/{id}/neighbors
func (h *ConsoleMemoryHandler) Neighbors(w http.ResponseWriter, r *http.Request) {
    id, _ := url.PathUnescape(r.PathValue("id"))
    strategy := r.URL.Query().Get("strategy")
    if strategy == "" { strategy = "semantic" }

    // Route to search hub's neighbor search
    results, _ := h.searchHub.NeighborSearch(r.Context(), &searchhub.NeighborRequest{
        ReferenceID: id,
        Strategy:    strategy,
        Limit:       10,
        TenantID:    authctx.TenantID(r.Context()),
    })
    httputil.JSON(w, 200, results)
}

// GET /v1/console/memory/{id}/versions
// Chỉ hoạt động với Supermemory engine (ID bắt đầu bằng "sm:")
func (h *ConsoleMemoryHandler) Versions(w http.ResponseWriter, r *http.Request) {
    id, _ := url.PathUnescape(r.PathValue("id"))
    if !strings.HasPrefix(id, "sm:") && !strings.HasPrefix(id, "supermemory:") {
        httputil.Error(w, "Version history only available for Supermemory", "NOT_SUPPORTED", 400)
        return
    }
    localID := strings.SplitN(id, ":", 2)[1]
    versions, _ := h.sm.GetVersionChain(r.Context(), localID, authctx.TenantID(r.Context()))
    httputil.JSON(w, 200, versions)
}
```

### Routes

```go
mux.HandleFunc("POST /v1/console/memory/search",           authMiddleware(mem.Search))
mux.HandleFunc("GET /v1/console/memory/{id}",              authMiddleware(mem.Get))
mux.HandleFunc("GET /v1/console/memory/{id}/neighbors",    authMiddleware(mem.Neighbors))
mux.HandleFunc("GET /v1/console/memory/{id}/versions",     authMiddleware(mem.Versions))
```

---

## Verification

```bash
curl -X POST http://localhost:8080/v1/console/memory/search \
  -H "Authorization: Bearer <token>" -H "x-tenant-id: <tid>" \
  -d '{"query":"user prefers dark mode","mode":"hybrid","engines":["memobase","graphiti"],"limit":10}'
# Expected: {"results":[...],"total":N,"latencyMs":120,"facets":{...}}
```
