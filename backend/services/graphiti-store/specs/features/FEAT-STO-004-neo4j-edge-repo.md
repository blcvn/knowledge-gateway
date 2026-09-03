---
id: FEAT-STO-004
title: Neo4j Edge Repository — Bi-Temporal Model
service: graphiti-store
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement Neo4j edge repository với bi-temporal model (valid_at, invalid_at, expired_at). Bao gồm edge CRUD, temporal invalidation, và time-range queries.

## Scope

### In Scope
- `internal/adapter/driver/neo4j/edge_repo.go` — SaveEdge, GetEdge, DeleteEdge, InvalidateEdge, GetEdgesInTimeRange
- Bi-temporal edge properties: valid_at, invalid_at, expired_at, created_at
- RELATES_TO edge type between Entity nodes
- MENTIONS edge type between Episodic → Entity
- Temporal range queries with window intersection

### Out of Scope
- Edge resolution logic (graphiti-knowledge responsibility)

## Thiết Kế Kỹ Thuật

### Cypher Patterns

```cypher
// SaveEdge
MATCH (source:Entity {uuid: $source_id}), (target:Entity {uuid: $target_id})
CREATE (source)-[r:RELATES_TO {
    uuid: $uuid, name: $name, fact: $fact,
    fact_embedding: $embedding,
    valid_at: datetime($valid_at),
    invalid_at: $invalid_at,
    expired_at: $expired_at,
    group_id: $group_id,
    episode_id: $episode_id,
    created_at: datetime()
}]->(target)
RETURN r

// InvalidateEdge (set invalid_at without deleting)
MATCH ()-[r:RELATES_TO {uuid: $uuid}]->()
SET r.invalid_at = datetime($invalid_at)
RETURN r

// GetEdgesInTimeRange (window intersection)
MATCH (s)-[r:RELATES_TO {group_id: $group_id}]->(t)
WHERE r.valid_at <= datetime($to)
  AND (r.invalid_at IS NULL OR r.invalid_at >= datetime($from))
RETURN s, r, t
```

## Acceptance Criteria

- [ ] AC-1: SaveEdge creates RELATES_TO relationship between two Entity nodes
- [ ] AC-2: InvalidateEdge sets invalid_at without deleting the edge
- [ ] AC-3: GetEdgesInTimeRange returns edges whose validity window intersects [from, to]
- [ ] AC-4: Edges with invalid_at = NULL are treated as "currently valid"
- [ ] AC-5: expired_at marks superseded edges (newer version exists)
- [ ] AC-6: All edge queries scoped by group_id

## Test Requirements
- **Integration tests**: Neo4j testcontainer, temporal edge scenarios
- **Minimum coverage**: 80%
