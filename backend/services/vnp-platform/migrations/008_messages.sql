-- migrations/008_messages.sql
-- Session messages table (TASK-BE-004)
-- Lưu các tin nhắn trong từng session

-- +migrate Up
CREATE TABLE IF NOT EXISTS messages (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id     UUID        NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    tenant_id      UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role           TEXT        NOT NULL,   -- user | assistant | system | tool
    content        TEXT        NOT NULL,
    memory_sources TEXT[]      DEFAULT '{}',  -- ["graphiti:ep_abc", "memobase:prof_xyz"]
    tokens         INT         DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_messages_tenant_id  ON messages(tenant_id);

-- +migrate Down
DROP TABLE IF EXISTS messages;
