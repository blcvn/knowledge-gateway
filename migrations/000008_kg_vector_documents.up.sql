CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE kg_vector_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID NOT NULL REFERENCES kg_nodes(id) ON DELETE CASCADE,
    node_type TEXT NOT NULL,
    domain_id TEXT NOT NULL REFERENCES domains(id),
    owner_tenant_id UUID NOT NULL REFERENCES tenants(id),
    owner_app_id UUID REFERENCES apps(id),
    acl_visible_to TEXT[] NOT NULL DEFAULT '{}'::text[],
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    status_value TEXT,
    authority_score INT,
    domain_props JSONB NOT NULL DEFAULT '{}'::jsonb,
    embedding vector NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_kg_vector_documents_node_id
    ON kg_vector_documents(node_id);

CREATE INDEX idx_kg_vector_documents_domain
    ON kg_vector_documents(domain_id)
    WHERE NOT is_deleted;

CREATE INDEX idx_kg_vector_documents_owner
    ON kg_vector_documents(owner_tenant_id, owner_app_id);

-- NOTE: An HNSW index requires a fixed dimension, e.g. vector(1536).
-- The embedding dimension depends on the configured provider (deterministic=8,
-- OpenAI=1536, etc.). Add the HNSW index manually after setting VECTOR_ADAPTER
-- and confirming the dimension, for example:
--   ALTER TABLE kg_vector_documents ALTER COLUMN embedding TYPE vector(1536);
--   CREATE INDEX idx_kg_vector_documents_embedding_hnsw
--       ON kg_vector_documents USING hnsw (embedding vector_cosine_ops);
