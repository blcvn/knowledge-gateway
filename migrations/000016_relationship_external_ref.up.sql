-- Relationship external references.
--
-- kg_nodes has carried an `external_ref` since 000004: it is the caller-owned identity that makes
-- a write an upsert instead of an insert. kg_relationships never had one, so a client that
-- re-sends the same logical edge (a graph re-persist, a retry) creates a new row every time.
--
-- Nullable and additive: existing callers that omit external_ref keep inserting exactly as before.
--
-- Index shape mirrors kg_nodes deliberately:
--   * partial UNIQUE over live rows  -> enforces one active relationship per reference, while
--     leaving soft-deleted rows in place so the same reference can be revived later.
--   * plain index over all rows      -> lookup by reference must still find a soft-deleted row,
--     because revival reads it before flipping is_deleted back to false.

ALTER TABLE kg_relationships
    ADD COLUMN IF NOT EXISTS external_ref TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_kg_relationships_external_ref_active
    ON kg_relationships(external_ref)
    WHERE external_ref IS NOT NULL AND NOT is_deleted;

CREATE INDEX IF NOT EXISTS idx_kg_relationships_external_ref
    ON kg_relationships(external_ref)
    WHERE external_ref IS NOT NULL;
