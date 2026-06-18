ALTER TABLE kg_relationships
    ADD COLUMN IF NOT EXISTS domain_version INT NOT NULL DEFAULT 1;

ALTER TABLE kg_vector_documents
    ADD COLUMN IF NOT EXISTS sync_version BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS kg_projection_versions (
    entity_id TEXT NOT NULL,
    entity_kind TEXT NOT NULL,
    source_version BIGINT NOT NULL,
    source_event_id TEXT NOT NULL,
    source_updated_at TIMESTAMPTZ NOT NULL,
    graph_backend TEXT,
    graph_version BIGINT,
    graph_synced_at TIMESTAMPTZ,
    vector_backend TEXT,
    vector_version BIGINT,
    vector_synced_at TIMESTAMPTZ,
    PRIMARY KEY (entity_id, entity_kind)
);
