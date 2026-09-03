-- Episode table
CREATE TABLE IF NOT EXISTS episodes (
    id UUID PRIMARY KEY,
    group_id VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    content_hash VARCHAR(255) NOT NULL,
    reference_time TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(group_id, content_hash)
);

-- Saga state table
CREATE TABLE IF NOT EXISTS sagas (
    id UUID PRIMARY KEY,
    episode_id UUID NOT NULL REFERENCES episodes(id),
    group_id VARCHAR(255) NOT NULL,
    state VARCHAR(50) NOT NULL,
    current_step VARCHAR(50) NOT NULL,
    error_details TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
