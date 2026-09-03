-- migrations/007_sessions.sql
-- Console sessions table (TASK-BE-004)
-- Lưu các AI memory sessions được quản lý qua Console UI

-- +migrate Up
CREATE TABLE IF NOT EXISTS sessions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id      TEXT        NOT NULL,        -- External user ID (từ SDK)
    agent_id     TEXT,
    engine       TEXT        NOT NULL DEFAULT 'zep',
    status       TEXT        NOT NULL DEFAULT 'active',
                                              -- active | completed | failed
    started_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at     TIMESTAMPTZ,
    metadata     JSONB       DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_tenant_status  ON sessions(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id        ON sessions(tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_created_at     ON sessions(tenant_id, created_at DESC);

-- Full-text search trên user_id và agent_id
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS tsv TSVECTOR
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(user_id, '') || ' ' || coalesce(agent_id, ''))) STORED;
CREATE INDEX IF NOT EXISTS idx_sessions_fts ON sessions USING GIN(tsv);

-- +migrate Down
DROP TABLE IF EXISTS sessions;
