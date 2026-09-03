-- Governance: Audit Entries
CREATE TABLE IF NOT EXISTS audit_entries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    TEXT NOT NULL,
    timestamp    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    operation    TEXT NOT NULL,
    target_ids   TEXT[] NOT NULL DEFAULT '{}',
    performed_by TEXT,
    project      TEXT NOT NULL DEFAULT '',
    details      JSONB,
    reason       TEXT
);

-- Snapshot records
CREATE TABLE IF NOT EXISTS snapshot_records (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    commit_hash TEXT,
    sessions    INTEGER NOT NULL DEFAULT 0,
    observations INTEGER NOT NULL DEFAULT 0,
    memories    INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_entries_tenant ON audit_entries(tenant_id, timestamp DESC);
CREATE INDEX idx_audit_entries_operation ON audit_entries(tenant_id, operation);
CREATE INDEX idx_audit_entries_project ON audit_entries(tenant_id, project);
