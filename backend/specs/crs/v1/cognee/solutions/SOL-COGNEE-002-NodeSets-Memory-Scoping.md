# SOL-COGNEE-002 — Solution: NodeSets Memory Scoping

| Field | Value |
|---|---|
| **Solution ID** | SOL-COGNEE-002 |
| **CR** | [CR-COGNEE-002](../../../../docs/crs/v1/cognee/CR-COGNEE-002*.md) |
| **TDD ref** | [02-cognee-services.md](../../../tdd/architecture/02-cognee-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |

---
## 1. Giải pháp

NodeSets = named subgraphs scoped per tenant/user/dataset. Cho phép query chỉ trên subset của graph.

### 1.1 `services/cognee-cognify/internal/domain/nodeset.go` [NEW]

```go
type NodeSet struct {
    ID        uuid.UUID
    TenantID  string
    DatasetID uuid.UUID
    Name      string       // "project_docs", "api_specs", etc.
    NodeIDs   []uuid.UUID  // explicit member nodes
    Filter    string       // Cypher WHERE clause for dynamic membership
    CreatedAt time.Time
}
```

### 1.2 Cypher query with NodeSet scope

```go
func (r *GraphRepo) QueryInNodeSet(ctx context.Context, nodeSetID, query string) ([]Node, error) {
    cypher := `
        MATCH (n:Entity)-[:IN_NODESET]->(ns:NodeSet {id: $ns_id})
        WHERE n.content CONTAINS $query
        RETURN n ORDER BY n.created_at DESC LIMIT 50`
    return r.runCypher(ctx, cypher, map[string]any{"ns_id": nodeSetID, "query": query})
}
```

## 2. File Changes

| File | Action |
|---|---|
| `services/cognee-cognify/internal/domain/nodeset.go` | NEW |
| `services/cognee-search/internal/usecase/search.go` | MODIFY — add nodeset filter |
| `deployment/dev/migrations/0XX_nodesets.sql` | NEW |
