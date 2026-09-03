-- search-service SQL migration (MERGE-P2-T4)
-- Covers: external data connectors + sync job tracking

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ── Connectors (sm-connector) ─────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS search_connectors (
    id             TEXT PRIMARY KEY,
    tenant_id      TEXT NOT NULL,
    name           TEXT NOT NULL,
    type           TEXT NOT NULL,
    config         JSONB NOT NULL DEFAULT '{}',
    sync_frequency TEXT NOT NULL DEFAULT 'manual',
    status         TEXT NOT NULL DEFAULT 'active',
    last_sync_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_connectors_tenant ON search_connectors(tenant_id);

CREATE TABLE IF NOT EXISTS search_sync_jobs (
    id            TEXT PRIMARY KEY,
    connector_id  TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',
    items_synced  INT NOT NULL DEFAULT 0,
    started_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ,
    error         TEXT,
    FOREIGN KEY (connector_id) REFERENCES search_connectors(id)
);
CREATE INDEX IF NOT EXISTS idx_sync_jobs_connector ON search_sync_jobs(connector_id);
