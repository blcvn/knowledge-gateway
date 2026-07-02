DROP INDEX IF EXISTS idx_kg_vector_documents_node_id_lookup;
DROP INDEX IF EXISTS idx_kg_relationships_to_node_id;
DROP INDEX IF EXISTS idx_kg_relationships_from_node_id;

ALTER TABLE kg_vector_documents
    ADD CONSTRAINT kg_vector_documents_node_id_fkey FOREIGN KEY (node_id) REFERENCES kg_nodes(id) ON DELETE CASCADE;

ALTER TABLE kg_relationships
    ADD CONSTRAINT kg_relationships_from_node_id_fkey FOREIGN KEY (from_node_id) REFERENCES kg_nodes(id),
    ADD CONSTRAINT kg_relationships_to_node_id_fkey FOREIGN KEY (to_node_id) REFERENCES kg_nodes(id);

