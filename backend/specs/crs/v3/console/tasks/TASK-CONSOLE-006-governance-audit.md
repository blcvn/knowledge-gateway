# TASK-CONSOLE-006 — Governance Center: Audit Log + OPA Policy + Infra Health

| Field | Value |
|---|---|
| **Task ID** | TASK-CONSOLE-006 |
| **Wave** | 2 |
| **Solution** | [SOL-CONSOLE-007](../solutions/SOL-CONSOLE-007-Governance-Center-APIs.md), [SOL-CONSOLE-008](../solutions/SOL-CONSOLE-008-Infrastructure-Health-APIs.md) |
| **Component** | `gateway/adapter/handler/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 4h |

---

## Mục tiêu

Implement Governance Center (audit + OPA) và Infrastructure Health handlers.

---

## Công việc cụ thể

### 1. DB Migrations [NEW]

`deployment/dev/migrations/0050_audit_policies.sql`:
```sql
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL, actor_id TEXT NOT NULL,
    operation TEXT NOT NULL, target_user_id TEXT,
    metadata JSONB, created_at TIMESTAMPTZ DEFAULT NOW()
);
REVOKE UPDATE, DELETE ON audit_log FROM PUBLIC;
CREATE INDEX idx_audit_tenant_time ON audit_log(tenant_id, created_at DESC);

CREATE TABLE opa_policies (
    name TEXT PRIMARY KEY, tenant_id TEXT NOT NULL,
    module TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

`deployment/dev/migrations/0051_infra_alerts.sql`:
```sql
CREATE TABLE infra_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    component TEXT NOT NULL, severity TEXT NOT NULL,
    message TEXT, since TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (component, severity)
);
```

### 2. Tạo `shared/pkg/audit/logger.go` [NEW]

```go
package audit

func Log(ctx context.Context, db *pgxpool.Pool, operation, targetUserID string, metadata map[string]any) {
    tenantID := tenant.FromContext(ctx)
    actorID  := auth.UserIDFromContext(ctx)
    data, _  := json.Marshal(metadata)
    db.Exec(ctx, `INSERT INTO audit_log (tenant_id, actor_id, operation, target_user_id, metadata) VALUES ($1,$2,$3,$4,$5)`,
        tenantID, actorID, operation, targetUserID, data)
}
```

### 3. Tạo `gateway/adapter/handler/governance_handler.go` [NEW]

Implement:
- `GET /v1/admin/audit` + `GET /v1/admin/audit/export` (CSV)
- `GET /v1/admin/policies` + `POST` + `DELETE` + `POST /validate`
- `GET /v1/admin/tenants` + `GET /v1/admin/users/{id}/memories`

### 4. Tạo `gateway/adapter/handler/infra_handler.go` [NEW]

Implement:
- `GET /v1/console/infrastructure/health` → parallel 5-component checks (3s timeout)
- `GET /v1/console/infrastructure/alerts` → active alerts from DB
- `GET /v1/console/infrastructure/resources` → vnp-observability gRPC

Background: `HealthMonitor.Start()` in gateway main — evaluates alert rules every 60s.

### 5. Routes `router.go` [MODIFY]

```go
r.Get("/v1/admin/audit",              govHandler.GetAuditLog)
r.Get("/v1/admin/audit/export",       govHandler.ExportAuditLog)
r.Post("/v1/admin/policies",          govHandler.CreatePolicy)
r.Post("/v1/admin/policies/validate", govHandler.ValidatePolicy)
r.Get("/v1/admin/tenants",            govHandler.ListTenants)
r.Get("/v1/admin/users/{id}/memories", govHandler.GetUserMemories)
r.Get("/v1/console/infrastructure/health",    infraHandler.GetHealth)
r.Get("/v1/console/infrastructure/alerts",    infraHandler.GetAlerts)
r.Get("/v1/console/infrastructure/resources", infraHandler.GetResources)
```

---

## Acceptance Criteria

- [ ] Audit log: immutable (REVOKE DELETE enforced)
- [ ] Export: valid CSV with headers
- [ ] OPA policy validate: returns result for sample input
- [ ] Health check: all 5 components in parallel, timeout 3s
- [ ] Alert rule engine: background goroutine starts with gateway
- [ ] Infra alerts: UPSERT per component/severity

## Files

```
deployment/dev/migrations/0050_audit_policies.sql  [NEW]
deployment/dev/migrations/0051_infra_alerts.sql    [NEW]
shared/pkg/audit/logger.go                        [NEW]
gateway/adapter/handler/governance_handler.go     [NEW]
gateway/adapter/handler/infra_handler.go          [NEW]
gateway/internal/infra/health_monitor.go          [NEW]
gateway/adapter/handler/router.go                 [MODIFY]
```
