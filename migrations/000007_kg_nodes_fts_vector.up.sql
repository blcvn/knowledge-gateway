ALTER TABLE kg_nodes
    ADD COLUMN fts_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', COALESCE(id::text, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(node_type, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(domain_id, '')), 'B') ||
        setweight(to_tsvector('simple', COALESCE(external_ref, '')), 'C') ||
        setweight(to_tsvector('simple', COALESCE(status_value, '')), 'D') ||
        COALESCE(jsonb_to_tsvector('simple', properties, '["string"]'), ''::tsvector)
    ) STORED;

CREATE INDEX idx_kg_nodes_fts_vector
    ON kg_nodes
    USING GIN (fts_vector);
