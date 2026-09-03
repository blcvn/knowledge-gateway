-- obs-service SQL migrations (MERGE-P3-T2)
-- Absorbed from: vnp-observability, vnp-infra, sm-engine

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ── Metrics Timeseries ─────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS obs_metrics (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    value       FLOAT NOT NULL,
    labels      JSONB NOT NULL DEFAULT '{}',
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_obs_metrics_name ON obs_metrics(name, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_obs_metrics_time ON obs_metrics(recorded_at DESC);

-- ── Error Tracking ─────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS obs_errors (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service    TEXT NOT NULL,
    type       TEXT NOT NULL,
    message    TEXT NOT NULL,
    stack      TEXT,
    count      INT NOT NULL DEFAULT 1,
    first_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(service, type, message)
);
CREATE INDEX IF NOT EXISTS idx_obs_errors_service ON obs_errors(service, last_seen DESC);

-- ── Cost Tracking ──────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS obs_costs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service      TEXT NOT NULL,
    period       TEXT NOT NULL,    -- "2026-06-11:hour" | "2026-06-11:day"
    llm_tokens   BIGINT NOT NULL DEFAULT 0,
    embed_tokens BIGINT NOT NULL DEFAULT 0,
    storage_mb   BIGINT NOT NULL DEFAULT 0,
    est_cost_usd NUMERIC(10,4) NOT NULL DEFAULT 0,
    recorded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(service, period)
);
CREATE INDEX IF NOT EXISTS idx_obs_costs_period ON obs_costs(service, period);
