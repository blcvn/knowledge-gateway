# TASK-CORE-005 — Cascading Forget (GDPR Delete)

| Field | Value |
|---|---|
| **Task ID** | TASK-CORE-005 |
| **Wave** | 2 |
| **Solution** | [SOL-CORE-003](../solutions/SOL-CORE-003-Cascading-Forget.md) |
| **Component** | `services/vnp-admin/`, `gateway/` |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-CORE-007 (audit migration) |
| **Estimated** | 5h |

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** vnp-event/usecase/gdpr_service.go: Forget() cascades across all 6 engines; dry-run preview
---

## Mục tiêu

`POST /v1/admin/forget` → cascading delete user data từ tất cả 8 engines + immutable audit.

---

## Công việc cụ thể

### 1. `gateway/adapter/handler/admin_handler.go` [MODIFY] — add Forget endpoint

```go
// POST /v1/admin/forget
func (h *AdminHandler) Forget(w http.ResponseWriter, r *http.Request) {
    if !hasAdminRole(r.Context()) {
        writeError(w, 403, "forbidden", "admin role required")
        return
    }

    var req domain.ForgetRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, 400, "invalid_request", err.Error())
        return
    }

    auditID := uuid.NewString()
    h.auditLog.Record(r.Context(), "forget.initiated", map[string]any{
        "audit_id": auditID, "user_id": req.UserID,
        "tenant_id": req.TenantID, "reason": req.Reason,
        "request_id": req.RequestID,
    })

    start := time.Now()
    result := h.forgetUC.CascadeDelete(r.Context(), req.TenantID, req.UserID)

    h.auditLog.Record(r.Context(), "forget.completed", map[string]any{
        "audit_id": auditID, "duration_ms": time.Since(start).Milliseconds(),
        "deleted_from": result.Success, "failed": result.Failed,
    })

    writeJSON(w, 200, map[string]any{
        "user_id": req.UserID, "deleted_from": result.Success,
        "duration_ms": time.Since(start).Milliseconds(), "audit_id": auditID,
    })
}
```

### 2. `services/vnp-admin/internal/usecase/forget.go` [NEW]

```go
type ForgetEngine interface {
    Name() string
    DeleteUser(ctx context.Context, tenantID, userID string) error
}

type ForgetUseCase struct {
    engines []ForgetEngine
    neo4j   port.Neo4jAdapter
    minio   port.MinIOAdapter
    events  port.EventRepository
    obs     port.ObserveRepository
}

type DeleteResult struct {
    Success []string
    Failed  map[string]error
}

func (u *ForgetUseCase) CascadeDelete(ctx context.Context, tenantID, userID string) DeleteResult {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    type r struct{ name string; err error }
    total := len(u.engines) + 4
    ch := make(chan r, total)

    for _, eng := range u.engines {
        go func(e ForgetEngine) {
            ch <- r{e.Name(), e.DeleteUser(ctx, tenantID, userID)}
        }(eng)
    }
    go func() { ch <- r{"neo4j", u.neo4j.DeleteUserNodes(ctx, tenantID, userID)} }()
    go func() { ch <- r{"minio", u.minio.DeleteUserBucket(ctx, tenantID, userID)} }()
    go func() { ch <- r{"events", u.events.DeleteUser(ctx, tenantID, userID)} }()
    go func() { ch <- r{"observe", u.obs.DeleteUser(ctx, tenantID, userID)} }()

    out := DeleteResult{Failed: map[string]error{}}
    for i := 0; i < total; i++ {
        res := <-ch
        if res.err != nil { out.Failed[res.name] = res.err } else { out.Success = append(out.Success, res.name) }
    }
    return out
}
```

### 3. Per-engine ForgetEngine adapters [NEW]

```
services/vnp-admin/internal/adapter/forget/cognee_forget.go
services/vnp-admin/internal/adapter/forget/graphiti_forget.go
services/vnp-admin/internal/adapter/forget/zep_forget.go
services/vnp-admin/internal/adapter/forget/memobase_forget.go
services/vnp-admin/internal/adapter/forget/ov_forget.go
services/vnp-admin/internal/adapter/forget/sm_forget.go
```

Mỗi adapter gọi gRPC `DeleteUser(tenant_id, user_id)` trên service tương ứng.

---

## Acceptance Criteria

- [ ] Admin-only: `403 Forbidden` nếu không có admin role
- [ ] Fan-out: tất cả 8 targets được gọi parallel
- [ ] 10s timeout: partial results returned (không fail toàn bộ)
- [ ] Audit log: immutable record trước và sau khi delete
- [ ] Response `deleted_from` list chứa tên engines thành công

## Files

```
gateway/adapter/handler/admin_handler.go           [MODIFY]
services/vnp-admin/internal/usecase/forget.go      [NEW]
services/vnp-admin/internal/port/forget.go         [NEW]
services/vnp-admin/internal/adapter/forget/*.go    [NEW — 6 adapters]
```
