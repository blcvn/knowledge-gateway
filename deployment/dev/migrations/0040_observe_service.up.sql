-- Observe Service: Sessions + Observations
-- Migration: 0040_observe_service.up.sql

CREATE TABLE IF NOT EXISTS agent_sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           TEXT NOT NULL,
    project             TEXT NOT NULL DEFAULT '',
    cwd                 TEXT,
    model               TEXT,
    agent_id            TEXT,
    status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'abandoned')),
    summary             TEXT,
    first_prompt        TEXT,
    observation_count   INTEGER NOT NULL DEFAULT 0,
    tags                TEXT[] NOT NULL DEFAULT '{}',
    started_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at            TIMESTAMPTZ,
    last_active_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS raw_observations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID NOT NULL REFERENCES agent_sessions(id) ON DELETE CASCADE,
    hook_type   TEXT NOT NULL,
    tool_name   TEXT,
    tool_input  JSONB,
    tool_output JSONB,
    user_prompt TEXT,
    assistant_response TEXT,
    agent_id    TEXT,
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS compressed_observations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID NOT NULL REFERENCES agent_sessions(id) ON DELETE CASCADE,
    source_id   UUID REFERENCES raw_observations(id),
    obs_type    TEXT NOT NULL,
    title       TEXT,
    subtitle    TEXT,
    narrative   TEXT,
    facts       TEXT[] NOT NULL DEFAULT '{}',
    concepts    TEXT[] NOT NULL DEFAULT '{}',
    files       TEXT[] NOT NULL DEFAULT '{}',
    importance  FLOAT NOT NULL DEFAULT 0.5,
    agent_id    TEXT,
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Deduplication cache (prevents duplicate observations within a session)
CREATE TABLE IF NOT EXISTS dedup_cache (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   UUID NOT NULL,
    content_hash TEXT NOT NULL UNIQUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_sessions_tenant ON agent_sessions(tenant_id, status, project);
CREATE INDEX IF NOT EXISTS idx_agent_sessions_started ON agent_sessions(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_raw_obs_session ON raw_observations(session_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_compressed_obs_session ON compressed_observations(session_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_dedup_hash ON dedup_cache(content_hash, session_id);
