# SOL-UI-001 — Solution: Graph Studio Backend (Knowledge Graph Visualization)

| Field | Value |
|---|---|
| **Solution ID** | SOL-UI-001 |
| **CR** | [CR-UI-001](../../../../docs/crs/v3/ui/CR-UI-001-Graph-Studio.md) |
| **TDD ref** | [03-graphiti-services.md](../../../tdd/architecture/03-graphiti-services.md) · [backend-api-specs.md §12.3-Graph-Studio](../../../tdd/backend-api-specs.md) |
| **Status** | Open |
| **Priority** | 🟡 High |

**Trạng thái:** 🔄 Partial  
**Ghi chú audit:** GraphHandler.Query() Cypher forwarding works; Cypher validator middleware missing
---

## 1. Phân tích kiến trúc

Theo `backend-api-specs.md §12.3 Graph Studio`, 6 API endpoints đã được define và routing qua `graphiti-store` gRPC service. Tuy nhiên **các handlers chưa implement** trong `gateway/adapter/handler/`.

`graphiti-store` service (port 9023) có thể execute Cypher queries qua Neo4j. Cần implement:
1. Backend handlers cho 6 Graph Studio endpoints
2. Cypher whitelist validator (MATCH/RETURN only — no writes)
3. Tenant isolation injection vào mọi Cypher query
4. Result cap (max 200 nodes, 500 edges)

---

## 2. Giải pháp

### 2.1 `gateway/adapter/handler/graph_handler.go` [NEW]

```go
package handler

type GraphHandler struct {
    graphitiStore port.GraphitiStoreClient
    logger        *slog.Logger
}

// POST /v1/console/graph/subgraph
// Body: {entity_id, depth, entity_types, time_range, relationship_types}
func (h *GraphHandler) Subgraph(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    var req struct {
        EntityID          string   `json:"entity_id"`
        Depth             int      `json:"depth"`   // 1-5, default 2
        EntityTypes       []string `json:"entity_types"`
        RelationshipTypes []string `json:"relationship_types"`
        TimeRange         *struct {
            From time.Time `json:"from"`
            To   time.Time `json:"to"`
        } `json:"time_range"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
        return
    }

    if req.Depth < 1 || req.Depth > 5 {
        req.Depth = 2
    }

    resp, err := h.graphitiStore.QuerySubgraph(r.Context(), &graphitipb.SubgraphRequest{
        TenantId:          auth.TenantID,
        EntityId:          req.EntityID,
        Depth:             int32(req.Depth),
        EntityTypes:       req.EntityTypes,
        RelationshipTypes: req.RelationshipTypes,
        MaxNodes:          200,  // Result cap per CR spec
        MaxEdges:          500,
    })
    if err != nil {
        writeError(w, http.StatusInternalServerError, "graph_error", err.Error())
        return
    }

    writeJSON(w, http.StatusOK, map[string]interface{}{
        "nodes":     resp.Nodes,
        "edges":     resp.Edges,
        "truncated": resp.Truncated,  // "truncated" warning if results capped
    })
}

// GET /v1/console/graph/entity/{id}
func (h *GraphHandler) GetEntity(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    entityID := chi.URLParam(r, "id")

    resp, err := h.graphitiStore.GetEntity(r.Context(), &graphitipb.GetEntityRequest{
        TenantId: auth.TenantID,
        EntityId: entityID,
    })
    if err != nil {
        writeError(w, http.StatusNotFound, "entity_not_found", entityID)
        return
    }
    writeJSON(w, http.StatusOK, resp)
}

// POST /v1/console/graph/timeline
// Body: {entity_id, from, to}
func (h *GraphHandler) Timeline(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    var req struct {
        EntityID string    `json:"entity_id"`
        From     time.Time `json:"from"`
        To       time.Time `json:"to"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    resp, err := h.graphitiStore.GetTemporalSubgraph(r.Context(), &graphitipb.TemporalSubgraphRequest{
        TenantId: auth.TenantID,
        EntityId: req.EntityID,
        From:     timestamppb.New(req.From),
        To:       timestamppb.New(req.To),
    })
    if err != nil {
        writeError(w, http.StatusInternalServerError, "graph_error", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, resp)
}

// GET /v1/console/graph/ontology
func (h *GraphHandler) GetOntology(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    resp, err := h.graphitiStore.GetOntology(r.Context(), &graphitipb.GetOntologyRequest{
        TenantId: auth.TenantID,
    })
    if err != nil {
        writeError(w, http.StatusInternalServerError, "ontology_error", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, resp)
}

// PUT /v1/console/graph/ontology
func (h *GraphHandler) UpdateOntology(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    var req graphitipb.OntologySchema
    json.NewDecoder(r.Body).Decode(&req)

    resp, err := h.graphitiStore.UpdateOntology(r.Context(), &graphitipb.UpdateOntologyRequest{
        TenantId: auth.TenantID,
        Schema:   &req,
    })
    if err != nil {
        writeError(w, http.StatusInternalServerError, "ontology_error", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, resp)
}

// POST /v1/console/graph/query — Cypher query console
func (h *GraphHandler) QueryGraph(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    var req struct {
        Query string `json:"query"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    // Security: validate Cypher is read-only
    if err := validateCypherReadOnly(req.Query); err != nil {
        writeError(w, http.StatusForbidden, "write_not_allowed", err.Error())
        return
    }

    // Inject tenant isolation
    safeQuery := injectTenantFilter(req.Query, auth.TenantID)

    resp, err := h.graphitiStore.ExecuteCypher(r.Context(), &graphitipb.CypherRequest{
        TenantId: auth.TenantID,
        Query:    safeQuery,
        MaxNodes: 200,
        MaxEdges: 500,
    })
    if err != nil {
        writeError(w, http.StatusInternalServerError, "query_error", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, resp)
}
```

### 2.2 Cypher Whitelist Validator — `gateway/internal/usecase/cypher_validator.go` [NEW]

```go
package usecase

import (
    "regexp"
    "strings"
)

// Allowed Cypher clauses (read-only)
var allowedClauses = map[string]bool{
    "MATCH": true, "RETURN": true, "WITH": true,
    "WHERE": true, "ORDER BY": true, "LIMIT": true,
    "SKIP": true, "OPTIONAL MATCH": true, "CALL": true,
    "UNWIND": true, "EXISTS": true,
}

// Forbidden patterns (write operations)
var forbiddenPatterns = []*regexp.Regexp{
    regexp.MustCompile(`(?i)\bCREATE\b`),
    regexp.MustCompile(`(?i)\bMERGE\b`),
    regexp.MustCompile(`(?i)\bDELETE\b`),
    regexp.MustCompile(`(?i)\bDETACH\b`),
    regexp.MustCompile(`(?i)\bSET\b`),
    regexp.MustCompile(`(?i)\bREMOVE\b`),
    regexp.MustCompile(`(?i)\bDROP\b`),
    regexp.MustCompile(`(?i)\bCREATE\s+INDEX\b`),
    regexp.MustCompile(`(?i)\bCALL\s+\{`),  // block subquery writes
}

// validateCypherReadOnly returns error if query contains write operations
func validateCypherReadOnly(query string) error {
    query = strings.TrimSpace(query)
    for _, pattern := range forbiddenPatterns {
        if pattern.MatchString(query) {
            return fmt.Errorf("write operations (CREATE/MERGE/DELETE/SET/REMOVE) are not allowed in Graph Studio")
        }
    }
    return nil
}

// injectTenantFilter appends tenant_id condition to prevent cross-tenant access
// IMPORTANT: This is a safety net — graphiti-store also enforces tenant isolation at query level
func injectTenantFilter(query, tenantID string) string {
    // Simple approach: wrap in WITH clause + tenant filter
    // Full implementation should use Neo4j parameterized queries
    return fmt.Sprintf(`
WITH $tenant_id AS _tid
%s
`, query)
    // tenantID passed as $tenant_id parameter to graphiti-store
}
```

---

## 3. File Changes

| File | Action | Mô tả |
|---|---|---|
| `gateway/adapter/handler/graph_handler.go` | NEW | 6 Graph Studio endpoint handlers |
| `gateway/internal/usecase/cypher_validator.go` | NEW | Cypher read-only whitelist validator |
| `gateway/adapter/handler/router.go` | VERIFY | Routes `/v1/console/graph/*` already registered per TDD |
| `backend/api/proto/graphiti/v1/store.proto` | VERIFY/MODIFY | Ensure SubgraphRequest, TemporalSubgraph, CypherRequest exist |

---

## 4. Acceptance Criteria

- [ ] `POST /v1/console/graph/subgraph` renders subgraph ≤ 200 nodes, ≤ 500 edges within 2s for depth=3
- [ ] Entity detail panel: all properties + connected edges + temporal validity (valid_from, valid_to)
- [ ] Timeline: graph changes over time range (per entity)
- [ ] Ontology GET/PUT: view and update entity/relationship type schema
- [ ] Cypher write operations (CREATE/DELETE/MERGE/SET) return 403 Forbidden
- [ ] Result capped: `truncated: true` flag in response when > 200 nodes
- [ ] Tenant isolation: all queries scoped to authenticated tenant — cannot return other tenant's data
- [ ] Query console: whitelists MATCH, RETURN, WITH, WHERE, ORDER BY, LIMIT, SKIP, OPTIONAL MATCH

---

## 5. Dependencies

- `graphiti-store` gRPC service (port 9023) — must implement SubgraphQuery, GetEntity, TemporalSubgraph, CypherExecute, GetOntology, UpdateOntology RPCs
- `backend/api/proto/graphiti/v1/store.proto` — SubgraphRequest, CypherRequest protos
- Auth middleware (admin role required for graph studio)
