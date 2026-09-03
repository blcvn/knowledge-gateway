-- migrations/005_console_users.sql
-- Console auth users table (TASK-BE-001)
-- Lưu ý: Bảng `users` trong 001_initial.sql là tenant-scoped membership.
-- Bảng này (`console_users`) là identity store cho UI Console login (bcrypt + JWT).

-- +migrate Up
CREATE TABLE IF NOT EXISTS console_users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT        NOT NULL,
    email         TEXT        UNIQUE NOT NULL,
    password_hash TEXT,                        -- bcrypt hash, NULL cho SSO-only users
    role          TEXT        NOT NULL DEFAULT 'user',
                                               -- admin | editor | viewer | user
    tenant_id     UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    avatar_url    TEXT,
    is_active     BOOLEAN     NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_console_users_email     ON console_users(email);
CREATE INDEX IF NOT EXISTS idx_console_users_tenant_id ON console_users(tenant_id);

-- Default admin user (password: changeme — thay trước khi production)
-- password_hash là bcrypt của 'changeme'
INSERT INTO console_users (id, name, email, password_hash, role, tenant_id)
SELECT
    gen_random_uuid(),
    'Admin',
    'admin@vnp.io',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj4tbQAj3hm6',
    'admin',
    id
FROM tenants
WHERE slug = 'default'
LIMIT 1
ON CONFLICT (email) DO NOTHING;

-- +migrate Down
DROP TABLE IF EXISTS console_users;
