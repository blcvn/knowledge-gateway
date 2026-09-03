-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE blobs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    type VARCHAR(50) NOT NULL,
    tokens INT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_blobs_tenant_user ON blobs(tenant_id, user_id);

CREATE TABLE buffer_zones (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    state VARCHAR(50) NOT NULL,
    token_count INT NOT NULL,
    threshold INT NOT NULL,
    blob_ids UUID[] NOT NULL,
    last_flushed TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(tenant_id, user_id)
);

CREATE TABLE profiles (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    topics JSONB NOT NULL,
    traits JSONB,
    version INT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(tenant_id, user_id)
);

CREATE TABLE event_gists (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    summary TEXT NOT NULL,
    key_facts JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_gists_tenant_user ON event_gists(tenant_id, user_id, created_at DESC);

-- +goose Down
DROP TABLE event_gists;
DROP TABLE profiles;
DROP TABLE buffer_zones;
DROP TABLE blobs;
