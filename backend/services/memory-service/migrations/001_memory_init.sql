-- memory-service SQL migration (MERGE-P2-T3)
-- Covers: memobase blobs + profiles, SM memories + documents

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS vector;

-- ── Memobase ─────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS memory_blobs (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    tenant_id  TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT 'conversation',
    content    TEXT NOT NULL,
    metadata   JSONB NOT NULL DEFAULT '{}',
    embedding  vector(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_blobs_user ON memory_blobs(user_id, tenant_id);
CREATE INDEX IF NOT EXISTS idx_blobs_tenant ON memory_blobs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_blobs_embedding ON memory_blobs
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE TABLE IF NOT EXISTS memory_profiles (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    TEXT NOT NULL,
    tenant_id  TEXT NOT NULL,
    key        TEXT NOT NULL,
    value      TEXT NOT NULL,
    category   TEXT NOT NULL DEFAULT 'fact',
    score      FLOAT NOT NULL DEFAULT 1.0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, tenant_id, key)
);
CREATE INDEX IF NOT EXISTS idx_profiles_user ON memory_profiles(user_id, tenant_id);

-- ── Supermemory ───────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS sm_memories (
    id         TEXT PRIMARY KEY,
    tenant_id  TEXT NOT NULL,
    content    TEXT NOT NULL,
    tags       TEXT[] NOT NULL DEFAULT '{}',
    metadata   JSONB NOT NULL DEFAULT '{}',
    embedding  vector(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sm_memories_tenant ON sm_memories(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sm_memories_embedding ON sm_memories
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE TABLE IF NOT EXISTS sm_documents (
    id         TEXT PRIMARY KEY,
    tenant_id  TEXT NOT NULL,
    title      TEXT NOT NULL,
    content    TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT 'markdown',
    url        TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sm_documents_tenant ON sm_documents(tenant_id);
