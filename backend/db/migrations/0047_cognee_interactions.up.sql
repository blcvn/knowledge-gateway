-- Migration: 0047_cognee_interactions.up.sql
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
