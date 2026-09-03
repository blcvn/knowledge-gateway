-- migrations/010_observability.sql
-- TASK-BE-010: error_aggregates table cho Observability Handler
-- Lưu lỗi theo dạng aggregate để tránh bùng nổ row khi lỗi lặp lại nhiều

-- +migrate Up
CREATE TABLE IF NOT EXISTS error_aggregates (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       VARCHAR(64) NOT NULL,
    service         TEXT        NOT NULL,
    message         TEXT        NOT NULL,
    message_hash    TEXT        NOT NULL,   -- SHA-256/MD5 để group identical errors
    count           INT         NOT NULL DEFAULT 1,
    last_occurrence TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    stack           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, message_hash)
);

CREATE INDEX IF NOT EXISTS idx_error_agg_tenant ON error_aggregates(tenant_id, last_occurrence DESC);
CREATE INDEX IF NOT EXISTS idx_error_agg_service ON error_aggregates(service);

-- Cost tracking per LLM call
CREATE TABLE IF NOT EXISTS llm_cost_events (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   VARCHAR(64) NOT NULL,
    model       TEXT        NOT NULL,
    input_tokens  INT       NOT NULL DEFAULT 0,
    output_tokens INT       NOT NULL DEFAULT 0,
    cost_usd    FLOAT       NOT NULL DEFAULT 0,
    service     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_llm_cost_tenant ON llm_cost_events(tenant_id, created_at DESC);

-- +migrate Down
DROP TABLE IF EXISTS llm_cost_events;
DROP TABLE IF EXISTS error_aggregates;
