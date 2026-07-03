CREATE TABLE IF NOT EXISTS kg_graph_identifiers (
    identifier_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_tenant_id UUID NOT NULL REFERENCES tenants(id),
    owner_app_id UUID REFERENCES apps(id),
    graph_scope TEXT NOT NULL,
    external_project_id TEXT,
    status TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'ARCHIVED', 'DELETED')),
    head_version_number BIGINT NOT NULL DEFAULT 0,
    head_version_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_tenant_id, owner_app_id, graph_scope)
);

CREATE INDEX IF NOT EXISTS idx_kg_graph_identifiers_owner_scope
    ON kg_graph_identifiers(owner_tenant_id, owner_app_id, graph_scope);

CREATE TABLE IF NOT EXISTS kg_graph_versions (
    version_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identifier_id UUID NOT NULL REFERENCES kg_graph_identifiers(identifier_id) ON DELETE CASCADE,
    version_number BIGINT NOT NULL,
    reference_id TEXT NOT NULL,
    parent_version_id UUID,
    storage_class TEXT NOT NULL DEFAULT 'ONLINE'
        CHECK (storage_class IN ('ONLINE', 'OFFLINE')),
    version_status TEXT NOT NULL DEFAULT 'SEALED'
        CHECK (version_status IN ('PENDING_ENTITIES', 'SEALED', 'FAILED_FINALIZATION')),
    change_summary TEXT,
    manifest_locator TEXT,
    sealed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (identifier_id, version_number),
    UNIQUE (identifier_id, reference_id)
);

CREATE INDEX IF NOT EXISTS idx_kg_graph_versions_identifier_created
    ON kg_graph_versions(identifier_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_kg_graph_versions_identifier_status
    ON kg_graph_versions(identifier_id, version_status, storage_class);

CREATE TABLE IF NOT EXISTS kg_graph_version_entities (
    version_id UUID NOT NULL REFERENCES kg_graph_versions(version_id) ON DELETE CASCADE,
    entity_kind TEXT NOT NULL
        CHECK (entity_kind IN ('node', 'relationship', 'embeddable_relationship')),
    entity_id UUID NOT NULL,
    change_kind TEXT NOT NULL
        CHECK (change_kind IN ('UPSERT', 'DELETE')),
    source_version BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (version_id, entity_kind, entity_id)
);

CREATE INDEX IF NOT EXISTS idx_kg_graph_version_entities_entity
    ON kg_graph_version_entities(entity_kind, entity_id);

CREATE TABLE IF NOT EXISTS kg_graph_projection_heads (
    identifier_id UUID NOT NULL REFERENCES kg_graph_identifiers(identifier_id) ON DELETE CASCADE,
    backend_kind TEXT NOT NULL,
    backend_name TEXT NOT NULL DEFAULT '',
    applied_version_id UUID,
    applied_version_number BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (identifier_id, backend_kind, backend_name)
);

CREATE INDEX IF NOT EXISTS idx_kg_graph_projection_heads_backend
    ON kg_graph_projection_heads(backend_kind, backend_name, applied_version_number);
