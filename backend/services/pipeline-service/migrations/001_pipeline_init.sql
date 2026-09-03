-- pipeline-service SQL migrations (MERGE-P3-T1)
-- Absorbed from: vnp-pipelines, ba-knowledge-service, ba-knowledge-worker

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ── Pipeline Jobs ──────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS pipeline_jobs (
    id           TEXT PRIMARY KEY,
    engine       TEXT NOT NULL,
    type         TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    payload      JSONB NOT NULL DEFAULT '{}',
    result       JSONB,
    error        TEXT,
    priority     INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_pipeline_jobs_engine ON pipeline_jobs(engine, status);
CREATE INDEX IF NOT EXISTS idx_pipeline_jobs_status ON pipeline_jobs(status);
CREATE INDEX IF NOT EXISTS idx_pipeline_jobs_created ON pipeline_jobs(created_at DESC);

-- ── Pipeline Workers ───────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS pipeline_workers (
    id        TEXT PRIMARY KEY,
    engine    TEXT NOT NULL,
    status    TEXT NOT NULL DEFAULT 'idle',
    job_id    TEXT,
    last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pipeline_workers_engine ON pipeline_workers(engine, last_seen DESC);

-- ── Knowledge PRDs (from ba-knowledge-service) ─────────────────────────────

CREATE TABLE IF NOT EXISTS knowledge_prds (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    content    TEXT NOT NULL,
    tags       TEXT[] NOT NULL DEFAULT '{}',
    status     TEXT NOT NULL DEFAULT 'draft',
    tenant_id  TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_prds_tenant ON knowledge_prds(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_prds_status ON knowledge_prds(status);

-- ── Knowledge Outlines ─────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS knowledge_outlines (
    id         TEXT PRIMARY KEY,
    prd_id     TEXT NOT NULL UNIQUE REFERENCES knowledge_prds(id) ON DELETE CASCADE,
    sections   JSONB NOT NULL DEFAULT '[]',
    status     TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
