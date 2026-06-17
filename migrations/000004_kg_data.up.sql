CREATE TABLE kg_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_type TEXT NOT NULL,
    domain_id TEXT NOT NULL REFERENCES domains(id),
    owner_tenant_id UUID NOT NULL REFERENCES tenants(id),
    owner_app_id UUID REFERENCES apps(id),
    visibility TEXT NOT NULL DEFAULT 'private'
        CHECK (visibility IN ('public', 'tenant_shared', 'private')),
    properties JSONB NOT NULL,
    domain_version INT NOT NULL,
    external_ref TEXT UNIQUE,
    status_value TEXT,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_kg_nodes_domain
    ON kg_nodes(domain_id)
    WHERE NOT is_deleted;

CREATE INDEX idx_kg_nodes_owner
    ON kg_nodes(owner_tenant_id, owner_app_id);

CREATE INDEX idx_kg_nodes_type
    ON kg_nodes(node_type, domain_id);

CREATE INDEX idx_kg_nodes_ref
    ON kg_nodes(external_ref)
    WHERE external_ref IS NOT NULL;

ALTER TABLE kg_nodes ENABLE ROW LEVEL SECURITY;

CREATE POLICY kg_nodes_isolation ON kg_nodes
    USING (
        owner_tenant_id = current_setting('app.tenant_id')::uuid
        OR owner_tenant_id = '00000000-0000-0000-0000-000000000000'
        OR EXISTS (
            SELECT 1
            FROM access_grants g
            WHERE g.grantee_tenant_id = current_setting('app.tenant_id')::uuid
              AND (g.grantee_app_id = current_setting('app.app_id')::uuid OR g.grantee_app_id IS NULL)
              AND g.status = 'active'
              AND (g.expires_at IS NULL OR g.expires_at > now())
              AND (
                  g.scope_type = 'all'
                  OR (g.scope_type = 'domain' AND g.scope_value = kg_nodes.domain_id)
                  OR (g.scope_type = 'node_type' AND g.scope_value = kg_nodes.node_type)
              )
        )
    );

CREATE TABLE kg_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rel_type TEXT NOT NULL,
    from_node_id UUID NOT NULL REFERENCES kg_nodes(id),
    to_node_id UUID NOT NULL REFERENCES kg_nodes(id),
    domain_id TEXT NOT NULL REFERENCES domains(id),
    owner_tenant_id UUID NOT NULL REFERENCES tenants(id),
    owner_app_id UUID REFERENCES apps(id),
    properties JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE kg_relationships ENABLE ROW LEVEL SECURITY;

CREATE POLICY kg_relationships_isolation ON kg_relationships
    USING (
        owner_tenant_id = current_setting('app.tenant_id')::uuid
        OR owner_tenant_id = '00000000-0000-0000-0000-000000000000'
        OR EXISTS (
            SELECT 1
            FROM access_grants g
            WHERE g.grantee_tenant_id = current_setting('app.tenant_id')::uuid
              AND (g.grantee_app_id = current_setting('app.app_id')::uuid OR g.grantee_app_id IS NULL)
              AND g.status = 'active'
              AND (g.expires_at IS NULL OR g.expires_at > now())
              AND (
                  g.scope_type = 'all'
                  OR (g.scope_type = 'domain' AND g.scope_value = kg_relationships.domain_id)
              )
        )
    );

CREATE TABLE kg_outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type TEXT NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'PROCESSING', 'DONE', 'FAILED')),
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_pending
    ON kg_outbox_events(status, created_at)
    WHERE status = 'PENDING';
