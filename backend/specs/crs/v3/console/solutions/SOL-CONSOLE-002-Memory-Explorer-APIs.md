# Solution: SOL-CONSOLE-002 — Memory Explorer Backend APIs

**CR:** CR-CONSOLE-002
**TDD refs:** `architecture/08-platform-services.md §VNP Search Hub`, `architecture/03-graphiti-services.md`
**Version:** v3/console

---

## 1. Architecture

Memory Explorer needs 4 endpoints, each served by a different backend service:
- `/search` → `vnp-search-hub` (RRF cross-engine, already implemented CR-CORE-002)
- `/memory/{id}` → engine-specific service (route by engine type)
- `/memory/{id}/neighbors` → `graphiti-search.GetSubgraph(seed=id, depth=1)`
- `/memory/{id}/versions` → `sm-memory` (version history)

---

## 2. Implementation

```go
// gateway/adapter/handler/memory_explorer_handler.go [NEW]
type MemoryExplorerHandler struct {
    registry   port.GRPCRegistry
    searchHub  vnpsearchpb.VnpSearchHubServiceClient
}

// POST /v1/console/memory/search
func (h *MemoryExplorerHandler) Search(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Query     string   `json:"query"`
        Types     []string `json:"types,omitempty"`
        UserID    string   `json:"user_id,omitempty"`
        TimeRange *struct {
            From string `json:"from"`
            To   string `json:"to"`
        } `json:"time_range,omitempty"`
        Limit  int `json:"limit"`
        Offset int `json:"offset"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    if req.Limit == 0 { req.Limit = 20 }
    tenantID := tenant.FromContext(r.Context())

    grpcReq := &vnpsearchpb.RecallRequest{
        TenantId: tenantID, UserId: req.UserID,
        Query: req.Query, Types: req.Types, MaxResults: int32(req.Limit),
    }
    if req.TimeRange != nil {
        grpcReq.TimeRangeFrom = req.TimeRange.From
        grpcReq.TimeRangeTo   = req.TimeRange.To
    }

    resp, err := h.searchHub.Recall(r.Context(), grpcReq)
    if err != nil { writeError(w, 500, "search_failed", err.Error()); return }
    writeJSON(w, 200, map[string]any{
        "results": resp.Results, "total_hits": resp.TotalHits,
        "engines_queried": resp.EnginesUsed,
    })
}

// GET /v1/console/memory/{id}?engine=graphiti
func (h *MemoryExplorerHandler) GetMemory(w http.ResponseWriter, r *http.Request) {
    memID  := chi.URLParam(r, "id")
    engine := r.URL.Query().Get("engine")  // required to route to correct service
    tenantID := tenant.FromContext(r.Context())

    if engine == "" {
        writeError(w, 400, "missing_engine", "engine param required (graphiti|cognee|zep|...)")
        return
    }

    // Route to engine-specific search service
    svcName := engineSearchService(engine) // "graphiti-search", "cognee-search", etc.
    conn, err := h.registry.Get(svcName)
    if err != nil { writeError(w, 503, "engine_unavailable", ""); return }

    client := memorypb.NewMemoryServiceClient(conn)
    resp, err := client.GetMemory(r.Context(), &memorypb.GetMemoryRequest{
        Id: memID, TenantId: tenantID,
    })
    if err != nil { writeError(w, 404, "not_found", ""); return }
    writeJSON(w, 200, resp)
}

// GET /v1/console/memory/{id}/neighbors
func (h *MemoryExplorerHandler) GetNeighbors(w http.ResponseWriter, r *http.Request) {
    memID := chi.URLParam(r, "id")
    tenantID := tenant.FromContext(r.Context())

    conn, _ := h.registry.Get("graphiti-search")
    client := graphpb.NewGraphitiSearchServiceClient(conn)
    resp, err := client.GetSubgraph(r.Context(), &graphpb.SubgraphRequest{
        TenantId: tenantID, SeedEntityId: memID, MaxDepth: 1,
    })
    if err != nil { writeError(w, 500, "neighbors_failed", err.Error()); return }
    writeJSON(w, 200, resp)
}

// GET /v1/console/memory/{id}/versions
func (h *MemoryExplorerHandler) GetVersions(w http.ResponseWriter, r *http.Request) {
    memID := chi.URLParam(r, "id")
    tenantID := tenant.FromContext(r.Context())

    conn, _ := h.registry.Get("sm-memory")
    client := smpb.NewSMMemoryServiceClient(conn)
    resp, err := client.GetVersionHistory(r.Context(), &smpb.VersionRequest{
        MemoryId: memID, TenantId: tenantID,
    })
    if err != nil { writeError(w, 404, "versions_not_found", ""); return }
    writeJSON(w, 200, resp)
}

func engineSearchService(engine string) string {
    table := map[string]string{
        "graphiti": "graphiti-search", "cognee": "cognee-search",
        "zep": "zep-search", "memobase": "memobase-context",
        "openviking": "ov-search", "supermemory": "sm-search",
    }
    return table[engine]
}
```

---

## 3. File Changes

| File | Action |
|---|---|
| `gateway/adapter/handler/memory_explorer_handler.go` | **[NEW]** |
| `gateway/adapter/handler/router.go` | **[MODIFY]** `/v1/console/memory/*` routes |
