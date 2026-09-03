-- migrations/003_admin_ext.sql
-- Additional admin tables for governance, policies, audit log (MERGE-P1-T2)

-- Policy table for access control policies
CREATE TABLE IF NOT EXISTS vnp_policies (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    rules      JSONB NOT NULL DEFAULT '{}',
    active     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_policies_tenant ON vnp_policies(tenant_id);

-- Audit log for governance and GDPR
CREATE TABLE IF NOT EXISTS vnp_audit_log (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id  UUID REFERENCES tenants(id) ON DELETE SET NULL,
    actor      TEXT NOT NULL,
    action     TEXT NOT NULL,
    resource   TEXT,
    metadata   JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_log_tenant ON vnp_audit_log(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created ON vnp_audit_log(created_at DESC);

-- GDPR forget job tracking
CREATE TABLE IF NOT EXISTS vnp_gdpr_jobs (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     TEXT NOT NULL,
    tenant_id   UUID REFERENCES tenants(id) ON DELETE SET NULL,
    status      TEXT NOT NULL DEFAULT 'queued',  -- queued, processing, done, failed
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_gdpr_jobs_user ON vnp_gdpr_jobs(user_id);
