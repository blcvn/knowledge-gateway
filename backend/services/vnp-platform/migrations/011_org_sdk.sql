-- migrations/011_org_sdk.sql
-- TASK-BE-013: sdk_api_keys + webhooks tables cho Org & SDK Handlers
-- api_keys: Console SDK keys (vnp_xxx) — khác với bảng api_keys trong migrations/001
-- webhooks: Event delivery endpoints

-- +migrate Up

-- SDK API Keys (tenant-scoped, xem xét conflict với api_keys trong migration 001)
-- Dùng tên mới sdk_api_keys để tránh conflict
CREATE TABLE IF NOT EXISTS sdk_api_keys (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    VARCHAR(64) NOT NULL,
    name         TEXT        NOT NULL,
    key_hash     TEXT        UNIQUE NOT NULL,   -- SHA-256 của raw key
    prefix       TEXT        NOT NULL,           -- 8 ký tự đầu để hiển thị
    permissions  TEXT[]      DEFAULT '{}',
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked      BOOLEAN     NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sdk_keys_tenant ON sdk_api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sdk_keys_hash   ON sdk_api_keys(key_hash);

-- Webhooks: Event delivery endpoints per tenant
CREATE TABLE IF NOT EXISTS webhooks (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    VARCHAR(64) NOT NULL,
    url          TEXT        NOT NULL,
    events       TEXT[]      NOT NULL DEFAULT '{}',
    secret_hash  TEXT,                             -- HMAC signing key hash (SHA-256)
    status       TEXT        NOT NULL DEFAULT 'active',  -- active|paused|failed
    success_rate FLOAT       NOT NULL DEFAULT 100.0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhooks_tenant ON webhooks(tenant_id);

-- +migrate Down
DROP TABLE IF EXISTS webhooks;
DROP TABLE IF EXISTS sdk_api_keys;
