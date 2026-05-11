-- 1. Extensions
CREATE EXTENSION IF NOT EXISTS vector;

-- 2. Tables
CREATE TABLE IF NOT EXISTS user_profiles (
    id UUID,
    user_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    attributes JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (id, project_id)
);

CREATE TABLE IF NOT EXISTS user_events (
    id UUID,
    user_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,
    event_data JSONB NOT NULL,
    embedding vector(1536), -- Default embedding dim
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS user_event_gists (
    id UUID,
    user_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,
    event_id UUID NOT NULL,
    gist_data JSONB NOT NULL,
    embedding vector(1536), -- Default embedding dim
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id)
);

-- 3. Indexes
CREATE INDEX IF NOT EXISTS idx_profiles_user ON user_profiles(user_id, project_id);
CREATE INDEX IF NOT EXISTS idx_events_user ON user_events(user_id, project_id);
CREATE INDEX IF NOT EXISTS idx_events_embedding ON user_events USING hnsw (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_gists_event ON user_event_gists(user_id, project_id, event_id);
CREATE INDEX IF NOT EXISTS idx_gists_embedding ON user_event_gists USING hnsw (embedding vector_cosine_ops);
