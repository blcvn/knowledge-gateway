-- Migration: 0052_webhooks
-- Task: TASK-009 / SOL-002 (Webhook entity)
-- Description: Create webhooks table for SDK event subscriptions

CREATE TABLE IF NOT EXISTS webhooks (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID         NOT NULL,
    url          TEXT         NOT NULL,
    events       TEXT[]       NOT NULL DEFAULT '{}',
    status       TEXT         NOT NULL DEFAULT 'active'
                              CHECK (status IN ('active', 'paused', 'failed')),
    secret_hash  TEXT,                        -- SHA-256 of signing secret (never plaintext)
    success_rate DOUBLE PRECISION DEFAULT 1.0,
    last_error   TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhooks_tenant     ON webhooks(tenant_id);
CREATE INDEX idx_webhooks_status     ON webhooks(tenant_id, status);
