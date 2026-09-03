# SOL-ENT-004 — Solution: Multi-Tenant Isolation Enforcement

| Field | Value |
|---|---|
| **Solution ID** | SOL-ENT-004 |
| **CR** | [CR-ENT-004](../../../../docs/crs/v5/enterprise/CR-ENT-004-MultiTenant-Isolation.md) |
| **TDD ref** | [09-shared-packages.md](../../../tdd/architecture/09-shared-packages.md) §tenant |
| **Status** | Open |
| **Priority** | 🔴 Critical |

---

## 1. Giải pháp — Defense in Depth (4 layers)

### Layer 1: Gateway TenantID injection

```go
// shared/pkg/tenant/middleware.go [MODIFY]
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := jwtutil.ParseClaims(r)
        if claims.TenantID == "" {
            writeError(w, 401, "missing_tenant", "TenantID required")
            return
        }
        ctx := context.WithValue(r.Context(), tenantKey{}, claims.TenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// FromContext — always call this in usecase layer
func FromContext(ctx context.Context) string {
    v, _ := ctx.Value(tenantKey{}).(string)
    return v
}
```

### Layer 2: Repository tenant guard

```go
// shared/pkg/tenant/guard.go [NEW]
// TenantGuard wraps any repository call to verify tenant_id in query
type TenantGuard struct {
    log *slog.Logger
}

func (g *TenantGuard) AssertQuery(tenantID string, query string) error {
    // Reject any SQL query missing tenant_id filter
    if !strings.Contains(query, "tenant_id") {
        g.log.Error("SECURITY: query missing tenant_id filter", "query", query)
        return ErrMissingTenantFilter
    }
    return nil
}
```

### Layer 3: Integration test gate

```go
// tests/integration/tenant_isolation_test.go [NEW]
func TestCrossTenantIsolation(t *testing.T) {
    tenantA := setupTenant(t, "tenant_a")
    tenantB := setupTenant(t, "tenant_b")

    // Store memory as tenant_a
    store(t, tenantA, "secret data for A")

    // Recall as tenant_b — must return 0 results
    results := recall(t, tenantB, "secret data for A")
    assert.Empty(t, results, "tenant_b must not see tenant_a's memory")

    // Forget as tenant_b — must return error
    err := forget(t, tenantB, tenantA.UserID)
    assert.Error(t, err, "tenant_b must not forget tenant_a's user")
}
```

### Layer 4: Prometheus alert

```yaml
# deployment/dev/prometheus/alerts.yml
- alert: MissingTenantFilter
  expr: increase(vnp_repository_missing_tenant_filter_total[5m]) > 0
  severity: critical
  annotations:
    summary: "Repository query executed without tenant_id filter"
```

---

## 2. File Changes

| File | Action |
|---|---|
| `shared/pkg/tenant/middleware.go` | MODIFY — strict validation |
| `shared/pkg/tenant/guard.go` | NEW — query guard |
| `tests/integration/tenant_isolation_test.go` | NEW |
| `deployment/dev/prometheus/alerts.yml` | MODIFY — add tenant alert |

---

## 3. Acceptance Criteria

- [ ] Zero cross-tenant data access (integration test gate in CI)
- [ ] Every repository query must contain tenant_id filter
- [ ] Alert fires immediately if query missing tenant filter
- [ ] API key scoped to single tenant (cannot access other tenants)
