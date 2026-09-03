# TASK-PLAT-023 — RBAC Router Integration

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-023 |
| **Wave** | 3 |
| **Solution** | [SOL-PLAT-007](../solutions/SOL-PLAT-007-RBAC-Authorization.md) §3 |
| **Component** | `gateway/adapter/handler/router.go` |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-PLAT-022 |
| **Estimated** | 2h |

---

## Mục tiêu

Gắn `RequirePermission()` middleware vào các routes trong gateway router.

---

## Công việc cụ thể

### 1. Sửa `gateway/adapter/handler/router.go` [MODIFY]

Thêm import:
```go
"vnp-memory/shared/pkg/auth"
```

Thêm permission guards cho các routes:
```go
// Memory API
r.With(auth.RequirePermission(auth.PermMemoryStore)).Post("/v1/memory/store", memoryHandler.Store)
r.With(auth.RequirePermission(auth.PermMemoryRecall)).Post("/v1/memory/recall", memoryHandler.Recall)
r.With(auth.RequirePermission(auth.PermMemoryForget)).Post("/v1/memory/forget", memoryHandler.Forget)
r.With(auth.RequirePermission(auth.PermMemoryRecall)).Get("/v1/memory/timeline", memoryHandler.Timeline)

// Admin API
r.With(auth.RequirePermission(auth.PermAdminForget)).Post("/v1/admin/forget", adminHandler.ForgetUser)
r.With(auth.RequirePermission(auth.PermAuditView)).Get("/v1/admin/audit", adminHandler.GetAuditLog)
r.With(auth.RequirePermission(auth.PermAuditView)).Get("/v1/admin/audit/export", adminHandler.ExportAuditLog)

// Console Org
r.With(auth.RequirePermission(auth.PermOrgWrite)).Put("/v1/console/org/settings", orgHandler.UpdateSettings)
r.With(auth.RequirePermission(auth.PermMembersAdmin)).Post("/v1/console/org/members/invite", orgHandler.InviteMember)
r.With(auth.RequirePermission(auth.PermMembersAdmin)).Delete("/v1/console/org/members/{id}", orgHandler.RemoveMember)
r.With(auth.RequirePermission(auth.PermMembersAdmin)).Put("/v1/console/org/members/{id}/role", orgHandler.ChangeRole)

// SDK
r.With(auth.RequirePermission(auth.PermAPIKeyCreate)).Post("/v1/console/sdk/keys", sdkHandler.CreateKey)
r.With(auth.RequirePermission(auth.PermAPIKeyRevoke)).Delete("/v1/console/sdk/keys/{id}", sdkHandler.RevokeKey)
```

### 2. Integration test `gateway/adapter/handler/rbac_integration_test.go` [NEW]

```go
// Test that viewer cannot access store endpoint
func TestRBAC_ViewerCannotStore(t *testing.T) {
    router := setupTestRouter()
    req := httptest.NewRequest("POST", "/v1/memory/store", body)
    req.Header.Set("X-Test-Role", "viewer") // test auth middleware
    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)
    assert.Equal(t, 403, rr.Code)
}
```

---

## Acceptance Criteria

- [ ] `POST /v1/memory/store` with viewer token → 403
- [ ] `POST /v1/admin/forget` with editor token → 403
- [ ] `POST /v1/admin/forget` with admin token → passes to handler
- [ ] `go build ./gateway/...` passes

## Files

```
gateway/adapter/handler/router.go                    [MODIFY]
gateway/adapter/handler/rbac_integration_test.go     [NEW]
```
