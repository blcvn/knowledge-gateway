-- migrations/009_audit_logs.sql
-- TASK-BE-009: audit_logs table cho Governance Handler
-- Lưu tất cả hành động quản trị: CREATE, UPDATE, DELETE, GDPR_FORGET, LOGIN

-- +migrate Up
CREATE TABLE IF NOT EXISTS audit_log_entries (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   VARCHAR(64) NOT NULL,
    actor_id    VARCHAR(64) NOT NULL,
    action      TEXT        NOT NULL,   -- CREATE|UPDATE|DELETE|GDPR_FORGET|LOGIN|REVOKE
    entity_type TEXT        NOT NULL,   -- MemoryItem|User|Tenant|Policy|APIKey|Webhook
    entity_id   TEXT,
    result      TEXT        NOT NULL DEFAULT 'success', -- success|failure
    metadata    JSONB       DEFAULT '{}',
    ip          TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_tenant ON audit_log_entries(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_actor  ON audit_log_entries(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log_entries(action);

-- OPA Policies for fine-grained access control
CREATE TABLE IF NOT EXISTS opa_policies (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   VARCHAR(64) NOT NULL,
    name        TEXT        NOT NULL,
    rego_code   TEXT        NOT NULL,
    scope       TEXT        NOT NULL DEFAULT 'memory:*',
    enabled     BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_opa_policies_tenant ON opa_policies(tenant_id);

-- +migrate Down
DROP TABLE IF EXISTS opa_policies;
DROP TABLE IF EXISTS audit_log_entries;
