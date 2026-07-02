#!/usr/bin/env bash

set -euo pipefail

psql "${DATABASE_URL:?DATABASE_URL is required}" -v ON_ERROR_STOP=1 <<'SQL'
WITH remaining_fks AS (
    SELECT conname
    FROM pg_constraint
    WHERE conrelid IN (
        'kg_relationships'::regclass,
        'kg_vector_documents'::regclass
    )
    AND contype = 'f'
)
SELECT
    COALESCE(bool_and(conname NOT IN (
        'kg_relationships_from_node_id_fkey',
        'kg_relationships_to_node_id_fkey',
        'kg_vector_documents_node_id_fkey'
    )), true) AS expected_fk_state
FROM remaining_fks;

SELECT
    (SELECT count(*) FROM kg_relationships r WHERE NOT EXISTS (SELECT 1 FROM kg_nodes n WHERE n.id = r.from_node_id)) AS orphan_from_count,
    (SELECT count(*) FROM kg_relationships r WHERE NOT EXISTS (SELECT 1 FROM kg_nodes n WHERE n.id = r.to_node_id)) AS orphan_to_count,
    (SELECT count(*) FROM kg_vector_documents v WHERE NOT EXISTS (SELECT 1 FROM kg_nodes n WHERE n.id = v.node_id)) AS orphan_vector_count;
SQL
