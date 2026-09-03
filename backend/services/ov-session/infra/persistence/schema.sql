-- +goose Up
CREATE TABLE ov_sessions (
    id UUID PRIMARY KEY,
    account_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    agent_id VARCHAR(64) NOT NULL DEFAULT 'default',
    title TEXT,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    archive_path TEXT,
    memories_count INT DEFAULT 0,
    compression_version VARCHAR(4) DEFAULT 'v2',
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL,
    committed_at TIMESTAMPTZ
);
CREATE INDEX idx_sessions_account_user ON ov_sessions(account_id, user_id);
CREATE INDEX idx_sessions_status ON ov_sessions(account_id, status);

CREATE TABLE ov_messages (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES ov_sessions(id),
    role VARCHAR(16) NOT NULL,
    content TEXT NOT NULL,
    tool_calls JSONB,
    token_count INT,
    sequence INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_messages_session_seq ON ov_messages(session_id, sequence);

CREATE TABLE ov_working_memory (
    session_id UUID PRIMARY KEY REFERENCES ov_sessions(id),
    title TEXT,
    state VARCHAR(16) NOT NULL DEFAULT 'ongoing',
    goals JSONB DEFAULT '[]',
    facts JSONB DEFAULT '[]',
    errors JSONB DEFAULT '[]',
    context JSONB DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE ov_extracted_memories (
    id UUID PRIMARY KEY,
    session_id UUID REFERENCES ov_sessions(id),
    account_id VARCHAR(64) NOT NULL,
    category VARCHAR(16) NOT NULL,
    content TEXT NOT NULL,
    confidence FLOAT8 DEFAULT 1.0,
    dedup_action VARCHAR(8) NOT NULL,
    fs_path TEXT,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_memories_account_cat ON ov_extracted_memories(account_id, category);

-- +goose Down
DROP TABLE ov_extracted_memories;
DROP TABLE ov_working_memory;
DROP TABLE ov_messages;
DROP TABLE ov_sessions;
