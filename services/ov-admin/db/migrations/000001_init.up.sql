CREATE TABLE IF NOT EXISTS ov_accounts (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(256) NOT NULL,
    namespace_policy JSONB DEFAULT '{}',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS ov_users (
    id VARCHAR(64) PRIMARY KEY,
    account_id VARCHAR(64) NOT NULL REFERENCES ov_accounts(id) ON DELETE CASCADE,
    name VARCHAR(256) NOT NULL,
    role VARCHAR(16) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS ov_agents (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES ov_users(id) ON DELETE CASCADE,
    account_id VARCHAR(64) NOT NULL REFERENCES ov_accounts(id) ON DELETE CASCADE,
    name VARCHAR(256) NOT NULL,
    config JSONB DEFAULT '{}',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS ov_api_key_hashes (
    key_id VARCHAR(64) PRIMARY KEY,
    account_id VARCHAR(64) NOT NULL REFERENCES ov_accounts(id) ON DELETE CASCADE,
    user_id VARCHAR(64),
    role VARCHAR(16) NOT NULL,
    hash VARCHAR(256) NOT NULL,
    prefix VARCHAR(16) NOT NULL,
    label VARCHAR(256),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_users_account ON ov_users(account_id);
CREATE INDEX idx_users_role ON ov_users(account_id, role);
CREATE INDEX idx_agents_user ON ov_agents(user_id);
CREATE INDEX idx_agents_account ON ov_agents(account_id);
CREATE INDEX idx_api_keys_prefix ON ov_api_key_hashes(prefix);
