# Change Request: CR-CONSOLE-006 — Graph Studio Backend Implementation

**CR ID:** CR-CONSOLE-006
**Component:** `backend/gateway/adapter/handler/graph_handler.go`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Console
**Feature:** [F17](../../../features/17-graph-studio/README.md)
**Depends On:** CR-UI-001

---

## 1. Mô tả

Backend implementation cho Graph Studio APIs. CR-UI-001 định nghĩa spec. CR này định nghĩa implementation chi tiết.

---

## 2. `gateway/adapter/handler/graph_handler.go` [NEW]

```go
// POST /v1/console/graph/subgraph
func (h *GraphHandler) GetSubgraph(w http.ResponseWriter, r *http.Request) {
    var req SubgraphRequest
    json.NewDecoder(r.Body).Decode(&req)
    if req.Depth == 0 { req.Depth = 3 }
    if req.Depth > 5 { req.Depth = 5 }

    conn := h.registry.Get("graphiti-search")
    client := graphpb.NewGraphitiSearchServiceClient(conn)
    resp, err := client.GetSubgraph(r.Context(), &graphpb.SubgraphRequest{
        TenantId: tenant.FromContext(r.Context()),
        SeedEntityId: req.SeedEntityID,
        MaxDepth: int32(req.Depth),
        FilterTypes: req.FilterTypes,
    })
    if err != nil { writeError(w, 500, "graph_error", err.Error()); return }

    // Limit result size
    if len(resp.Nodes) > 200 {
        resp.Nodes = resp.Nodes[:200]
        resp.Truncated = true
    }
    writeJSON(w, 200, resp)
}

// POST /v1/console/graph/query
func (h *GraphHandler) RunQuery(w http.ResponseWriter, r *http.Request) {
    var req QueryRequest
    json.NewDecoder(r.Body).Decode(&req)

    // Security: block write operations
    lower := strings.ToLower(req.Query)
    for _, op := range []string{"create", "delete", "merge", "set", "remove", "drop"} {
        if strings.Contains(lower, op) {
            writeError(w, 403, "write_blocked", "only read queries allowed (MATCH/RETURN)")
            return
        }
    }

    // Inject tenant_id
    query := injectTenantFilter(req.Query, tenant.FromContext(r.Context()))

    conn := h.registry.Get("graphiti-search")
    client := graphpb.NewGraphitiSearchServiceClient(conn)
    resp, err := client.RunCypher(r.Context(), &graphpb.CypherRequest{Query: query})
    if err != nil { writeError(w, 500, "query_failed", err.Error()); return }
    writeJSON(w, 200, resp)
}
```

---

## 3. New gRPC methods needed in `graphiti-search`

```protobuf
rpc GetSubgraph(SubgraphRequest) returns (SubgraphResponse);
rpc RunCypher(CypherRequest) returns (CypherResponse);
rpc GetEntityDetail(EntityDetailRequest) returns (EntityDetailResponse);
rpc GetTimeline(TimelineRequest) returns (TimelineResponse);
```

---

## 4. Acceptance Criteria

- [ ] Subgraph capped at 200 nodes (Truncated flag if more)
- [ ] Cypher MATCH/RETURN allowed, write operations → 403
- [ ] All queries inject tenant_id WHERE clause
- [ ] Entity detail includes temporal edge validity
- [ ] Timeline returns events in chronological order
