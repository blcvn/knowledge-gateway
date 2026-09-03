# TASK-009: Add `webhooks` Database Migration

**Solution**: [SOL-002](../solutions/SOL-002-org-sdk-api.md)  
**CR**: CR-002  
**Priority**: 🟡 High  
**Estimate**: 30 minutes  
**Status**: TODO  
**Depends on**: TASK-008

---

## Context

The `Webhook` entity (added in TASK-008) requires a new database table. All SQL migrations for this project are stored in `deployment/dev/migrations/`.

---

## Exact Task

### Step 1: Find the latest migration number

```bash
ls deployment/dev/migrations/ | sort | tail -5
```

### Step 2: Create the new migration file

Create `deployment/dev/migrations/<NEXT_NUMBER>_add_webhooks.sql` where `<NEXT_NUMBER>` is the next sequential number (e.g., if latest is `0042_something.sql`, create `0043_add_webhooks.sql`):

```sql
-- Migration: add webhooks table for SDK webhook management
-- CR: CR-002 | TASK-009

CREATE TABLE IF NOT EXISTS webhooks (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    url          TEXT        NOT NULL,
    events       TEXT[]      NOT NULL DEFAULT '{}',
    status       TEXT        NOT NULL DEFAULT 'active'
                             CHECK (status IN ('active', 'paused', 'failed')),
    secret_hash  TEXT,                         -- SHA-256 of signing secret, NULL if no secret
    success_rate FLOAT       NOT NULL DEFAULT 1.0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhooks_tenant_id ON webhooks(tenant_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_status    ON webhooks(tenant_id, status);

-- Add prefix and status columns to api_keys table if not present
ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS prefix     TEXT        DEFAULT '',
    ADD COLUMN IF NOT EXISTS status     TEXT        NOT NULL DEFAULT 'active'
                                        CHECK (status IN ('active', 'revoked', 'expired')),
    ADD COLUMN IF NOT EXISTS scopes     TEXT[]      NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS last_used  TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

COMMENT ON TABLE webhooks IS 'SDK webhook endpoints registered per tenant';
COMMENT ON COLUMN webhooks.secret_hash IS 'SHA-256 of signing secret. Never exposed via API.';
```

### Step 3: Verify the migration can be applied

If a local dev environment is running:
```bash
# Apply migration manually to verify syntax:
psql "$VNP_MEMORY_POSTGRES_DSN" -f deployment/dev/migrations/<NEXT_NUMBER>_add_webhooks.sql
```

If not running locally, verify SQL syntax only:
```bash
psql --dry-run "$VNP_MEMORY_POSTGRES_DSN" -f deployment/dev/migrations/<NEXT_NUMBER>_add_webhooks.sql
# or use psql -c to test individual statements
```

---

## Files to Create

| File | Content |
|------|---------|
| `deployment/dev/migrations/<NEXT_NUMBER>_add_webhooks.sql` | Migration SQL above |

---

## Acceptance Criteria

- [ ] Migration file created in `deployment/dev/migrations/` with correct sequential number
- [ ] `webhooks` table created with `id`, `tenant_id`, `url`, `events`, `status`, `secret_hash`, `success_rate`, `created_at`
- [ ] Index on `webhooks.tenant_id` created
- [ ] `api_keys` table altered to add `prefix`, `status`, `scopes`, `last_used`, `expires_at` columns (with `IF NOT EXISTS`)
- [ ] SQL is valid PostgreSQL (no syntax errors)
