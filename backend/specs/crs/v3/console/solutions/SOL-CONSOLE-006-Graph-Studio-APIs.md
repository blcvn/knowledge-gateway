# Solution: SOL-CONSOLE-006 — Graph Studio Backend Implementation

**CR:** CR-CONSOLE-006
**TDD refs:** `architecture/03-graphiti-services.md`, `architecture/01-gateway.md §3`
**Version:** v3/console
**Depends on:** CR-UI-001, SOL-UI-001

---

## 1. Architecture

Graph Studio routes through Gateway → `graphiti-search` gRPC service.
Needs 2 new gRPC methods added to `graphiti-search`: `GetSubgraph`, `RunCypher`.

---

## 2. New gRPC Methods (graphiti-search)

```protobuf
// shared/proto/graphiti/v1/graphiti_search.proto [MODIFY]
message SubgraphRequest {
    string tenant_id    = 1;
    string seed_entity_id = 2;
    int32  max_depth    = 3;
    repeated string filter_types = 4;
}

message SubgraphResponse {
    repeated GraphNode nodes = 1;
    repeated GraphEdge edges = 2;
    bool truncated = 3;  // true if result capped
}

message GraphNode {
    string id = 1; string name = 2; string type = 3;
    map<string, string> properties = 4;
}

message GraphEdge {
    string id = 1; string source_id = 2; string target_id = 3;
    string relationship = 4; string valid_from = 5; string valid_to = 6;
}

message CypherRequest {
    string tenant_id = 1; string query = 2;
}
message CypherResponse {
    repeated string columns = 1;
    repeated google.protobuf.Struct rows = 2;
    int64 execution_time_ms = 3;
}

rpc GetSubgraph(SubgraphRequest) returns (SubgraphResponse);
rpc RunCypher(CypherRequest)   returns (CypherResponse);
rpc GetEntityDetail(EntityDetailRequest) returns (GraphNode);
rpc GetTimeline(TimelineRequest) returns (TimelineResponse);
```

---

## 3. Gateway GraphHandler

```go
// gateway/adapter/handler/graph_handler.go [NEW]
type GraphHandler struct {
    registry port.GRPCRegistry
}

func (h *GraphHandler) graphClient() graphpb.GraphitiSearchServiceClient {
    conn, _ := h.registry.Get("graphiti-search")
    return graphpb.NewGraphitiSearchServiceClient(conn)
}

// POST /v1/console/graph/subgraph
func (h *GraphHandler) GetSubgraph(w http.ResponseWriter, r *http.Request) {
    var req struct {
        SeedEntityID string   `json:"seed_entity_id"`
        Depth        int      `json:"depth"`
        FilterTypes  []string `json:"filter_types,omitempty"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    if req.Depth == 0 { req.Depth = 3 }
    if req.Depth > 5  { req.Depth = 5 }

    resp, err := h.graphClient().GetSubgraph(r.Context(), &graphpb.SubgraphRequest{
        TenantId:     tenant.FromContext(r.Context()),
        SeedEntityId: req.SeedEntityID,
        MaxDepth:     int32(req.Depth),
        FilterTypes:  req.FilterTypes,
    })
    if err != nil { writeError(w, 500, "subgraph_failed", err.Error()); return }
    writeJSON(w, 200, resp)
}

// GET /v1/console/graph/entity/{id}
func (h *GraphHandler) GetEntity(w http.ResponseWriter, r *http.Request) {
    entityID := chi.URLParam(r, "id")
    resp, err := h.graphClient().GetEntityDetail(r.Context(), &graphpb.EntityDetailRequest{
        TenantId: tenant.FromContext(r.Context()), EntityId: entityID,
    })
    if err != nil { writeError(w, 404, "entity_not_found", ""); return }
    writeJSON(w, 200, resp)
}

// POST /v1/console/graph/timeline
func (h *GraphHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
    var req struct {
        EntityID  string `json:"entity_id"`
        TimeRange struct{ From, To string } `json:"time_range"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    resp, err := h.graphClient().GetTimeline(r.Context(), &graphpb.TimelineRequest{
        TenantId: tenant.FromContext(r.Context()),
        EntityId: req.EntityID,
        From: req.TimeRange.From, To: req.TimeRange.To,
    })
    if err != nil { writeError(w, 500, "timeline_failed", err.Error()); return }
    writeJSON(w, 200, resp)
}

// GET /v1/console/graph/ontology
func (h *GraphHandler) GetOntology(w http.ResponseWriter, r *http.Request) {
    resp, err := h.graphClient().GetOntology(r.Context(), &graphpb.OntologyRequest{
        TenantId: tenant.FromContext(r.Context()),
    })
    if err != nil { writeError(w, 500, "ontology_failed", err.Error()); return }
    writeJSON(w, 200, resp)
}

// PUT /v1/console/graph/ontology
func (h *GraphHandler) UpdateOntology(w http.ResponseWriter, r *http.Request) {
    var req graphpb.OntologyUpdateRequest
    json.NewDecoder(r.Body).Decode(&req)
    req.TenantId = tenant.FromContext(r.Context())
    _, err := h.graphClient().UpdateOntology(r.Context(), &req)
    if err != nil { writeError(w, 500, "update_failed", err.Error()); return }
    writeJSON(w, 200, map[string]bool{"updated": true})
}

// POST /v1/console/graph/query — Cypher read-only
func (h *GraphHandler) RunQuery(w http.ResponseWriter, r *http.Request) {
    var req struct{ Query string `json:"query"`; Engine string `json:"engine"` }
    json.NewDecoder(r.Body).Decode(&req)

    // Security: block write operations
    lower := strings.ToLower(strings.TrimSpace(req.Query))
    for _, op := range []string{"create ", "delete ", "merge ", "set ", " remove ", "drop "} {
        if strings.Contains(lower, op) {
            writeError(w, 403, "write_blocked", "only MATCH/RETURN queries allowed")
            return
        }
    }

    // Inject tenant filter
    tenantID := tenant.FromContext(r.Context())
    if !strings.Contains(lower, "tenant_id") {
        // Wrap in WITH clause to inject tenant filter
        req.Query = injectTenantFilter(req.Query, tenantID)
    }

    resp, err := h.graphClient().RunCypher(r.Context(), &graphpb.CypherRequest{
        TenantId: tenantID, Query: req.Query,
    })
    if err != nil { writeError(w, 500, "query_failed", err.Error()); return }
    writeJSON(w, 200, resp)
}

func injectTenantFilter(query, tenantID string) string {
    // Simple heuristic: if query has WHERE, append AND; else add WHERE
    lower := strings.ToLower(query)
    if strings.Contains(lower, "where") {
        return query + fmt.Sprintf(` AND n.tenant_id = "%s"`, tenantID)
    }
    return query + fmt.Sprintf(` WHERE n.tenant_id = "%s"`, tenantID)
}
```

---

## 4. File Changes

| File | Action |
|---|---|
| `shared/proto/graphiti/v1/graphiti_search.proto` | **[MODIFY]** add GetSubgraph, RunCypher, GetTimeline |
| `backend/api/proto/graphiti/v1/*_grpc.pb.go` | **[GENERATED]** |
| `services/graphiti-search/internal/adapter/grpc/graph_server.go` | **[MODIFY]** implement new methods |
| `gateway/adapter/handler/graph_handler.go` | **[NEW]** |
| `gateway/adapter/handler/router.go` | **[MODIFY]** `/v1/console/graph/*` routes |
