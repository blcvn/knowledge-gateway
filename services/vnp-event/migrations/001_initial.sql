-- vnp-event initial schema
-- Reference: services/vnp-event/specs/tdd.md §2

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";  -- pgvector

-- User Events (bi-temporal)
CREATE TABLE IF NOT EXISTS user_events (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID NOT NULL,
    tenant_id  UUID NOT NULL,
    source     VARCHAR(20) NOT NULL,  -- COGNEE|GRAPHITI|MEMOBASE|OPENVIKING|ZEP|SUPERMEMORY
    content    TEXT NOT NULL,
    tags       TEXT[] NOT NULL DEFAULT '{}',
    embedding  vector(1536),          -- pgvector: OpenAI ada-002 dimension
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- bi-temporal: when event was true
    invalid_at TIMESTAMPTZ                           -- bi-temporal: when event ceased being true
);

CREATE INDEX idx_events_tenant ON user_events(tenant_id);
CREATE INDEX idx_events_user ON user_events(tenant_id, user_id);
CREATE INDEX idx_events_source ON user_events(source);
CREATE INDEX idx_events_valid_at ON user_events(valid_at);
CREATE INDEX idx_events_tags ON user_events USING GIN(tags);
CREATE INDEX idx_events_embedding ON user_events USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

-- Event Gists (summarized batches)
CREATE TABLE IF NOT EXISTS event_gists (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_ids  UUID[] NOT NULL,
    summary    TEXT NOT NULL,
    embedding  vector(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_gists_embedding ON event_gists USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 50);
