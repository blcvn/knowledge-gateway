DROP INDEX IF EXISTS idx_kg_relationships_external_ref;
DROP INDEX IF EXISTS idx_kg_relationships_external_ref_active;

ALTER TABLE kg_relationships
    DROP COLUMN IF EXISTS external_ref;
