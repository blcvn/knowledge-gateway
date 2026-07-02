CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'suspended', 'trial')),
    tier TEXT NOT NULL DEFAULT 'free'
        CHECK (tier IN ('free', 'pro', 'enterprise')),
    default_sharing_policy TEXT NOT NULL DEFAULT 'deny_all'
        CHECK (default_sharing_policy IN ('deny_all', 'share_within_tenant_read')),
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO tenants (id, slug, name, status, tier)
VALUES ('00000000-0000-0000-0000-000000000000', 'platform', 'Aevlex Platform', 'active', 'enterprise');

CREATE TABLE apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL
        CHECK (type IN ('agent_consumer', 'ingestion_producer', 'admin_tool', 'hybrid')),
    api_key_hash TEXT NOT NULL,
    api_key_prefix TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'revoked')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ,
    UNIQUE (tenant_id, slug)
);

CREATE INDEX idx_apps_api_key_hash
    ON apps(api_key_hash)
    WHERE status = 'active';

CREATE TABLE access_grants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    grantor_tenant_id UUID NOT NULL REFERENCES tenants(id),
    grantor_app_id UUID REFERENCES apps(id),
    grantee_tenant_id UUID NOT NULL REFERENCES tenants(id),
    grantee_app_id UUID REFERENCES apps(id),
    scope_type TEXT NOT NULL
        CHECK (scope_type IN ('domain', 'node_type', 'dataset_tag', 'all')),
    scope_value TEXT,
    permission TEXT NOT NULL
        CHECK (permission IN ('read', 'search', 'write', 'admin')),
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'revoked', 'expired')),
    expires_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ,
    CONSTRAINT chk_scope_value CHECK (
        (scope_type = 'all' AND scope_value IS NULL)
        OR (scope_type <> 'all' AND scope_value IS NOT NULL)
    )
);

CREATE INDEX idx_grant_grantee
    ON access_grants(grantee_tenant_id, grantee_app_id, status);

CREATE INDEX idx_grant_grantor
    ON access_grants(grantor_tenant_id, grantor_app_id, status);

CREATE INDEX idx_grant_scope
    ON access_grants(scope_type, scope_value)
    WHERE status = 'active';

-- PostgreSQL requires all partition key columns to be included in any PRIMARY KEY
-- on a partitioned table. We use (id, created_at) as the composite PK so that
-- PARTITION BY RANGE (created_at) is satisfied.
CREATE TABLE access_audit_log (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    requester_tenant_id UUID NOT NULL,
    requester_app_id UUID NOT NULL,
    action TEXT NOT NULL
        CHECK (action IN ('read', 'search', 'write', 'grant_created', 'grant_revoked')),
    resource_domain_id TEXT,
    resource_owner_tenant_id UUID,
    allowed BOOLEAN NOT NULL,
    reason TEXT NOT NULL,
    request_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);
