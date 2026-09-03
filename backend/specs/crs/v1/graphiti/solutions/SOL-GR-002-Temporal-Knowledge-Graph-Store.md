# SOL-GR-002 — Solution: Temporal Knowledge Graph Store

| Field | Value |
|---|---|
| **Solution ID** | SOL-GR-002 |
| **CR** | CR-GR-002 |
| **TDD ref** | [03-graphiti-services.md](../../../tdd/architecture/03-graphiti-services.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |
| **Component** | `services/graphiti-ingestion` |

---

## 1. Phân tích

Multi-backend graph storage: Neo4j (primary) + pgvector (for embedding search).

### Key implementation: `services/graphiti-ingestion/internal/adapter/graph/neo4j.go` [NEW]

```go
type Neo4jGraphRepo struct { driver neo4j.Driver }

func (r *Neo4jGraphRepo) UpsertEntity(ctx context.Context, tenantID string, entity *Entity) error {
    cypher := `
        MERGE (e:Entity {id: $id, tenant_id: $tenant_id})
        SET e.name = $name, e.type = $type,
            e.created_at = $created_at, e.updated_at = timestamp()
        RETURN e`
    _, err := r.driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
        return tx.Run(ctx, cypher, map[string]any{
            "id": entity.ID, "tenant_id": tenantID,
            "name": entity.Name, "type": entity.Type,
            "created_at": entity.CreatedAt,
        })
    })
    return err
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/graphiti-ingestion/internal/adapter/graph/neo4j.go` | NEW — Neo4j adapter |
| `services/graphiti-ingestion/internal/adapter/vector/pgvector.go` | NEW — pgvector adapter |

---

## 3. Acceptance Criteria

- [ ] Entities stored with tenant isolation
- [ ] Temporal edges include valid_from, valid_to
- [ ] pgvector embeddings queryable by cosine similarity
