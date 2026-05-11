-- Migration 001: Create ingestion tables
-- Up

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS datasets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    file_count INT NOT NULL DEFAULT 0,
    total_size_bytes BIGINT NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE TABLE IF NOT EXISTS data_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_id UUID NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    tenant_id VARCHAR(255) NOT NULL,
    source VARCHAR(20) NOT NULL,
    filename VARCHAR(500) DEFAULT '',
    mime_type VARCHAR(100) DEFAULT '',
    raw_text TEXT DEFAULT '',
    storage_path TEXT DEFAULT '',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    file_hash VARCHAR(64) DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_datasets_tenant ON datasets(tenant_id);
CREATE INDEX IF NOT EXISTS idx_data_items_dataset ON data_items(dataset_id);
CREATE INDEX IF NOT EXISTS idx_data_items_hash ON data_items(file_hash, dataset_id);
CREATE INDEX IF NOT EXISTS idx_data_items_tenant ON data_items(tenant_id);

-- Row Level Security
ALTER TABLE datasets ENABLE ROW LEVEL SECURITY;
ALTER TABLE data_items ENABLE ROW LEVEL SECURITY;

-- RLS policies (applied when app.tenant_id is set)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE policyname = 'datasets_tenant_rls') THEN
        CREATE POLICY datasets_tenant_rls ON datasets
            USING (tenant_id = current_setting('app.tenant_id', true));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE policyname = 'data_items_tenant_rls') THEN
        CREATE POLICY data_items_tenant_rls ON data_items
            USING (tenant_id = current_setting('app.tenant_id', true));
    END IF;
END $$;
