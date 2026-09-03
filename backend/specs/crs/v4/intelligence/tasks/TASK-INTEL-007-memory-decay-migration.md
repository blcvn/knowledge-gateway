# TASK-INTEL-007 — Database schema: add salience_score, access_count, archived columns

| Field | Value |
|---|---|
| **Task ID** | TASK-INTEL-007 |
| **Wave** | 2 |
| **Solution** | [SOL-INTEL-004](../solutions/SOL-INTEL-004-Memory-Decay-Eviction.md) |
| **Component** | `deployment/dev/migrations/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 1h |

---

## Mục tiêu

Database schema: add salience_score, access_count, archived columns

---

## Công việc cụ thể

### `deployment/dev/migrations/0043_memory_salience.sql` [NEW]

```sql
-- +migrate Up

-- Add to agent_memories table (memory-service)
ALTER TABLE agent_memories
    ADD COLUMN IF NOT EXISTS salience_score  FLOAT   NOT NULL DEFAULT 0.5,
    ADD COLUMN IF NOT EXISTS access_count    INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS archived        BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS importance_score FLOAT  NOT NULL DEFAULT 0.5;

CREATE INDEX idx_memories_salience ON agent_memories(tenant_id, salience_score)
    WHERE archived = false;

-- For sm-memory service
ALTER TABLE sm_memories
    ADD COLUMN IF NOT EXISTS superseded_by  TEXT,    -- ID of newer memory
    ADD COLUMN IF NOT EXISTS salience_score  FLOAT   NOT NULL DEFAULT 0.5,
    ADD COLUMN IF NOT EXISTS archived        BOOLEAN NOT NULL DEFAULT false;

-- +migrate Down
ALTER TABLE agent_memories DROP COLUMN IF EXISTS salience_score;
ALTER TABLE agent_memories DROP COLUMN IF EXISTS access_count;
ALTER TABLE agent_memories DROP COLUMN IF EXISTS archived;
ALTER TABLE agent_memories DROP COLUMN IF EXISTS importance_score;
ALTER TABLE sm_memories DROP COLUMN IF EXISTS superseded_by;
ALTER TABLE sm_memories DROP COLUMN IF EXISTS salience_score;
ALTER TABLE sm_memories DROP COLUMN IF EXISTS archived;
```

---

## Acceptance Criteria

- [ ] salience_score, access_count, archived columns added
- [ ] Index on (tenant_id, salience_score) WHERE archived=false
- [ ] Migration runs without error
- [ ] DOWN migration clean

## Files

```
deployment/dev/migrations/0043_memory_salience.sql  [NEW]
```
