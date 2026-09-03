# TASK-BE-004 — PostgreSQL: Sessions + Messages tables

| Field | Value |
|---|---|
| **Task ID** | TASK-BE-004 |
| **Layer** | Backend — PostgreSQL |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-004 §2.1](../solutions/SOL-004-Sessions-Solution.md) |
| **Priority** | 🔴 P0 |
| **Depends On** | TASK-BE-001 |
| **Estimated** | 1h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `vnp-platform/migrations/0003_create_sessions.sql` |
| CREATE | `vnp-platform/migrations/0004_create_messages.sql` |

---

## Implementation

### File: `0003_create_sessions.sql`

```sql
-- +migrate Up
CREATE TABLE IF NOT EXISTS sessions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL,
    user_id      TEXT        NOT NULL,        -- External user ID (from SDK)
    agent_id     TEXT,
    engine       TEXT        NOT NULL DEFAULT 'zep',
    status       TEXT        NOT NULL DEFAULT 'active',
                                              -- active | completed | failed
    started_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at     TIMESTAMPTZ,
    metadata     JSONB       DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_tenant_status  ON sessions(tenant_id, status);
CREATE INDEX idx_sessions_user_id        ON sessions(tenant_id, user_id);
CREATE INDEX idx_sessions_created_at     ON sessions(tenant_id, created_at DESC);

-- Full-text search
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS tsv TSVECTOR
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(user_id, '') || ' ' || coalesce(agent_id, ''))) STORED;
CREATE INDEX idx_sessions_fts ON sessions USING GIN(tsv);

-- +migrate Down
DROP TABLE IF EXISTS sessions;
```

### File: `0004_create_messages.sql`

```sql
-- +migrate Up
CREATE TABLE IF NOT EXISTS messages (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id     UUID        NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    tenant_id      UUID        NOT NULL,
    role           TEXT        NOT NULL,   -- user | assistant | system | tool
    content        TEXT        NOT NULL,
    memory_sources TEXT[]      DEFAULT '{}',  -- ["graphiti:ep_abc", "memobase:prof_xyz"]
    tokens         INT         DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_session_id ON messages(session_id, created_at ASC);

-- +migrate Down
DROP TABLE IF EXISTS messages;
```

---

## Verification

```bash
psql $DATABASE_URL -c "\d sessions"
psql $DATABASE_URL -c "\d messages"

# Test FTS
psql $DATABASE_URL -c "INSERT INTO sessions (tenant_id, user_id) VALUES ('test-uuid', 'user123');"
psql $DATABASE_URL -c "SELECT id FROM sessions WHERE tsv @@ to_tsquery('user123');"
```
