# TASK-ENT-007 — Governance: Memory Visibility API

| Field | Value |
|---|---|
| **Task ID** | TASK-ENT-007 |
| **Wave** | 3 |
| **Solution** | [SOL-ENT-003](../solutions/SOL-ENT-003-Governance-Center.md) §1.1 |
| **Component** | `services/vnp-admin/internal/usecase/` |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-ENT-001 |
| **Estimated** | 5h |

---

## Mục tiêu

Memory Visibility: admin can view all memories of any user across 7 engines.

---

## Công việc cụ thể

### `services/vnp-admin/internal/usecase/visibility.go` [NEW]

```go
type VisibilityUseCase struct {
    engines map[string]SearchEngine // engine name → client
    audit   port.AuditRepository
}

type MemoryInventory struct {
    UserID     string                      `json:"user_id"`
    TenantID   string                      `json:"tenant_id"`
    Engines    map[string][]MemoryUnit     `json:"engines"`
    TotalCount int                         `json:"total_count"`
    QueryTime  int64                       `json:"query_time_ms"`
}

func (u *VisibilityUseCase) GetUserMemories(ctx context.Context, req *VisibilityRequest) (*MemoryInventory, error) {
    start := time.Now()
    var wg sync.WaitGroup
    results := map[string][]MemoryUnit{}
    mu := sync.Mutex{}

    engines := []string{"cognee-search", "graphiti-search", "zep-search",
        "memobase-engine", "ov-search", "sm-search", "observe-service"}

    for _, eng := range engines {
        wg.Add(1)
        go func(e string) {
            defer wg.Done()
            units, _ := u.engines[e].ListUserMemories(ctx, req.TenantID, req.UserID, req.Filter)
            mu.Lock()
            results[e] = units
            mu.Unlock()
        }(eng)
    }
    wg.Wait()

    // Audit: log who viewed this user's memories
    u.audit.Record(ctx, "memory.visibility", map[string]any{
        "viewed_user_id": req.UserID, "engines": engines,
    })

    total := 0
    for _, units := range results { total += len(units) }

    return &MemoryInventory{
        UserID: req.UserID, TenantID: req.TenantID,
        Engines: results, TotalCount: total,
        QueryTime: time.Since(start).Milliseconds(),
    }, nil
}
```

### API endpoint

```go
// GET /v1/admin/users/{id}/memories?engine=cognee&limit=50
func (h *AdminHandler) GetUserMemories(w http.ResponseWriter, r *http.Request) {
    if !hasAdminRole(r.Context()) { writeError(w, 403, "forbidden", ""); return }
    userID := chi.URLParam(r, "id")
    tenantID := tenant.FromContext(r.Context())
    inventory, err := h.visibilityUC.GetUserMemories(r.Context(), &VisibilityRequest{
        TenantID: tenantID, UserID: userID,
        Filter: r.URL.Query().Get("engine"),
    })
    if err != nil { writeError(w, 500, "visibility_error", err.Error()); return }
    writeJSON(w, 200, inventory)
}
```

---

## Acceptance Criteria

- [ ] Admin sees all memories across 7 engines in < 2s
- [ ] Audit log records every visibility query
- [ ] Filter by engine name supported
- [ ] Non-admin → 403
- [ ] Engine errors don't fail entire request (partial results)

## Files

```
services/vnp-admin/internal/usecase/visibility.go          [NEW]
gateway/adapter/handler/admin_handler.go                   [MODIFY — GetUserMemories]
gateway/adapter/handler/router.go                          [MODIFY]
```

---

**Ghi chú audit:** vnp-admin/usecase/memory_visibility.go [NEW]: MemoryVisibilityService.ListMemories/DeleteMemory/GetMemoryDetails
