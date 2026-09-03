# SOL-CORE-003 — Solution: Cascading Forget (GDPR Delete)

| Field | Value |
|---|---|
| **Solution ID** | SOL-CORE-003 |
| **CR** | [CR-CORE-003](../../../../docs/crs/v3/core-memory/CR-CORE-003-Cascading-Forget.md) |
| **TDD ref** | [08-platform-services.md](../../../tdd/architecture/08-platform-services.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |

---

## 1. Giải pháp

### 1.1 `gateway/adapter/handler/admin_handler.go` [MODIFY]

```go
// POST /v1/admin/forget
func (h *AdminHandler) Forget(w http.ResponseWriter, r *http.Request) {
    if !hasRole(r.Context(), "admin") {
        writeError(w, 403, "forbidden", "admin role required")
        return
    }
    var req ForgetRequest
    json.NewDecoder(r.Body).Decode(&req)

    auditID := uuid.NewString()
    h.audit.Log(r.Context(), "forget.initiated", map[string]any{
        "audit_id": auditID, "user_id": req.UserID,
        "tenant_id": req.TenantID, "reason": req.Reason,
    })

    start := time.Now()
    results := h.forgetUC.CascadeDelete(r.Context(), req.TenantID, req.UserID)

    h.audit.Log(r.Context(), "forget.completed", map[string]any{
        "audit_id": auditID, "duration_ms": time.Since(start).Milliseconds(),
        "deleted_from": results.Success, "failed": results.Failed,
    })

    writeJSON(w, 200, ForgetResponse{
        UserID: req.UserID, DeletedFrom: results.Success,
        DurationMs: time.Since(start).Milliseconds(), AuditID: auditID,
    })
}
```

### 1.2 `services/vnp-admin/internal/usecase/forget.go` [NEW]

```go
type ForgetUseCase struct {
    engines []ForgetEngine  // per-engine adapters
    neo4j   Neo4jAdapter
    minio   MinIOAdapter
    events  EventRepo
    obs     ObserveRepo
}

type DeleteResult struct {
    Success []string
    Failed  map[string]error
}

func (u *ForgetUseCase) CascadeDelete(ctx context.Context, tenantID, userID string) DeleteResult {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    type result struct { engine string; err error }
    ch := make(chan result, len(u.engines)+4)

    // Fan-out: all memory engines
    for _, eng := range u.engines {
        go func(e ForgetEngine) {
            err := e.DeleteUser(ctx, tenantID, userID)
            ch <- result{e.Name(), err}
        }(eng)
    }

    // Fan-out: Neo4j graph entities
    go func() {
        err := u.neo4j.DeleteUserNodes(ctx, tenantID, userID)
        ch <- result{"neo4j", err}
    }()

    // Fan-out: MinIO files
    go func() {
        err := u.minio.DeleteUserBucket(ctx, tenantID, userID)
        ch <- result{"minio", err}
    }()

    // Fan-out: event timeline
    go func() {
        err := u.events.DeleteUser(ctx, tenantID, userID)
        ch <- result{"events", err}
    }()

    // Fan-out: observe sessions/observations
    go func() {
        err := u.obs.DeleteUser(ctx, tenantID, userID)
        ch <- result{"observe", err}
    }()

    total := len(u.engines) + 4
    out := DeleteResult{Failed: map[string]error{}}
    for i := 0; i < total; i++ {
        r := <-ch
        if r.err != nil { out.Failed[r.engine] = r.err } else { out.Success = append(out.Success, r.engine) }
    }
    return out
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `gateway/adapter/handler/admin_handler.go` | MODIFY — add Forget endpoint |
| `services/vnp-admin/internal/usecase/forget.go` | NEW |
| `services/vnp-admin/internal/port/forget.go` | NEW — ForgetEngine interface |
| `services/vnp-admin/internal/adapter/*/forget_adapter.go` | NEW — per-engine adapters |
| `deployment/dev/migrations/0XX_audit_log.sql` | NEW — audit_log table |

---

## 3. Acceptance Criteria

- [ ] GDPR delete hoàn tất < 10s với 6 engines + Neo4j + MinIO
- [ ] Immutable audit log với reason + request_id
- [ ] Partial success: report failed engines, không rollback thành công
- [ ] Admin-only endpoint (RBAC check)
- [ ] `GET /v1/admin/forget/{audit_id}` để track status
