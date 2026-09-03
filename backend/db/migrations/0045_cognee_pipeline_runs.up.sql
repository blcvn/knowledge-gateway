-- Migration: 0045_cognee_pipeline_runs.up.sql
-- Purpose: Track async cognify and memify pipeline job status
-- CR: CR-COGNEE-001 (Memify), CR-COGNEE-006 (Custom Pipelines)

CREATE TABLE IF NOT EXISTS cognee_pipeline_runs (
    id          TEXT        PRIMARY KEY,
    dataset_id  TEXT        NOT NULL,
    tenant_id   TEXT        NOT NULL,
    type        TEXT        NOT NULL,    -- 'cognify' | 'memify'
    status      TEXT        NOT NULL DEFAULT 'QUEUED',  -- QUEUED | RUNNING | COMPLETED | FAILED
    new_nodes   INT         NOT NULL DEFAULT 0,
    new_edges   INT         NOT NULL DEFAULT 0,
    error       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cognee_pipeline_runs_dataset
    ON cognee_pipeline_runs(dataset_id, tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_cognee_pipeline_runs_tenant
    ON cognee_pipeline_runs(tenant_id, created_at DESC);

COMMENT ON TABLE cognee_pipeline_runs IS
    'Tracks background cognify and memify pipeline job status for CR-COGNEE-001/006';

COMMENT ON COLUMN cognee_pipeline_runs.type IS '''cognify'' for StartCognify jobs, ''memify'' for Memify jobs';
COMMENT ON COLUMN cognee_pipeline_runs.status IS 'QUEUED | RUNNING | COMPLETED | FAILED';
