-- Indexes backing graph-scope reads and deletes.
--
-- A graph scope is derived from node properties (see deriveGraphScope in internal/write): a client
-- stamps `_kg_graph_scope` and, when it partitions its graph, `kg_level` / `feature_ref`. Reading
-- or deleting "everything in this scope at this level" is therefore a JSONB expression predicate,
-- which without an index degrades into a sequential scan of the whole table.
--
-- Composite ordering matches the predicate: scope always equality-matched first, then level, then
-- feature_ref. Partial on NOT is_deleted because every scope query excludes tombstones.

CREATE INDEX IF NOT EXISTS idx_kg_nodes_graph_scope
    ON kg_nodes (
        (properties ->> '_kg_graph_scope'),
        (properties ->> 'kg_level'),
        (properties ->> 'feature_ref')
    )
    WHERE NOT is_deleted;

CREATE INDEX IF NOT EXISTS idx_kg_relationships_graph_scope
    ON kg_relationships (
        (properties ->> '_kg_graph_scope'),
        (properties ->> 'kg_level'),
        (properties ->> 'feature_ref')
    )
    WHERE NOT is_deleted;

-- text_pattern_ops makes `external_ref LIKE 'prefix%'` index-usable regardless of the database
-- collation. The existing DELETE /v1/kg/write/nodes:by-external-ref-prefix endpoint runs exactly
-- that predicate and has been scanning until now.
CREATE INDEX IF NOT EXISTS idx_kg_nodes_external_ref_prefix
    ON kg_nodes (external_ref text_pattern_ops)
    WHERE external_ref IS NOT NULL AND NOT is_deleted;
