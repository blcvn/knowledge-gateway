# Change Request: CR-ENT-004 — Multi-Tenant Isolation Enforcement

**CR ID:** CR-ENT-004
**Component:** `backend/shared/pkg/tenant`, `backend/gateway`
**Priority:** 🔴 Critical
**Status:** Open
**Version:** v5 / Enterprise & Operations
**Solution:** [S9 — Enterprise Governance](../../../bussiness/solutions/S9-governance-compliance.md)
**Features:** [F14](../../../features/14-auth-multitenancy/README.md)
**ADR:** [ADR-006](../../../adr/ADR-006-tenantid-isolation.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P2-03 | Platform Engineer | Không có kiểm tra tự động cross-tenant isolation |
| PP-P4-03 | Enterprise Architect | Data leak risk — audit không đủ |

**Security requirement:** Zero cross-tenant data access — Tenant A không được đọc memory của Tenant B dù cùng shared database.

---

## 2. Defense-in-Depth Approach

```
Layer 1 — API Gateway: TenantID injection từ JWT claims
Layer 2 — Repository: Mandatory tenant_id WHERE clause
Layer 3 — Integration test: CI gate cho cross-tenant isolation
Layer 4 — Monitoring: Alert nếu query thiếu tenant_id filter
```

---

## 3. Implementation

### 3.1 `backend/shared/pkg/tenant/middleware.go` [MODIFY]

```go
// TenantMiddleware injects TenantID từ JWT vào context
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := jwtutil.ParseClaims(r)
        if claims.TenantID == "" {
            http.Error(w, "missing tenant", 401)
            return
        }
        ctx := context.WithValue(r.Context(), TenantKey, claims.TenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// FromContext panics if TenantID missing (dev error caught early)
func FromContext(ctx context.Context) string {
    tid, ok := ctx.Value(TenantKey).(string)
    if !ok || tid == "" {
        panic("BUG: TenantID not in context — missing TenantMiddleware?")
    }
    return tid
}
```

### 3.2 Integration test (CI gate)

```go
// tests/integration/tenant_isolation_test.go [NEW]
func TestCrossTenantLeakPrevention(t *testing.T) {
    // Store secret data as Tenant A
    storeAs(t, "tenant_a", "ultra secret project data")
    
    // Attempt to recall as Tenant B → must return 0 results
    results := recallAs(t, "tenant_b", "secret project")
    assert.Empty(t, results, "SECURITY: Cross-tenant data leak detected!")
}
// This test MUST pass in CI before any deploy
```

---

## 4. Acceptance Criteria

- [ ] Every DB query has `WHERE tenant_id = $N` (enforced by `tenant.FromContext`)
- [ ] JWT without tenant_id → 401 Unauthorized
- [ ] Cross-tenant integration test passes in CI (mandatory gate)
- [ ] `tenant.FromContext()` panics nếu context thiếu TenantID (catch bug early)
- [ ] Monitoring alert: query without tenant_id filter (log + Prometheus counter)
