-- Allow the version status the code has always written when a sync session is given up on.
--
-- kg_graph_versions.version_status was created (000012) permitting only PENDING_ENTITIES, SEALED and
-- FAILED_FINALIZATION, but both paths that give up on a session write 'ABANDONED':
-- Repository.AbandonGraphVersion (an explicit abandon by the client) and
-- cleanupExpiredSyncSessionInTx (the background sweep for sessions past their TTL). Every such
-- UPDATE was rejected by this check on Postgres. The in-memory store has no constraint and accepts
-- the value, so the unit suite passed throughout and the failure only ever appeared against a real
-- database.
--
-- The consequence was worse than a failed abandon. The sweep updates the version and deletes the
-- scope lease in one transaction, so the rejected UPDATE rolled the DELETE back with it: a lease
-- left behind by a writer that died mid-write was never released, by anyone, and the graph scope
-- stayed locked permanently. Every later write to that scope failed with SYNC_SCOPE_LOCKED and no
-- client-side recovery was possible, because releasing another session's lease is the service's job.
--
-- Widening the constraint is the whole fix: the code, the API and the read paths already treat
-- ABANDONED as a real state (see internal/write/service.go, which checks for it before reusing a
-- session), and it is only this table's definition that never learned about it.

ALTER TABLE kg_graph_versions
    DROP CONSTRAINT IF EXISTS kg_graph_versions_version_status_check;

ALTER TABLE kg_graph_versions
    ADD CONSTRAINT kg_graph_versions_version_status_check
    CHECK (version_status IN ('PENDING_ENTITIES', 'SEALED', 'FAILED_FINALIZATION', 'ABANDONED'));

-- Release the leases stranded by the bug.
--
-- Any version still PENDING_ENTITIES from before this migration could never have been abandoned, so
-- its lease is unreleasable by definition. Only sessions already past the longest supported stale
-- TTL are touched, so a session opened by a live writer during the deploy is left alone.
UPDATE kg_graph_versions
SET version_status = 'ABANDONED'
WHERE version_status = 'PENDING_ENTITIES'
  AND created_at < now() - INTERVAL '24 hours';

-- A lease names the version holding it, so the ones to drop are exactly those whose version is no
-- longer accepting entities: the sessions abandoned just above, plus any that had already finished
-- without their lease being cleaned up.
DELETE FROM kg_graph_scope_leases lease
WHERE NOT EXISTS (
    SELECT 1
    FROM kg_graph_versions version
    WHERE version.version_id = lease.version_id
      AND version.version_status = 'PENDING_ENTITIES'
);
