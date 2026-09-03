# TASK-CE-011 — Database Migrations (4 New Tables)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-CE-011 |
| **Wave** | 4 |
| **Component** | `db/migrations/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-001 §2.6, SOL-003 §2.6, SOL-005 §2.6 |
| **Priority** | 🔴 Critical |
| **Depends On** | — (can run in parallel with Wave 2) |
| **Estimated** | 1h |

---

## Context

Tạo 3 migration files cho 4 new PostgreSQL tables được thêm bởi Cognee feature parity:
- `cognee_pipeline_runs` — CR-001 (Memify job tracking)
- `cognee_datapoints` — CR-003 (DataPoint metadata)
- `cognee_interactions` — CR-005 (Search interaction log)
- `cognee_feedback_records` — CR-005 (Feedback records)

Sử dụng naming convention của project: `NNNN_description.up.sql` / `.down.sql`.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `db/migrations/0020_cognee_pipeline_runs.up.sql` |
| CREATE | `db/migrations/0020_cognee_pipeline_runs.down.sql` |
| CREATE | `db/migrations/0021_cognee_datapoints.up.sql` |
| CREATE | `db/migrations/0021_cognee_datapoints.down.sql` |
| CREATE | `db/migrations/0022_cognee_interactions.up.sql` |
| CREATE | `db/migrations/0022_cognee_interactions.down.sql` |

---

## Implementation

### Migration 0020 — `cognee_pipeline_runs`

**UP: `0020_cognee_pipeline_runs.up.sql`**
```sql
-- Migration: 0020_cognee_pipeline_runs.up.sql
-- Purpose: Track async cognify and memify pipeline job status
-- CR: CR-COGNEE-001 (Memify), CR-COGNEE-006 (Custom Pipelines)

CREATE TABLE IF NOT EXISTS cognee_pipeline_runs (
    id          TEXT        PRIMARY KEY,
    dataset_id  TEXT        NOT NULL,
    tenant_id   TEXT        NOT NULL,
    type        TEXT        NOT NULL,    -- 'cognify' | 'memify'
    status      TEXT        NOT NULL DEFAULT 'QUEUED',  -- QUEUED | RUNNING | COMPLETED | FAILED
    new_nodes   INT         NOT NULL DEFAULT 0,
    new_edges   INT         NOT NULL DEFAULT 0,
    error       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cognee_pipeline_runs_dataset
    ON cognee_pipeline_runs(dataset_id, tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_cognee_pipeline_runs_tenant
    ON cognee_pipeline_runs(tenant_id, created_at DESC);

COMMENT ON TABLE cognee_pipeline_runs IS
    'Tracks background cognify and memify pipeline job status for CR-COGNEE-001/006';

COMMENT ON COLUMN cognee_pipeline_runs.type IS '''cognify'' for StartCognify jobs, ''memify'' for Memify jobs';
COMMENT ON COLUMN cognee_pipeline_runs.status IS 'QUEUED | RUNNING | COMPLETED | FAILED';
```

**DOWN: `0020_cognee_pipeline_runs.down.sql`**
```sql
DROP TABLE IF EXISTS cognee_pipeline_runs;
```

---

### Migration 0021 — `cognee_datapoints`

**UP: `0021_cognee_datapoints.up.sql`**
```sql
-- Migration: 0021_cognee_datapoints.up.sql
-- Purpose: Metadata for structured DataPoint ingestion (zero LLM)
-- CR: CR-COGNEE-003 (DataPoint Schema)

CREATE TABLE IF NOT EXISTS cognee_datapoints (
    id           UUID        PRIMARY KEY,
    version      INT         NOT NULL DEFAULT 1,
    dataset_id   UUID        NOT NULL,
    tenant_id    TEXT        NOT NULL,
    type         TEXT        NOT NULL,          -- "Paper" | "Employee" | "Product" etc.
    fields       JSONB       NOT NULL DEFAULT '{}',
    index_fields TEXT[]      NOT NULL DEFAULT '{}',
    node_sets    TEXT[]      NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cognee_datapoints_dataset
    ON cognee_datapoints(dataset_id, tenant_id);

CREATE INDEX IF NOT EXISTS idx_cognee_datapoints_type
    ON cognee_datapoints(tenant_id, type);

CREATE INDEX IF NOT EXISTS idx_cognee_datapoints_node_sets
    ON cognee_datapoints USING GIN (node_sets);

CREATE INDEX IF NOT EXISTS idx_cognee_datapoints_fields
    ON cognee_datapoints USING GIN (fields jsonb_path_ops);

COMMENT ON TABLE cognee_datapoints IS
    'Metadata for structured DataPoints ingested without LLM (CR-COGNEE-003)';

COMMENT ON COLUMN cognee_datapoints.type IS 'DataPoint schema type: Paper, Employee, Product, etc.';
COMMENT ON COLUMN cognee_datapoints.fields IS 'All field values as JSONB (dynamic schema)';
COMMENT ON COLUMN cognee_datapoints.index_fields IS 'Only these fields are embedded into Qdrant vectors';
COMMENT ON COLUMN cognee_datapoints.node_sets IS 'NodeSet tags for memory scoping (CR-COGNEE-002)';
COMMENT ON COLUMN cognee_datapoints.version IS 'Incremented on each upsert (idempotent ingestion)';
```

**DOWN: `0021_cognee_datapoints.down.sql`**
```sql
DROP TABLE IF EXISTS cognee_datapoints;
```

---

### Migration 0022 — `cognee_interactions` + `cognee_feedback_records`

**UP: `0022_cognee_interactions.up.sql`**
```sql
-- Migration: 0022_cognee_interactions.up.sql
-- Purpose: Interaction logging and feedback records for self-improving retrieval
-- CR: CR-COGNEE-005 (Feedback Loop)

-- Table 1: Search interaction log
CREATE TABLE IF NOT EXISTS cognee_interactions (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     TEXT        NOT NULL,
    session_id    TEXT,                          -- optional grouping
    dataset_id    UUID,                          -- optional dataset filter used in search
    query         TEXT        NOT NULL,
    strategy      TEXT        NOT NULL,          -- SIMILARITY | GRAPH_COMPLETION | KEYWORD | ...
    result_ids    TEXT[]      NOT NULL DEFAULT '{}',
    result_scores FLOAT8[]    NOT NULL DEFAULT '{}',
    node_sets     TEXT[]      NOT NULL DEFAULT '{}',
    timestamp     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cognee_interactions_tenant
    ON cognee_interactions(tenant_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_cognee_interactions_session
    ON cognee_interactions(session_id, timestamp DESC)
    WHERE session_id IS NOT NULL;

-- Table 2: User feedback on search interactions
CREATE TABLE IF NOT EXISTS cognee_feedback_records (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    interaction_id  UUID        NOT NULL REFERENCES cognee_interactions(id) ON DELETE CASCADE,
    tenant_id       TEXT        NOT NULL,
    score           FLOAT8      NOT NULL,       -- -1.0 (negative) to 1.0 (positive)
    text            TEXT,                       -- optional text comment
    affected_nodes  TEXT[]      NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_feedback_score CHECK (score >= -1.0 AND score <= 1.0)
);

CREATE INDEX IF NOT EXISTS idx_cognee_feedback_interaction
    ON cognee_feedback_records(interaction_id);

CREATE INDEX IF NOT EXISTS idx_cognee_feedback_tenant
    ON cognee_feedback_records(tenant_id, created_at DESC);

COMMENT ON TABLE cognee_interactions IS
    'Logs search calls when save_interaction=true, for feedback tracking (CR-COGNEE-005)';

COMMENT ON TABLE cognee_feedback_records IS
    'User feedback on search interactions, triggers Neo4j edge weight adjustment (CR-COGNEE-005)';

COMMENT ON COLUMN cognee_feedback_records.score IS '-1.0 = negative, 0 = neutral, 1.0 = positive';
COMMENT ON COLUMN cognee_feedback_records.affected_nodes IS 'Node IDs whose edge weights were adjusted';
```

**DOWN: `0022_cognee_interactions.down.sql`**
```sql
DROP TABLE IF EXISTS cognee_feedback_records;
DROP TABLE IF EXISTS cognee_interactions;
```

---

## Migration Order

Migrations must be applied in sequence:

```bash
# Apply all cognee migrations
migrate -database postgres://... -path db/migrations up

# Or apply specific
migrate -database postgres://... -path db/migrations goto 22
```

**Dependencies:**
- `0020` — independent, can run first
- `0021` — independent, can run after 0020
- `0022` — depends on nothing; `cognee_feedback_records.interaction_id` references `cognee_interactions`

---

## Verification

```bash
# Apply migrations
migrate -database $DATABASE_URL -path db/migrations up

# Verify tables created
psql $DATABASE_URL -c "\dt cognee_*"
# Expected: 4 tables

# Verify indices
psql $DATABASE_URL -c "\di cognee_*"

# Verify constraints
psql $DATABASE_URL -c "
SELECT conname, contype, pg_get_constraintdef(oid)
FROM pg_constraint
WHERE conrelid IN (
    'cognee_pipeline_runs'::regclass,
    'cognee_datapoints'::regclass,
    'cognee_interactions'::regclass,
    'cognee_feedback_records'::regclass
)
ORDER BY conrelid::text, conname;
"

# Test constraint: feedback_score must be in [-1.0, 1.0]
psql $DATABASE_URL -c "
INSERT INTO cognee_interactions (tenant_id, query, strategy) VALUES ('t1', 'test', 'SIMILARITY');
INSERT INTO cognee_feedback_records (interaction_id, tenant_id, score)
    SELECT id, 't1', 2.0 FROM cognee_interactions WHERE query='test';
"
-- Expected: ERROR: violates check constraint chk_feedback_score

# Rollback test
migrate -database $DATABASE_URL -path db/migrations down 3
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| `cognee_pipeline_runs` table exists with correct schema | ✅ |
| `cognee_datapoints` table with JSONB fields + GIN indices | ✅ |
| `cognee_interactions` table with session_id nullable | ✅ |
| `cognee_feedback_records` with FK + CHECK constraint | ✅ |
| All `.down.sql` rollback migrations work | ✅ |
| `go build ./...` with new repo implementations | ✅ |
