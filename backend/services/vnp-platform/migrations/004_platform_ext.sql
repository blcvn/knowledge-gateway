-- migrations/004_platform_ext.sql
-- Additional tables for analytics, spaces, forget rules (MERGE-P1-T3)

-- Forget rules for adaptive memory management (sm-analytics domain)
CREATE TABLE IF NOT EXISTS vnp_forget_rules (
    id        UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    rules     JSONB NOT NULL DEFAULT '[]',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_forget_rules_tenant ON vnp_forget_rules(tenant_id);

-- Debugger traces for memory context debugging
CREATE TABLE IF NOT EXISTS vnp_debug_traces (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID REFERENCES tenants(id) ON DELETE CASCADE,
    user_id     TEXT,
    query       TEXT,
    context     JSONB NOT NULL DEFAULT '{}',
    steps       JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_debug_traces_tenant ON vnp_debug_traces(tenant_id);
CREATE INDEX IF NOT EXISTS idx_debug_traces_created ON vnp_debug_traces(created_at DESC);
