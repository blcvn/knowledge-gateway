# TASK-BE-009 — Console Governance Handler + `audit_logs` migration

| Field | Value |
|---|---|
| **Task ID** | TASK-BE-009 |
| **Layer** | Backend — Go / PostgreSQL |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-006 CR-007](../solutions/SOL-006-Adaptive-to-Org-Solutions.md) + [SOL-007 §7](../solutions/SOL-007-Gap-Fixes.md) |
| **Priority** | 🟠 P1 |
| **Depends On** | — |
| **Estimated** | 4h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `vnp-platform/migrations/0005_create_audit_logs.sql` |
| CREATE | `vnp-platform/migrations/0006_create_opa_policies.sql` |
| CREATE | `gateway/internal/adapter/handler/console_governance_handler.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

### Migration: `0005_create_audit_logs.sql`

```sql
-- +migrate Up
CREATE TABLE IF NOT EXISTS audit_logs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    actor_id    UUID        NOT NULL,
    action      TEXT        NOT NULL,       -- CREATE|UPDATE|DELETE|GDPR_FORGET|LOGIN
    entity_type TEXT        NOT NULL,       -- MemoryItem|User|Tenant|Policy
    entity_id   TEXT,
    result      TEXT        NOT NULL DEFAULT 'success',
    metadata    JSONB       DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_tenant ON audit_logs(tenant_id, created_at DESC);
CREATE INDEX idx_audit_actor  ON audit_logs(actor_id);
CREATE INDEX idx_audit_action ON audit_logs(action);

-- +migrate Down
DROP TABLE IF EXISTS audit_logs;
```

### Migration: `0006_create_opa_policies.sql`

```sql
-- +migrate Up
CREATE TABLE IF NOT EXISTS opa_policies (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    name        TEXT        NOT NULL,
    rego_code   TEXT        NOT NULL,
    scope       TEXT        NOT NULL DEFAULT 'memory:*',
    enabled     BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_opa_tenant ON opa_policies(tenant_id);

-- +migrate Down
DROP TABLE IF EXISTS opa_policies;
```

### Handler: `console_governance_handler.go`

```go
package handler

type ConsoleGovernanceHandler struct {
    adminSvc  VNPAdminClient    // gRPC → vnp-admin
    db        *sql.DB           // audit_logs, opa_policies
    eventSvc  VNPEventClient    // GDPR cascade
    searchHub VNPSearchHubClient // GDPR cross-engine delete
    allEngines []EngineClient   // Tất cả engines để cascade GDPR delete
}

// GET /v1/console/governance/tenants
func (h *ConsoleGovernanceHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
    tenants, _ := h.adminSvc.ListTenants(r.Context())
    httputil.JSON(w, 200, tenants)
}

// POST /v1/console/governance/tenants
func (h *ConsoleGovernanceHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
    var req map[string]any; json.NewDecoder(r.Body).Decode(&req)
    tenant, _ := h.adminSvc.CreateTenant(r.Context(), req)
    httputil.JSON(w, 201, tenant)
}

// PUT /v1/console/governance/tenants/{id}
func (h *ConsoleGovernanceHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    var req map[string]any; json.NewDecoder(r.Body).Decode(&req)
    tenant, _ := h.adminSvc.UpdateTenant(r.Context(), id, req)
    httputil.JSON(w, 200, tenant)
}

// GET /v1/console/governance/policies
func (h *ConsoleGovernanceHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
    tenantID := authctx.TenantID(r.Context())
    rows, _ := h.db.QueryContext(r.Context(),
        `SELECT id, name, rego_code, scope, enabled FROM opa_policies WHERE tenant_id = $1`, tenantID)
    // scan and return
}

// POST /v1/console/governance/policies
func (h *ConsoleGovernanceHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) { /* ... */ }

// PUT /v1/console/governance/policies/{id}
func (h *ConsoleGovernanceHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) { /* ... */ }

// GET /v1/console/governance/audit
func (h *ConsoleGovernanceHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
    tenantID := authctx.TenantID(r.Context())
    q := r.URL.Query()
    where := "WHERE tenant_id = $1"
    args := []any{tenantID}
    if action := q.Get("action"); action != "" { where += " AND action = $2"; args = append(args, action) }
    // ... etc for actor_id, entity_type, from, to filters
    rows, _ := h.db.QueryContext(r.Context(),
        `SELECT id, actor_id, action, entity_type, entity_id, result, created_at FROM audit_logs `+where+` ORDER BY created_at DESC LIMIT 100`,
        args...)
    // scan and return
}

// POST /v1/console/governance/gdpr/forget/preview
func (h *ConsoleGovernanceHandler) GDPRPreview(w http.ResponseWriter, r *http.Request) {
    var req struct { UserID string `json:"user_id"` }
    json.NewDecoder(r.Body).Decode(&req)
    tenantID := authctx.TenantID(r.Context())

    // Count items per engine (dry-run)
    breakdown := map[string]int{}
    g, ctx := errgroup.WithContext(r.Context())
    var mu sync.Mutex
    for _, name := range []string{"memobase", "graphiti", "zep", "sm", "cognee"} {
        eng := name
        g.Go(func() error {
            count, _ := h.countUserData(ctx, eng, req.UserID, tenantID)
            mu.Lock(); breakdown[eng] = count; mu.Unlock()
            return nil
        })
    }
    g.Wait()
    total := 0
    for _, v := range breakdown { total += v }
    httputil.JSON(w, 200, map[string]any{
        "user_id": req.UserID, "estimated_items": total,
        "breakdown_by_engine": breakdown, "warnings": []string{},
    })
}

// POST /v1/console/governance/gdpr/forget
func (h *ConsoleGovernanceHandler) GDPRForget(w http.ResponseWriter, r *http.Request) {
    var req struct { UserID string `json:"user_id"` }
    json.NewDecoder(r.Body).Decode(&req)
    tenantID := authctx.TenantID(r.Context())

    // Parallel cascade delete across all engines
    g, ctx := errgroup.WithContext(r.Context())
    g.Go(func() error { return h.memoEngine.DeleteUser(ctx, req.UserID, tenantID) })
    g.Go(func() error { return h.zep.DeleteUser(ctx, req.UserID, tenantID) })
    g.Go(func() error { return h.sm.DeleteUser(ctx, req.UserID, tenantID) })
    g.Go(func() error { return h.graphiti.DeleteEpisodesByUser(ctx, req.UserID, tenantID) })
    g.Go(func() error { return h.eventSvc.PurgeUserEvents(ctx, req.UserID, tenantID) })
    g.Wait()

    // Audit log
    h.db.ExecContext(r.Context(),
        `INSERT INTO audit_logs (tenant_id, actor_id, action, entity_type, entity_id)
         VALUES ($1, $2, 'GDPR_FORGET', 'User', $3)`,
        tenantID, authctx.UserID(r.Context()), req.UserID)

    httputil.JSON(w, 200, map[string]any{"success": true, "deleted_count": 0})
}
```

### Routes

```go
mux.HandleFunc("GET /v1/console/governance/tenants",             authMiddleware(gov.ListTenants))
mux.HandleFunc("POST /v1/console/governance/tenants",            authMiddleware(gov.CreateTenant))
mux.HandleFunc("PUT /v1/console/governance/tenants/{id}",        authMiddleware(gov.UpdateTenant))
mux.HandleFunc("GET /v1/console/governance/policies",            authMiddleware(gov.ListPolicies))
mux.HandleFunc("POST /v1/console/governance/policies",           authMiddleware(gov.CreatePolicy))
mux.HandleFunc("PUT /v1/console/governance/policies/{id}",       authMiddleware(gov.UpdatePolicy))
mux.HandleFunc("GET /v1/console/governance/audit",               authMiddleware(gov.GetAuditLogs))
mux.HandleFunc("POST /v1/console/governance/gdpr/forget/preview", authMiddleware(gov.GDPRPreview))
mux.HandleFunc("POST /v1/console/governance/gdpr/forget",        authMiddleware(gov.GDPRForget))
```
