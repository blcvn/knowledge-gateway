-- migrations/002_auth.sql
-- Auth users table (from sm-auth, MERGE-P1-T1)
-- Separate from admin.users (tenant-scoped) — this is the SSO/JWT identity store

CREATE TABLE IF NOT EXISTS auth_users (
    id               TEXT PRIMARY KEY,
    email            VARCHAR(255) NOT NULL,
    name             VARCHAR(255) NOT NULL DEFAULT '',
    password_hash    TEXT,                         -- bcrypt hash, NULL for OAuth users
    auth_provider    VARCHAR(20) NOT NULL DEFAULT 'email',  -- 'email' | 'google'
    auth_provider_id TEXT,                         -- OAuth sub (e.g. Google sub claim)
    role             VARCHAR(20) NOT NULL DEFAULT 'user',
    active           BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_users_email ON auth_users(email);
CREATE INDEX IF NOT EXISTS idx_auth_users_provider ON auth_users(auth_provider, auth_provider_id)
    WHERE auth_provider_id IS NOT NULL;
