ALTER TABLE kg_relationships
    DROP CONSTRAINT IF EXISTS kg_relationships_from_node_id_fkey,
    DROP CONSTRAINT IF EXISTS kg_relationships_to_node_id_fkey;

ALTER TABLE kg_vector_documents
    DROP CONSTRAINT IF EXISTS kg_vector_documents_node_id_fkey;

CREATE INDEX IF NOT EXISTS idx_kg_relationships_from_node_id
    ON kg_relationships(from_node_id);

CREATE INDEX IF NOT EXISTS idx_kg_relationships_to_node_id
    ON kg_relationships(to_node_id);

CREATE INDEX IF NOT EXISTS idx_kg_vector_documents_node_id_lookup
    ON kg_vector_documents(node_id);

