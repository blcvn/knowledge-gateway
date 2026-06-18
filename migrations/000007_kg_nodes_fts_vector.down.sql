DROP INDEX IF EXISTS idx_kg_nodes_fts_vector;
ALTER TABLE kg_nodes DROP COLUMN IF EXISTS fts_vector;
