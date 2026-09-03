# TASK-UI-001 — Graph Studio HTTP Handlers

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-001 |
| **Wave** | 1 (Backend API) |
| **Solution** | [SOL-UI-001](../solutions/SOL-UI-001-Graph-Studio.md) §2.1 |
| **Component** | `gateway/adapter/handler/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-UI-002, TASK-UI-003 |
| **Estimated** | 4h |

---

## Mục tiêu

Implement 6 Graph Studio HTTP handlers trong gateway. Proxy tới `graphiti-store` gRPC service. Enforce result cap (200 nodes / 500 edges) và tenant isolation.

---

## Công việc cụ thể

### 1. Tạo `gateway/adapter/handler/graph_handler.go` [NEW]

Implement tất cả 6 handlers. Full implementation xem [SOL-UI-001 §2.1](../solutions/SOL-UI-001-Graph-Studio.md).

**Handler signatures:**

```go
type GraphHandler struct {
    graphitiStore port.GraphitiStoreClient
    cypherVal     *usecase.CypherValidator
    logger        *slog.Logger
}

// POST /v1/console/graph/subgraph
// Body: {entity_id, depth, entity_types, relationship_types, time_range}
// Returns: {nodes: [], edges: [], truncated: bool}
func (h *GraphHandler) Subgraph(w http.ResponseWriter, r *http.Request)

// GET /v1/console/graph/entity/{id}
// Returns: {id, type, properties, edges, temporal_validity}
func (h *GraphHandler) GetEntity(w http.ResponseWriter, r *http.Request)

// POST /v1/console/graph/timeline
// Body: {entity_id, from, to}
// Returns: timeline of graph changes
func (h *GraphHandler) Timeline(w http.ResponseWriter, r *http.Request)

// GET /v1/console/graph/ontology
// Returns: {entity_types, relationship_types, property_constraints}
func (h *GraphHandler) GetOntology(w http.ResponseWriter, r *http.Request)

// PUT /v1/console/graph/ontology
// Body: OntologySchema — updates entity/relationship types for future extractions
func (h *GraphHandler) UpdateOntology(w http.ResponseWriter, r *http.Request)

// POST /v1/console/graph/query
// Body: {query: "MATCH (n)..."}
// Returns: {rows: [], nodes: [], truncated: bool}
func (h *GraphHandler) QueryGraph(w http.ResponseWriter, r *http.Request)
```

**Key constraints to enforce:**

```go
// In Subgraph handler:
const MaxNodes = 200
const MaxEdges = 500

resp, err := h.graphitiStore.QuerySubgraph(r.Context(), &graphitipb.SubgraphRequest{
    TenantId: auth.TenantID,    // ← always inject tenant_id
    EntityId: req.EntityID,
    Depth:    clamp(req.Depth, 1, 5),  // max depth = 5
    MaxNodes: MaxNodes,
    MaxEdges: MaxEdges,
})

// Return truncated flag when results are capped
writeJSON(w, http.StatusOK, map[string]interface{}{
    "nodes":     resp.Nodes,
    "edges":     resp.Edges,
    "truncated": resp.Truncated,  // true if > MaxNodes/MaxEdges
})

// In QueryGraph handler:
if err := h.cypherVal.Validate(req.Query); err != nil {
    writeError(w, http.StatusForbidden, "write_not_allowed", err.Error())
    return
}
```

### 2. Tạo `gateway/adapter/client/graphiti_store_client.go` [NEW or VERIFY EXISTS]

```go
package client

type GraphitiStoreClient struct {
    conn   *grpc.ClientConn
    client graphitipb.GraphitiStoreServiceClient
}

func (c *GraphitiStoreClient) QuerySubgraph(ctx context.Context, req *graphitipb.SubgraphRequest) (*graphitipb.SubgraphResponse, error) {
    ctx = middleware.PropagateToGRPC(ctx) // inject trace context
    return c.client.QuerySubgraph(ctx, req)
}

// Implement all 6 gRPC methods:
// QuerySubgraph, GetEntity, GetTemporalSubgraph, GetOntology, UpdateOntology, ExecuteCypher
```

---

## Acceptance Criteria

- [ ] `POST /v1/console/graph/subgraph` returns `{nodes, edges, truncated}` with max 200/500
- [ ] `GET /v1/console/graph/entity/{id}` returns entity with all properties + temporal edges
- [ ] `POST /v1/console/graph/timeline` returns ordered list of graph changes
- [ ] `GET /v1/console/graph/ontology` returns current entity + relationship type schema
- [ ] `PUT /v1/console/graph/ontology` updates schema (affects future LLM extractions)
- [ ] `POST /v1/console/graph/query` executes Cypher, returns rows + optional graph viz
- [ ] All handlers inject `tenant_id` from AuthContext (never from request body)
- [ ] `truncated: true` returned when result exceeds caps
- [ ] `go build ./gateway/...` passes

## Files

```
gateway/adapter/handler/graph_handler.go          [NEW]
gateway/adapter/client/graphiti_store_client.go   [NEW or VERIFY]
```
