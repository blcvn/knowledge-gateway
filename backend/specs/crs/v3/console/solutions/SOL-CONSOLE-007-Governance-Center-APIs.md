# Solution: SOL-CONSOLE-007 — Governance Center Backend APIs

**CR:** CR-CONSOLE-007
**TDD refs:** `architecture/12-agentmemory-services.md §governance`, `models/vnp-admin.md`
**Version:** v3/console

---

## 1. Architecture

Governance Center extends `vnp-admin` service with:
- Audit log query (PostgreSQL `audit_log` table — immutable)
- OPA policy management (CRUD + validate)
- Tenant management (admin tools)

---

## 2. Audit Log Handler

```go
// gateway/adapter/handler/governance_handler.go [NEW]
type GovernanceHandler struct {
    registry port.GRPCRegistry
    db       *pgxpool.Pool
    opaClient port.OPAClient
}

// GET /v1/admin/audit?user_id=xxx&operation=forget&from=2026-09-01&limit=50
func (h *GovernanceHandler) GetAuditLog(w http.ResponseWriter, r *http.Request) {
    tenantID := tenant.FromContext(r.Context())
    userID   := r.URL.Query().Get("user_id")
    operation := r.URL.Query().Get("operation")
    from     := r.URL.Query().Get("from")
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit == 0 { limit = 50 }

    q := `
        SELECT id, actor_id, operation, target_user_id, metadata, created_at
        FROM audit_log
        WHERE tenant_id = $1
    `
    args := []any{tenantID}
    i := 2
    if userID != "" { q += fmt.Sprintf(" AND target_user_id = $%d", i); args = append(args, userID); i++ }
    if operation != "" { q += fmt.Sprintf(" AND operation = $%d", i); args = append(args, operation); i++ }
    if from != "" { q += fmt.Sprintf(" AND created_at >= $%d", i); args = append(args, from); i++ }
    q += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", i)
    args = append(args, limit)

    rows, err := h.db.Query(r.Context(), q, args...)
    if err != nil { writeError(w, 500, "query_failed", err.Error()); return }
    defer rows.Close()

    entries := []map[string]any{}
    for rows.Next() {
        var id, actorID, op, targetUserID string
        var metadata map[string]any
        var createdAt time.Time
        rows.Scan(&id, &actorID, &op, &targetUserID, &metadata, &createdAt)
        entries = append(entries, map[string]any{
            "id": id, "actor_id": actorID, "operation": op,
            "target_user_id": targetUserID, "metadata": metadata,
            "created_at": createdAt,
        })
    }
    writeJSON(w, 200, map[string]any{"entries": entries, "total": len(entries)})
}

// GET /v1/admin/audit/export
func (h *GovernanceHandler) ExportAuditLog(w http.ResponseWriter, r *http.Request) {
    tenantID := tenant.FromContext(r.Context())
    from     := r.URL.Query().Get("from")
    to       := r.URL.Query().Get("to")

    rows, err := h.db.Query(r.Context(), `
        SELECT id, actor_id, operation, target_user_id, created_at
        FROM audit_log WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3
        ORDER BY created_at`,
        tenantID, from, to)
    if err != nil { writeError(w, 500, "export_failed", err.Error()); return }
    defer rows.Close()

    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", "attachment; filename=audit_log.csv")
    enc := csv.NewWriter(w)
    enc.Write([]string{"id", "actor_id", "operation", "target_user_id", "created_at"})
    for rows.Next() {
        var id, actorID, op, targetUserID string; var ts time.Time
        rows.Scan(&id, &actorID, &op, &targetUserID, &ts)
        enc.Write([]string{id, actorID, op, targetUserID, ts.Format(time.RFC3339)})
    }
    enc.Flush()
}

// POST /v1/admin/policies
func (h *GovernanceHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Name   string `json:"name"`
        Module string `json:"module"` // Rego policy content
    }
    json.NewDecoder(r.Body).Decode(&req)
    if err := h.opaClient.PutPolicy(r.Context(), req.Name, req.Module); err != nil {
        writeError(w, 400, "policy_invalid", err.Error())
        return
    }
    writeJSON(w, 201, map[string]string{"name": req.Name, "status": "active"})
}

// POST /v1/admin/policies/validate
func (h *GovernanceHandler) ValidatePolicy(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Module string         `json:"module"` // Rego content
        Input  map[string]any `json:"input"`  // sample input
    }
    json.NewDecoder(r.Body).Decode(&req)
    result, err := h.opaClient.Evaluate(r.Context(), req.Module, req.Input)
    if err != nil { writeError(w, 400, "policy_error", err.Error()); return }
    writeJSON(w, 200, map[string]any{"result": result, "valid": true})
}

// GET /v1/admin/tenants
func (h *GovernanceHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
    // Admin-only: no tenant scoping
    conn, _ := h.registry.Get("vnp-platform")
    client  := platformpb.NewVnpPlatformServiceClient(conn)
    resp, _ := client.ListTenants(r.Context(), &platformpb.ListTenantsRequest{})
    writeJSON(w, 200, map[string]any{"tenants": resp.Tenants})
}

// GET /v1/admin/users/{id}/memories
func (h *GovernanceHandler) GetUserMemories(w http.ResponseWriter, r *http.Request) {
    userID   := chi.URLParam(r, "id")
    tenantID := tenant.FromContext(r.Context())
    conn, _  := h.registry.Get("vnp-search-hub")
    client   := searchpb.NewVnpSearchHubServiceClient(conn)
    resp, err := client.Recall(r.Context(), &searchpb.RecallRequest{
        TenantId: tenantID, UserId: userID,
        Query: "", MaxResults: 100, // no query = all memories
    })
    if err != nil { writeError(w, 500, "recall_failed", err.Error()); return }
    writeJSON(w, 200, map[string]any{"user_id": userID, "memories": resp.Results})
}
```

---

## 3. Audit Log Writing Pattern

```go
// shared/pkg/audit/logger.go [NEW]
// Write to audit_log table for sensitive operations
func LogAuditEvent(ctx context.Context, db *pgxpool.Pool, operation, targetUserID string, metadata map[string]any) {
    tenantID := tenant.FromContext(ctx)
    actorID  := auth.UserIDFromContext(ctx)
    metaJSON, _ := json.Marshal(metadata)
    db.Exec(ctx, `
        INSERT INTO audit_log (tenant_id, actor_id, operation, target_user_id, metadata)
        VALUES ($1, $2, $3, $4, $5)`,
        tenantID, actorID, operation, targetUserID, metaJSON)
}
// Usage in admin.ForgetUser:
//   audit.LogAuditEvent(ctx, db, "admin.forget", userID, map[string]any{"engines": engines_deleted})
```

---

## 4. DB Migration

```sql
-- deployment/dev/migrations/0050_audit_policies.sql
CREATE TABLE IF NOT EXISTS audit_log (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      TEXT NOT NULL,
    actor_id       TEXT NOT NULL,
    operation      TEXT NOT NULL,
    target_user_id TEXT,
    metadata       JSONB,
    created_at     TIMESTAMPTZ DEFAULT NOW()
);
-- Immutable: NO UPDATE or DELETE grants on this table
REVOKE UPDATE, DELETE ON audit_log FROM PUBLIC;
CREATE INDEX idx_audit_tenant_time ON audit_log(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS opa_policies (
    name       TEXT PRIMARY KEY,
    tenant_id  TEXT NOT NULL,
    module     TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 5. File Changes

| File | Action |
|---|---|
| `gateway/adapter/handler/governance_handler.go` | **[NEW]** |
| `gateway/internal/port/opa_client.go` | **[NEW]** OPA interface |
| `gateway/internal/adapter/opa/client.go` | **[NEW]** OPA HTTP client |
| `shared/pkg/audit/logger.go` | **[NEW]** |
| `deployment/dev/migrations/0050_audit_policies.sql` | **[NEW]** |
| `gateway/adapter/handler/router.go` | **[MODIFY]** governance routes |
