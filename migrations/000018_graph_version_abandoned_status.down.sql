-- Restore the original status set.
--
-- Rows already carrying ABANDONED would violate the narrower constraint, so they are moved to
-- FAILED_FINALIZATION first: of the three permitted values it is the only one that also means "this
-- version never completed", and calling an abandoned version SEALED would present a partial write as
-- a finished one.

UPDATE kg_graph_versions
SET version_status = 'FAILED_FINALIZATION'
WHERE version_status = 'ABANDONED';

ALTER TABLE kg_graph_versions
    DROP CONSTRAINT IF EXISTS kg_graph_versions_version_status_check;

ALTER TABLE kg_graph_versions
    ADD CONSTRAINT kg_graph_versions_version_status_check
    CHECK (version_status IN ('PENDING_ENTITIES', 'SEALED', 'FAILED_FINALIZATION'));
