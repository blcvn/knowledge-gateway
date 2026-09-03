# TASK-ENT-002 — Repository Tenant Guard

| Field | Value |
|---|---|
| **Task ID** | TASK-ENT-002 |
| **Wave** | 1 |
| **Solution** | [SOL-ENT-004](../solutions/SOL-ENT-004-MultiTenant-Isolation.md) §1.2 |
| **Component** | `shared/pkg/tenant/guard.go` |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-ENT-001 |
| **Estimated** | 2h |

---

## Mục tiêu

Query guard: detect and reject SQL/Cypher queries missing tenant_id filter. Alert on violation.

---

## Công việc cụ thể

### `shared/pkg/tenant/guard.go` [NEW]

```go
package tenant

import (
    "log/slog"
    "strings"
    "github.com/prometheus/client_golang/prometheus"
)

var missingTenantFilterTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{Name: "vnp_repository_missing_tenant_filter_total"},
    []string{"service", "operation"},
)

func init() { prometheus.MustRegister(missingTenantFilterTotal) }

type QueryGuard struct {
    service string
    log     *slog.Logger
    strict  bool // if true: return error on violation; if false: log only
}

func NewQueryGuard(service string, strict bool) *QueryGuard {
    return &QueryGuard{service: service, log: slog.Default(), strict: strict}
}

// AssertTenantFilter — call before executing any database query
func (g *QueryGuard) AssertTenantFilter(operation, query string) error {
    normalized := strings.ToLower(query)
    if !strings.Contains(normalized, "tenant_id") {
        missingTenantFilterTotal.WithLabelValues(g.service, operation).Inc()
        g.log.Error("SECURITY: query missing tenant_id filter",
            "service", g.service, "operation", operation, "query", query)
        if g.strict {
            return ErrMissingTenantFilter
        }
    }
    return nil
}

var ErrMissingTenantFilter = errors.New("security: query must include tenant_id filter")
```

### Usage in repositories

```go
// services/observe-service/internal/adapter/pg/observation_repo.go
func (r *ObservationRepo) GetBySession(ctx context.Context, sessionID, tenantID string) ([]Observation, error) {
    query := `SELECT * FROM observations WHERE session_id=$1 AND tenant_id=$2`
    r.guard.AssertTenantFilter("GetBySession", query) // passes: has tenant_id
    return r.db.Query(ctx, query, sessionID, tenantID)
}
```

---

## Acceptance Criteria

- [ ] AssertTenantFilter detects missing tenant_id in SQL
- [ ] Counter metric incremented on violation
- [ ] Strict mode: returns error on violation
- [ ] Log mode: logs only (for gradual rollout)
- [ ] `go test ./shared/pkg/tenant/...` passes

## Files

```
shared/pkg/tenant/guard.go       [NEW]
shared/pkg/tenant/guard_test.go  [NEW]
```

---

**Ghi chú audit:** shared/pkg/tenant/guard.go [NEW]: Guard() + GuardProject() + MustGuard() + WithTenant() + WithTenantProject()
