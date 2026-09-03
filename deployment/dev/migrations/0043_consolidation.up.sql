-- Consolidation: Session Summaries, Procedural Memories, Lessons, Insights
-- Migration: 0043_consolidation.up.sql

CREATE TABLE IF NOT EXISTS session_summaries (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id        UUID NOT NULL REFERENCES agent_sessions(id) ON DELETE CASCADE,
    tenant_id         TEXT NOT NULL,
    title             TEXT,
    narrative         TEXT,
    key_decisions     TEXT[] NOT NULL DEFAULT '{}',
    files_modified    TEXT[] NOT NULL DEFAULT '{}',
    concepts          TEXT[] NOT NULL DEFAULT '{}',
    observation_count INTEGER NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS procedural_memories (
    id                TEXT PRIMARY KEY,
    tenant_id         TEXT NOT NULL,
    project           TEXT NOT NULL DEFAULT '',
    name              TEXT,
    steps             TEXT[] NOT NULL DEFAULT '{}',
    step_hash         TEXT NOT NULL UNIQUE,
    trigger_condition TEXT,
    expected_outcome  TEXT,
    frequency         INTEGER NOT NULL DEFAULT 0,
    confidence        FLOAT NOT NULL DEFAULT 0.5,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS lessons (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    TEXT NOT NULL,
    project      TEXT NOT NULL DEFAULT '',
    content      TEXT NOT NULL,
    confidence   FLOAT NOT NULL DEFAULT 0.7,
    source       TEXT,
    categories   TEXT[] NOT NULL DEFAULT '{}',
    access_count INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS insights (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    content     TEXT NOT NULL,
    lesson_ids  TEXT[] NOT NULL DEFAULT '{}',
    confidence  FLOAT NOT NULL DEFAULT 0.6,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_session_summaries_tenant ON session_summaries(tenant_id);
CREATE INDEX IF NOT EXISTS idx_procedural_tenant ON procedural_memories(tenant_id, project);
CREATE INDEX IF NOT EXISTS idx_lessons_confidence ON lessons(tenant_id, confidence DESC);
CREATE INDEX IF NOT EXISTS idx_insights_tenant ON insights(tenant_id);
