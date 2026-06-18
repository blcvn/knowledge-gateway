DROP TABLE IF EXISTS kg_projection_versions;

ALTER TABLE kg_vector_documents
    DROP COLUMN IF EXISTS sync_version;

ALTER TABLE kg_relationships
    DROP COLUMN IF EXISTS domain_version;
