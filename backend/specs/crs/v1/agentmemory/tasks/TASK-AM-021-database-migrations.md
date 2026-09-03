# TASK-AM-021 — Database Migrations (All New Tables)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-021 |
| **Wave** | 4 (Governance) |
| **Component** | `deploy/dev/migrations/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-001, SOL-002, SOL-004, SOL-006, SOL-007 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-AM-001 |
| **Estimated** | 3h |

---

## Context

Tạo toàn bộ PostgreSQL migration files cho AgentMemory module. Phải chạy TRƯỚC khi implement các service.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `deploy/dev/migrations/0040_observe_service.up.sql` |
| CREATE | `deploy/dev/migrations/0041_agent_memory.up.sql` |
| CREATE | `deploy/dev/migrations/0042_orchestration.up.sql` |
| CREATE | `deploy/dev/migrations/0043_consolidation.up.sql` |
| CREATE | `deploy/dev/migrations/0044_governance.up.sql` |
| CREATE | `deploy/dev/migrations/0040_observe_service.down.sql` |
| CREATE | `deploy/dev/migrations/0041_agent_memory.down.sql` |
| CREATE | `deploy/dev/migrations/0042_orchestration.down.sql` |
| CREATE | `deploy/dev/migrations/0043_consolidation.down.sql` |
| CREATE | `deploy/dev/migrations/0044_governance.down.sql` |

---

## Migration Content

### `0040_observe_service.up.sql`

```sql
-- Observe Service: Sessions + Observations
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

-- Deduplication cache
CREATE TABLE IF NOT EXISTS dedup_cache (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    content_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_sessions_tenant ON agent_sessions(tenant_id, status, project);
CREATE INDEX idx_agent_sessions_started ON agent_sessions(started_at DESC);
CREATE INDEX idx_raw_obs_session ON raw_observations(session_id, timestamp);
CREATE INDEX idx_compressed_obs_session ON compressed_observations(session_id, timestamp);
CREATE INDEX idx_dedup_hash ON dedup_cache(content_hash, session_id);
```

### `0041_agent_memory.up.sql`

```sql
-- AgentMemory: Long-term memories + Memory Slots
CREATE TABLE IF NOT EXISTS agent_memories (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             TEXT NOT NULL,
    project               TEXT NOT NULL DEFAULT '',
    type                  TEXT NOT NULL CHECK (type IN ('pattern','preference','architecture','bug','workflow','fact')),
    title                 TEXT NOT NULL,
    content               TEXT NOT NULL,
    concepts              TEXT[] NOT NULL DEFAULT '{}',
    files                 TEXT[] NOT NULL DEFAULT '{}',
    session_ids           TEXT[] NOT NULL DEFAULT '{}',
    strength              FLOAT NOT NULL DEFAULT 0.7 CHECK (strength >= 0 AND strength <= 1),
    version               INTEGER NOT NULL DEFAULT 1,
    parent_id             UUID REFERENCES agent_memories(id),
    supersedes            TEXT[] NOT NULL DEFAULT '{}',
    related_ids           TEXT[] NOT NULL DEFAULT '{}',
    source_observation_ids TEXT[] NOT NULL DEFAULT '{}',
    is_latest             BOOLEAN NOT NULL DEFAULT TRUE,
    forget_after          TIMESTAMPTZ,
    agent_id              TEXT,
    flagged_eviction      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
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

CREATE INDEX idx_agent_memories_tenant_project ON agent_memories(tenant_id, project, type);
CREATE INDEX idx_agent_memories_latest ON agent_memories(tenant_id, project, is_latest) WHERE is_latest = TRUE;
CREATE INDEX idx_agent_memories_forget ON agent_memories(forget_after) WHERE forget_after IS NOT NULL;
CREATE INDEX idx_agent_memories_concepts ON agent_memories USING GIN (concepts);
CREATE INDEX idx_memory_slots_tenant ON memory_slots(tenant_id, scope);
```

### `0042_orchestration.up.sql`

```sql
-- Orchestration: Actions, Leases, Signals, Checkpoints, Sentinels, Routines, Sketches, Crystals
CREATE TABLE IF NOT EXISTS actions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       TEXT NOT NULL,
    project         TEXT NOT NULL DEFAULT '',
    agent_id        TEXT,
    title           TEXT NOT NULL,
    description     TEXT,
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','active','blocked','done','cancelled','failed')),
    priority        INTEGER NOT NULL DEFAULT 50 CHECK (priority BETWEEN 0 AND 100),
    requires        TEXT[] NOT NULL DEFAULT '{}',
    conflicts_with  TEXT[] NOT NULL DEFAULT '{}',
    tags            TEXT[] NOT NULL DEFAULT '{}',
    result          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS leases (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action_id   UUID NOT NULL REFERENCES actions(id) ON DELETE CASCADE,
    agent_id    TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','expired','released')),
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    renewed_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS signals (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    from_agent  TEXT NOT NULL,
    to_agent    TEXT NOT NULL,
    signal_type TEXT NOT NULL CHECK (signal_type IN ('handoff','update','cancel','request','response','alert')),
    content     TEXT NOT NULL,
    thread_id   TEXT,
    reply_to    UUID REFERENCES signals(id),
    is_read     BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '7 days'),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS routines (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    project     TEXT NOT NULL DEFAULT '',
    name        TEXT NOT NULL,
    description TEXT,
    steps       JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS checkpoints (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    project     TEXT NOT NULL DEFAULT '',
    agent_id    TEXT,
    action_id   UUID REFERENCES actions(id),
    title       TEXT NOT NULL,
    description TEXT,
    status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected','expired')),
    approved_by TEXT,
    rejected_by TEXT,
    reason      TEXT,
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours'),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS sentinels (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    TEXT NOT NULL,
    name         TEXT NOT NULL,
    condition    JSONB NOT NULL,  -- {type, target, value}
    action_id    UUID REFERENCES actions(id),
    signal_to    TEXT,
    status       TEXT NOT NULL DEFAULT 'watching' CHECK (status IN ('watching','triggered','expired')),
    expires_at   TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '30 days'),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    triggered_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS sketches (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    project     TEXT NOT NULL DEFAULT '',
    session_id  TEXT,
    title       TEXT NOT NULL,
    action_ids  TEXT[] NOT NULL DEFAULT '{}',
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','promoted','expired')),
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '72 hours'),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS crystals (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        TEXT NOT NULL,
    source_action_ids TEXT[] NOT NULL DEFAULT '{}',
    narrative        TEXT,
    key_outcomes     TEXT[] NOT NULL DEFAULT '{}',
    files_affected   TEXT[] NOT NULL DEFAULT '{}',
    lessons          TEXT[] NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_actions_tenant_status ON actions(tenant_id, status, project);
CREATE INDEX idx_leases_action ON leases(action_id, status) WHERE status = 'active';
CREATE INDEX idx_leases_expires ON leases(expires_at) WHERE status = 'active';
CREATE INDEX idx_signals_to_agent ON signals(tenant_id, to_agent, is_read);
CREATE INDEX idx_signals_expires ON signals(expires_at);
CREATE INDEX idx_checkpoints_status ON checkpoints(tenant_id, status);
CREATE INDEX idx_sentinels_watching ON sentinels(status) WHERE status = 'watching';
```

### `0043_consolidation.up.sql`

```sql
-- Consolidation: Session Summaries, Procedural Memories, Lessons, Insights
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

CREATE INDEX idx_session_summaries_tenant ON session_summaries(tenant_id);
CREATE INDEX idx_procedural_tenant ON procedural_memories(tenant_id, project);
CREATE INDEX idx_lessons_confidence ON lessons(tenant_id, confidence DESC);
CREATE INDEX idx_insights_tenant ON insights(tenant_id);
```

### `0044_governance.up.sql`

```sql
-- Governance: Audit Entries
CREATE TABLE IF NOT EXISTS audit_entries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    TEXT NOT NULL,
    timestamp    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    operation    TEXT NOT NULL,
    target_ids   TEXT[] NOT NULL DEFAULT '{}',
    performed_by TEXT,
    project      TEXT NOT NULL DEFAULT '',
    details      JSONB,
    reason       TEXT
);

-- Snapshot records
CREATE TABLE IF NOT EXISTS snapshot_records (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    commit_hash TEXT,
    sessions    INTEGER NOT NULL DEFAULT 0,
    observations INTEGER NOT NULL DEFAULT 0,
    memories    INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_entries_tenant ON audit_entries(tenant_id, timestamp DESC);
CREATE INDEX idx_audit_entries_operation ON audit_entries(tenant_id, operation);
CREATE INDEX idx_audit_entries_project ON audit_entries(tenant_id, project);
```

### Down Migrations

```sql
-- 0040_observe_service.down.sql
DROP TABLE IF EXISTS dedup_cache CASCADE;
DROP TABLE IF EXISTS compressed_observations CASCADE;
DROP TABLE IF EXISTS raw_observations CASCADE;
DROP TABLE IF EXISTS agent_sessions CASCADE;

-- 0041_agent_memory.down.sql
DROP TABLE IF EXISTS memory_slots CASCADE;
DROP TABLE IF EXISTS agent_memories CASCADE;

-- 0042_orchestration.down.sql
DROP TABLE IF EXISTS crystals CASCADE;
DROP TABLE IF EXISTS sketches CASCADE;
DROP TABLE IF EXISTS sentinels CASCADE;
DROP TABLE IF EXISTS checkpoints CASCADE;
DROP TABLE IF EXISTS routines CASCADE;
DROP TABLE IF EXISTS signals CASCADE;
DROP TABLE IF EXISTS leases CASCADE;
DROP TABLE IF EXISTS actions CASCADE;

-- 0043_consolidation.down.sql
DROP TABLE IF EXISTS insights CASCADE;
DROP TABLE IF EXISTS lessons CASCADE;
DROP TABLE IF EXISTS procedural_memories CASCADE;
DROP TABLE IF EXISTS session_summaries CASCADE;

-- 0044_governance.down.sql
DROP TABLE IF EXISTS snapshot_records CASCADE;
DROP TABLE IF EXISTS audit_entries CASCADE;
```

---

## Verification

```bash
# Run migrations
cd deploy/dev
make migrate-up

# Verify tables
psql $DATABASE_URL -c "\dt agent_*"
psql $DATABASE_URL -c "\dt raw_*"
psql $DATABASE_URL -c "\dt compressed_*"
psql $DATABASE_URL -c "\dt actions"
psql $DATABASE_URL -c "\dt leases"
psql $DATABASE_URL -c "\dt signals"
psql $DATABASE_URL -c "\dt audit_entries"

# Test rollback
make migrate-down STEPS=5
make migrate-up
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| All 5 up migrations run without error | ✅ |
| All 5 down migrations run without error | ✅ |
| Re-running up after down → idempotent (`IF NOT EXISTS`) | ✅ |
| Indexes on tenant_id columns for query performance | ✅ |
| Foreign key CASCADE DELETE on session_id refs | ✅ |
| Constraints: status, type ENUMs, priority range | ✅ |
