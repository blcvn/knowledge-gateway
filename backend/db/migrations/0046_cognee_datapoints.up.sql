-- Migration: 0046_cognee_datapoints.up.sql
-- Purpose: Metadata for structured DataPoint ingestion (zero LLM)
-- CR: CR-COGNEE-003 (DataPoint Schema)

CREATE TABLE IF NOT EXISTS cognee_datapoints (
    id           UUID        PRIMARY KEY,
    version      INT         NOT NULL DEFAULT 1,
    dataset_id   UUID        NOT NULL,
    tenant_id    TEXT        NOT NULL,
    type         TEXT        NOT NULL,          -- "Paper" | "Employee" | "Product" etc.
    fields       JSONB       NOT NULL DEFAULT '{}',
    index_fields TEXT[]      NOT NULL DEFAULT '{}',
    node_sets    TEXT[]      NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cognee_datapoints_dataset
    ON cognee_datapoints(dataset_id, tenant_id);

CREATE INDEX IF NOT EXISTS idx_cognee_datapoints_type
    ON cognee_datapoints(tenant_id, type);

CREATE INDEX IF NOT EXISTS idx_cognee_datapoints_node_sets
    ON cognee_datapoints USING GIN (node_sets);

CREATE INDEX IF NOT EXISTS idx_cognee_datapoints_fields
    ON cognee_datapoints USING GIN (fields jsonb_path_ops);

COMMENT ON TABLE cognee_datapoints IS
    'Metadata for structured DataPoints ingested without LLM (CR-COGNEE-003)';

COMMENT ON COLUMN cognee_datapoints.type IS 'DataPoint schema type: Paper, Employee, Product, etc.';
COMMENT ON COLUMN cognee_datapoints.fields IS 'All field values as JSONB (dynamic schema)';
COMMENT ON COLUMN cognee_datapoints.index_fields IS 'Only these fields are embedded into Qdrant vectors';
COMMENT ON COLUMN cognee_datapoints.node_sets IS 'NodeSet tags for memory scoping (CR-COGNEE-002)';
COMMENT ON COLUMN cognee_datapoints.version IS 'Incremented on each upsert (idempotent ingestion)';
