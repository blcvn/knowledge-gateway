-- vnp-platform initial schema (consolidated)
-- Includes: admin, event, auth, analytics, project domains
-- Reference: services/vnp-platform/specs/tdd.md

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";

-- ============ ADMIN domain ============

CREATE TABLE IF NOT EXISTS tenants (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       VARCHAR(255) NOT NULL UNIQUE,
    slug       VARCHAR(128) NOT NULL UNIQUE,
    tier       VARCHAR(20) NOT NULL DEFAULT 'free',
    config     JSONB NOT NULL DEFAULT '{}',
    active     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    key_hash    VARCHAR(64) NOT NULL UNIQUE,
    key_prefix  VARCHAR(12) NOT NULL,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    active      BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS users (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email      VARCHAR(255) NOT NULL,
    name       VARCHAR(255) NOT NULL DEFAULT '',
    role       VARCHAR(20) NOT NULL DEFAULT 'member',
    metadata   JSONB NOT NULL DEFAULT '{}',
    active     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);

-- ============ EVENT domain ============

CREATE TABLE IF NOT EXISTS user_events (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID NOT NULL,
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source     VARCHAR(20) NOT NULL,
    content    TEXT NOT NULL,
    tags       TEXT[] NOT NULL DEFAULT '{}',
    embedding  vector(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    invalid_at TIMESTAMPTZ
);

CREATE INDEX idx_platform_events_tenant ON user_events(tenant_id);
CREATE INDEX idx_platform_events_user ON user_events(tenant_id, user_id);
CREATE INDEX idx_platform_events_tags ON user_events USING GIN(tags);

-- ============ ANALYTICS domain ============

CREATE TABLE IF NOT EXISTS usage_records (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    period      VARCHAR(7) NOT NULL,  -- "2026-05"
    api_calls   BIGINT NOT NULL DEFAULT 0,
    storage_mb  BIGINT NOT NULL DEFAULT 0,
    llm_tokens  BIGINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, period)
);

-- ============ PROJECT domain ============

CREATE TABLE IF NOT EXISTS spaces (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_spaces_tenant ON spaces(tenant_id);

-- Common indexes
CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX idx_users_tenant ON users(tenant_id);
