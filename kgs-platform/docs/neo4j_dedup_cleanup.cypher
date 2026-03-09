// Neo4j cleanup script for legacy duplicate data.
// Run carefully on dev/staging first.
// Requires APOC for relationship rewire step.

// -----------------------------------------------------------------------------
// T1-9: Detect duplicate nodes by business key (app_id, tenant_id, id)
// -----------------------------------------------------------------------------
MATCH (n)
WITH n.app_id AS app_id, n.tenant_id AS tenant_id, n.id AS id, collect(n) AS nodes
WHERE app_id IS NOT NULL AND tenant_id IS NOT NULL AND id IS NOT NULL AND size(nodes) > 1
RETURN app_id, tenant_id, id, size(nodes) AS duplicate_count
ORDER BY duplicate_count DESC, app_id, tenant_id, id;

// -----------------------------------------------------------------------------
// T1-10: Merge duplicate nodes (keep first node, move relationships, delete rest)
// -----------------------------------------------------------------------------
MATCH (n)
WITH n.app_id AS app_id, n.tenant_id AS tenant_id, n.id AS id, collect(n) AS nodes
WHERE app_id IS NOT NULL AND tenant_id IS NOT NULL AND id IS NOT NULL AND size(nodes) > 1
WITH nodes[0] AS keep, nodes[1..] AS duplicates
UNWIND duplicates AS dup
CALL {
  WITH keep, dup
  MATCH (dup)-[r]->(m)
  WHERE m <> keep
  WITH keep, m, r, type(r) AS rel_type, properties(r) AS rel_props
  CALL apoc.create.relationship(keep, rel_type, rel_props, m) YIELD rel
  DELETE r
  RETURN count(*) AS moved_outgoing
}
CALL {
  WITH keep, dup
  MATCH (m)-[r]->(dup)
  WHERE m <> keep
  WITH keep, m, r, type(r) AS rel_type, properties(r) AS rel_props
  CALL apoc.create.relationship(m, rel_type, rel_props, keep) YIELD rel
  DELETE r
  RETURN count(*) AS moved_incoming
}
DETACH DELETE dup;

// -----------------------------------------------------------------------------
// T1-11: Remove duplicate relationships by (source, target, type, id)
// -----------------------------------------------------------------------------
MATCH (a)-[r]->(b)
WITH a, b, type(r) AS rel_type, r.id AS rel_id, collect(r) AS rels
WHERE rel_id IS NOT NULL AND size(rels) > 1
WITH rels[0] AS keep, rels[1..] AS duplicates
UNWIND duplicates AS dup
DELETE dup;

