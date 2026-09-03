# Change Request: CR-CORE-003 — Cascading Forget (GDPR Delete)

**CR ID:** CR-CORE-003
**Component:** `backend/gateway`, `backend/services/vnp-admin`
**Priority:** 🔴 Critical
**Status:** Open
**Version:** v3 / Core Memory & Integration
**Solution:** [S9 — Enterprise Governance](../../../bussiness/solutions/S9-governance-compliance.md)
**Features:** [F01](../../../features/01-unified-memory-api/README.md), [F22](../../../features/22-governance-center/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P4-02 | Enterprise Architect | GDPR forget phải gọi manual từng engine — bỏ sót, không audit trail |
| PP-P4-01 | Enterprise Architect | Không biết AI đang nhớ gì về user — không có visibility |

**Before:** Dev phải call 6 separate delete APIs + manual audit log.
**After:** 1 API call → cascading delete tất cả 6 engines + Neo4j + MinIO + audit log, hoàn tất `< 3s`.

---

## 2. API Contract

```http
POST /v1/admin/forget
Authorization: Bearer <admin-key>
{
  "user_id": "u_123",
  "tenant_id": "t_456",
  "reason": "gdpr_request",
  "request_id": "gdpr_2026_001"   // for audit
}

→ 200 OK
{
  "user_id": "u_123",
  "deleted_from": ["cognee", "graphiti", "memobase", "zep", "openviking", "supermemory", "observe", "events"],
  "duration_ms": 1847,
  "audit_id": "audit_xyz"
}
```

---

## 3. Thay đổi đề xuất

### 3.1 `backend/gateway/internal/adapter/handler/admin_handler.go` [MODIFY]

```go
func (h *AdminHandler) Forget(w http.ResponseWriter, r *http.Request) {
    var req ForgetRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Only admin role
    if !isAdmin(r.Context()) { http.Error(w, "forbidden", 403); return }
    
    // Audit: log forget request (immutable)
    auditID := h.audit.LogForgetRequest(r.Context(), req)
    
    // Fan-out parallel delete
    start := time.Now()
    results := h.forgetService.CascadeDelete(r.Context(), req.TenantID, req.UserID)
    
    // Audit: log completion
    h.audit.LogForgetComplete(r.Context(), auditID, results, time.Since(start))
    
    writeJSON(w, 200, ForgetResponse{
        UserID:       req.UserID,
        DeletedFrom:  results.SuccessEngines,
        DurationMs:   time.Since(start).Milliseconds(),
        AuditID:      auditID,
    })
}
```

### 3.2 `backend/services/vnp-admin/internal/usecase/forget.go` [NEW]

```go
func (s *ForgetService) CascadeDelete(ctx context.Context, tenantID, userID string) *DeleteResult {
    engines := []DeleteAdapter{
        s.cognee, s.graphiti, s.memobase, s.zep, s.openviking, s.supermemory,
        s.observe, s.events,
    }
    
    var wg sync.WaitGroup
    results := &DeleteResult{}
    
    for _, eng := range engines {
        wg.Add(1)
        go func(e DeleteAdapter) {
            defer wg.Done()
            if err := e.DeleteUser(ctx, tenantID, userID); err != nil {
                results.AddFailure(e.Name(), err)
            } else {
                results.AddSuccess(e.Name())
            }
        }(eng)
    }
    wg.Wait()
    return results
}
```

---

## 4. Acceptance Criteria

- [ ] Cascading delete hoàn tất `< 3 giây`
- [ ] Audit log immutable (không thể delete audit records)
- [ ] Mọi delete phải có `tenant_id` — không thể delete cross-tenant
- [ ] Partial failure report (report engines thành công / thất bại)
- [ ] Only admin API key có thể call endpoint này
- [ ] `DELETE WHERE tenant_id=$1 AND user_id=$2` trên PostgreSQL, Neo4j, MinIO
