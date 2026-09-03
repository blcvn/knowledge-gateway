# SOL-GR-007 — Solution: Admin Service (Tenant Management)

| Field | Value |
|---|---|
| **Solution ID** | SOL-GR-007 |
| **CR** | CR-GR-007 |
| **TDD ref** | [03-graphiti-services.md](../../../tdd/architecture/03-graphiti-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |
| **Component** | `services/graphiti-admin` |

---

## 1. Phân tích

Admin operations: index rebuild, node deletion, metrics collection.

```go
// services/graphiti-admin/internal/usecase/admin.go [NEW]
func (u *AdminUseCase) RebuildIndex(ctx context.Context, tenantID string) error {
    // 1. Get all entities for tenant
    entities, _ := u.graphRepo.GetAllEntities(ctx, tenantID)
    // 2. Re-embed all entities
    for _, e := range entities {
        embedding, _ := u.embedder.Embed(ctx, e.Text)
        u.vectorRepo.Update(ctx, e.ID, embedding)
    }
    return nil
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/graphiti-admin/internal/usecase/admin.go` | NEW |
| `gateway/adapter/handler/admin_handler.go` | MODIFY — add graphiti admin endpoints |

---

## 3. Acceptance Criteria

- [ ] Index rebuild without downtime
- [ ] Tenant-level graph purge
- [ ] Metrics: node count, edge count, embedding count per tenant
