CREATE TABLE IF NOT EXISTS graphiti_episodes (
    uuid UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    group_id VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    source VARCHAR(50) NOT NULL,
    reference_time TIMESTAMP WITH TIME ZONE NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    saga_id UUID,
    entity_types JSONB,
    edge_types JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS graphiti_episode_dedup (
    content_hash VARCHAR(64) PRIMARY KEY,
    episode_id UUID NOT NULL REFERENCES graphiti_episodes(uuid),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS graphiti_saga_state (
    id UUID PRIMARY KEY,
    episode_id UUID NOT NULL,
    group_id VARCHAR(255) NOT NULL,
    current_step VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    step_history JSONB,
    retry_count INT DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_episodes_group_id ON graphiti_episodes(group_id);
CREATE INDEX idx_saga_group_id ON graphiti_saga_state(group_id);
