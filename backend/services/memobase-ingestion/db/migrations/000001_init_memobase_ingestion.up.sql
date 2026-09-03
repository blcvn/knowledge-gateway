CREATE TABLE IF NOT EXISTS general_blobs (
    id UUID PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,
    blob_type VARCHAR(20) NOT NULL,
    blob_data JSONB NOT NULL,
    add_fields JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_blobs_user_type ON general_blobs(user_id, project_id, blob_type);

CREATE TABLE IF NOT EXISTS buffer_zones (
    id UUID PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,
    blob_id UUID,
    blob_type VARCHAR(20) NOT NULL,
    token_size INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'idle',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT fk_blob FOREIGN KEY(blob_id) REFERENCES general_blobs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_buffer_user_status ON buffer_zones(user_id, project_id, blob_type, status);
CREATE INDEX IF NOT EXISTS idx_buffer_status_idle ON buffer_zones(status) WHERE status = 'idle';
