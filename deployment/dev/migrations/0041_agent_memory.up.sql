-- AgentMemory: Long-term memories + Memory Slots
-- Migration: 0041_agent_memory.up.sql

CREATE TABLE IF NOT EXISTS agent_memories (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              TEXT NOT NULL,
    project                TEXT NOT NULL DEFAULT '',
    type                   TEXT NOT NULL CHECK (type IN ('pattern','preference','architecture','bug','workflow','fact')),
    title                  TEXT NOT NULL,
    content                TEXT NOT NULL,
    concepts               TEXT[] NOT NULL DEFAULT '{}',
    files                  TEXT[] NOT NULL DEFAULT '{}',
    session_ids            TEXT[] NOT NULL DEFAULT '{}',
    strength               FLOAT NOT NULL DEFAULT 0.7 CHECK (strength >= 0 AND strength <= 1),
    version                INTEGER NOT NULL DEFAULT 1,
    parent_id              UUID REFERENCES agent_memories(id),
    supersedes             TEXT[] NOT NULL DEFAULT '{}',
    related_ids            TEXT[] NOT NULL DEFAULT '{}',
    source_observation_ids TEXT[] NOT NULL DEFAULT '{}',
    is_latest              BOOLEAN NOT NULL DEFAULT TRUE,
    forget_after           TIMESTAMPTZ,
    agent_id               TEXT,
    flagged_eviction       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS memory_slots (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    project     TEXT NOT NULL DEFAULT '',
    scope       TEXT NOT NULL DEFAULT 'project' CHECK (scope IN ('project','global')),
    label       TEXT NOT NULL,
    content     TEXT NOT NULL DEFAULT '',
    description TEXT,
    size_limit  INTEGER NOT NULL DEFAULT 0,  -- 0 = unlimited
    pinned      BOOLEAN NOT NULL DEFAULT FALSE,
    read_only   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, scope, label)
);

CREATE INDEX IF NOT EXISTS idx_agent_memories_tenant_project ON agent_memories(tenant_id, project, type);
CREATE INDEX IF NOT EXISTS idx_agent_memories_latest ON agent_memories(tenant_id, project, is_latest) WHERE is_latest = TRUE;
CREATE INDEX IF NOT EXISTS idx_agent_memories_forget ON agent_memories(forget_after) WHERE forget_after IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_agent_memories_concepts ON agent_memories USING GIN (concepts);
CREATE INDEX IF NOT EXISTS idx_memory_slots_tenant ON memory_slots(tenant_id, scope);
