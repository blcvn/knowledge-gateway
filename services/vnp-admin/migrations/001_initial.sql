-- vnp-admin initial schema
-- Reference: services/vnp-admin/specs/tdd.md §2, docs/data-model.md

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants
CREATE TABLE IF NOT EXISTS tenants (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       VARCHAR(255) NOT NULL UNIQUE,
    plan       VARCHAR(50) NOT NULL DEFAULT 'free',
    config     JSONB NOT NULL DEFAULT '{}',
    active     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_name ON tenants(name);
CREATE INDEX idx_tenants_plan ON tenants(plan);

-- API Keys
CREATE TABLE IF NOT EXISTS api_keys (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    key_hash   VARCHAR(64) NOT NULL UNIQUE,  -- SHA-256 hex
    key_prefix VARCHAR(12) NOT NULL,          -- First 12 chars for identification
    scope      VARCHAR(20) NOT NULL DEFAULT 'read',
    rate_limit INTEGER NOT NULL DEFAULT 0,
    active     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_active ON api_keys(active) WHERE active = true;

-- Users
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

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_email ON users(tenant_id, email);
