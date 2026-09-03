-- Migration: 0053_memory_salience
-- Task: TASK-INTEL-007 / SOL-INTEL-004 (Memory Decay & Salience Eviction)
-- Description: Add salience scoring columns to agent_memories table

ALTER TABLE agent_memories
    ADD COLUMN IF NOT EXISTS salience_score   DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS access_count     INTEGER          NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_accessed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS archived         BOOLEAN          NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS archived_at      TIMESTAMPTZ;

-- Index for decay sweep (find low-salience, non-archived memories)
CREATE INDEX IF NOT EXISTS idx_agent_memories_salience
    ON agent_memories (salience_score ASC)
    WHERE archived = false;

-- Index for archival queries
CREATE INDEX IF NOT EXISTS idx_agent_memories_archived
    ON agent_memories (tenant_id, archived)
    WHERE archived = true;

COMMENT ON COLUMN agent_memories.salience_score IS
    'Salience = importance × recency × frequency. Decays via half-life formula. Memories with salience < 0.05 are archived.';
