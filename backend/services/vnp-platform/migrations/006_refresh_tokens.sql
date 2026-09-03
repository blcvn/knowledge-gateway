-- migrations/006_refresh_tokens.sql
-- Refresh token rotation table for Console Auth (TASK-BE-001)
-- Lưu trữ SHA-256 hash của refresh token (không lưu raw token)

-- +migrate Up
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES console_users(id) ON DELETE CASCADE,
    token_hash    TEXT        UNIQUE NOT NULL,  -- SHA-256 của raw refresh token
    expires_at    TIMESTAMPTZ NOT NULL,
    revoked       BOOLEAN     NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id    ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);

-- +migrate Down
DROP TABLE IF EXISTS refresh_tokens;
