-- kg-service PostgreSQL migrations
-- Episode storage with pgvector embeddings (MERGE-P2-T1)

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS vector;

-- Episodes: text + vector embeddings (pgvector fallback when Neo4j unavailable)
CREATE TABLE IF NOT EXISTS kg_episodes (
    uuid       UUID PRIMARY KEY,
    tenant_id  TEXT NOT NULL,
    name       TEXT NOT NULL,
    content    TEXT NOT NULL,
    source     TEXT NOT NULL DEFAULT 'message',
    source_id  TEXT,
    embedding  vector(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_kg_episodes_tenant ON kg_episodes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_kg_episodes_embedding ON kg_episodes
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Cognee dataset metadata tracking (MERGE-P2-T2)
CREATE TABLE IF NOT EXISTS cognee_datasets (
    id         TEXT PRIMARY KEY,
    tenant_id  TEXT NOT NULL,
    name       TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'empty',
    data_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_cognee_datasets_tenant ON cognee_datasets(tenant_id);

-- Cognee job tracking
CREATE TABLE IF NOT EXISTS cognee_jobs (
    job_id     TEXT PRIMARY KEY,
    dataset_id TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'pending',
    progress   FLOAT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    done_at    TIMESTAMPTZ
);
