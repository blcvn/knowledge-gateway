CREATE TABLE IF NOT EXISTS kg_graph_scope_leases (
    owner_tenant_id UUID NOT NULL,
    owner_app_id UUID,
    graph_scope TEXT NOT NULL,
    version_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (owner_tenant_id, owner_app_id, graph_scope)
);
