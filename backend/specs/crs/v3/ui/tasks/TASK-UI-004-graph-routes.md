# TASK-UI-004 — Graph Studio Route Registration

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-004 |
| **Wave** | 1 (Backend API) |
| **Solution** | [SOL-UI-001](../solutions/SOL-UI-001-Graph-Studio.md) §3 |
| **Component** | `gateway/adapter/handler/router.go` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-UI-001 |
| **Estimated** | 1h |

---

## Mục tiêu

Register tất cả 6 Graph Studio routes vào gateway router. Wire CypherValidator vào GraphHandler.

---

## Công việc cụ thể

### 1. Modify `gateway/adapter/handler/router.go` [MODIFY]

```go
// In SetupRoutes() or equivalent, add Graph Studio routes
// Requires admin role (Graph Studio is developer/admin tool)

r.Route("/v1/console/graph", func(r chi.Router) {
    r.Use(authMiddleware)
    r.Use(requireAdmin) // Graph Studio requires admin or ml_engineer role

    // Subgraph visualization
    r.Post("/subgraph", graphH.Subgraph)

    // Entity detail panel
    r.Get("/entity/{id}", graphH.GetEntity)

    // Timeline view
    r.Post("/timeline", graphH.Timeline)

    // Ontology management
    r.Get("/ontology", graphH.GetOntology)
    r.Put("/ontology", graphH.UpdateOntology)

    // Cypher query console (read-only, validated)
    r.Post("/query", graphH.QueryGraph)
})
```

### 2. Wire dependencies in `gateway/main.go` or DI setup [MODIFY]

```go
// Initialize Graph Studio dependencies
cypherValidator := usecase.NewCypherValidator()

graphitiStoreClient := client.NewGraphitiStoreClient(
    inProcessRegistry.GetConn("graphiti-store"),
)

graphHandler := handler.NewGraphHandler(
    graphitiStoreClient,
    cypherValidator,
    logger,
)

// Pass to router setup
router := handler.SetupRoutes(
    // ...existing handlers...
    graphHandler,
)
```

### 3. E2E smoke test `gateway/adapter/handler/graph_handler_integration_test.go` [NEW]

```go
//go:build integration

package handler_test

// Tests require graphiti-store gRPC service running
func TestGraphSubgraph_TenantIsolation(t *testing.T) {
    // Start test server with mock graphiti-store
    // Call POST /v1/console/graph/subgraph as tenant-A
    // Assert response doesn't contain tenant-B nodes
}

func TestGraphQuery_WritesBlocked(t *testing.T) {
    req := httptest.NewRequest("POST", "/v1/console/graph/query",
        strings.NewReader(`{"query":"CREATE (n:Test)"}`))
    rr := httptest.NewRecorder()
    // Call handler directly (no HTTP server needed)
    graphH.QueryGraph(rr, req)
    assert.Equal(t, http.StatusForbidden, rr.Code)
}
```

---

## Acceptance Criteria

- [ ] All 6 routes registered and reachable
- [ ] `POST /v1/console/graph/query` → 403 for write queries
- [ ] `GET /v1/console/graph/entity/{id}` → 404 for non-existent entity
- [ ] All routes require `admin` role (or configurable `ml_engineer` role)
- [ ] `CypherValidator` wired into `GraphHandler` at startup
- [ ] `go build ./gateway/...` passes

## Files

```
gateway/adapter/handler/router.go                              [MODIFY — add /v1/console/graph/* routes]
gateway/main.go                                                [MODIFY — wire GraphHandler deps]
gateway/adapter/handler/graph_handler_integration_test.go      [NEW]
```
