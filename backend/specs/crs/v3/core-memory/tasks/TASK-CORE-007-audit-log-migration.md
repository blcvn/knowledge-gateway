# TASK-CORE-007 — Audit Log Database Migration

| Field | Value |
|---|---|
| **Task ID** | TASK-CORE-007 |
| **Wave** | 2 |
| **Solution** | [SOL-CORE-003](../solutions/SOL-CORE-003-Cascading-Forget.md) §1.3 |
| **Component** | `deployment/dev/migrations/` |
| **Priority** | 🔴 Critical |
| **Depends On** | — |
| **Estimated** | 1h |

---

## Mục tiêu

Tạo immutable audit_log table với Row Level Security (INSERT-only).

---

## Công việc cụ thể

### `deployment/dev/migrations/0040_audit_log.sql` [NEW]

```sql
-- +migrate Up

CREATE TABLE IF NOT EXISTS audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT        NOT NULL,
    user_id     TEXT,                           -- user subject (may be null for system ops)
    actor_id    TEXT        NOT NULL,           -- who performed the action (admin/system)
    operation   TEXT        NOT NULL,           -- 'forget.initiated', 'forget.completed', 'store', 'recall'
    resource_id TEXT,                           -- memory_id, session_id, etc.
    metadata    JSONB,                          -- extra context (reason, duration, etc.)
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_tenant_id   ON audit_log(tenant_id);
CREATE INDEX idx_audit_user_id     ON audit_log(user_id);
CREATE INDEX idx_audit_operation   ON audit_log(operation);
CREATE INDEX idx_audit_created_at  ON audit_log(created_at DESC);

-- Immutable: enable RLS, only INSERT allowed
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY audit_insert_only ON audit_log
    FOR INSERT WITH CHECK (true);
-- SELECT allowed to service role, no UPDATE/DELETE

COMMENT ON TABLE audit_log IS 'Immutable compliance audit trail for all memory operations';

-- +migrate Down
DROP TABLE IF EXISTS audit_log;
```

### `services/vnp-admin/internal/adapter/pg/audit_repo.go` [NEW]

```go
type AuditRepository struct { db *pgxpool.Pool }

func (r *AuditRepository) Record(ctx context.Context, operation string, meta map[string]any) error {
    tenantID := tenant.FromContext(ctx)
    actorID  := actor.FromContext(ctx)
    metaJSON, _ := json.Marshal(meta)

    _, err := r.db.Exec(ctx, `
        INSERT INTO audit_log(tenant_id, actor_id, operation, metadata)
        VALUES ($1, $2, $3, $4)`,
        tenantID, actorID, operation, metaJSON)
    return err
}
```

---

## Acceptance Criteria

- [ ] `audit_log` table created with RLS enabled
- [ ] INSERT succeeds, UPDATE/DELETE blocked by RLS
- [ ] Indexes on tenant_id, user_id, operation, created_at
- [ ] `AuditRepository.Record` stores correct fields
- [ ] Migration runs cleanly: `migrate up` no errors

## Files

```
deployment/dev/migrations/0040_audit_log.sql                [NEW]
services/vnp-admin/internal/adapter/pg/audit_repo.go       [NEW]
services/vnp-admin/internal/port/audit.go                   [NEW]
```
