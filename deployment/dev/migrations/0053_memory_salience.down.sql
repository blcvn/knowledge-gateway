-- Revert salience columns
ALTER TABLE agent_memories
    DROP COLUMN IF EXISTS salience_score,
    DROP COLUMN IF EXISTS access_count,
    DROP COLUMN IF EXISTS last_accessed_at,
    DROP COLUMN IF EXISTS archived,
    DROP COLUMN IF EXISTS archived_at;

DROP INDEX IF EXISTS idx_agent_memories_salience;
DROP INDEX IF EXISTS idx_agent_memories_archived;
