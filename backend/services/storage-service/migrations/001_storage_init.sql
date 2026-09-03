-- migrations/001_storage_init.sql
-- storage-service initial schema (MERGE-P1-T4)
-- Absorbed from: ov-session, ov-resource

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ═══════════════════════════════════════════════
-- ov-session: Chat sessions and message history
-- ═══════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS ov_sessions (
    id         TEXT PRIMARY KEY,
    tenant_id  TEXT NOT NULL,
    base_dir   TEXT NOT NULL DEFAULT '/',
    status     TEXT NOT NULL DEFAULT 'open',   -- 'open' | 'committed'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ov_sessions_tenant ON ov_sessions(tenant_id);

CREATE TABLE IF NOT EXISTS ov_messages (
    id         TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES ov_sessions(id) ON DELETE CASCADE,
    role       TEXT NOT NULL,                  -- 'user' | 'assistant' | 'system'
    content    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ov_messages_session ON ov_messages(session_id);

-- ═══════════════════════════════════════════════
-- ov-resource: Ingested external resources
-- ═══════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS ov_resources (
    id         TEXT PRIMARY KEY,
    tenant_id  TEXT NOT NULL,
    uri        TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT 'document',   -- 'document' | 'image' | 'code' | 'web'
    status     TEXT NOT NULL DEFAULT 'pending',    -- 'pending' | 'processing' | 'indexed' | 'failed'
    embed_path TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ov_resources_tenant ON ov_resources(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ov_resources_status ON ov_resources(status);
