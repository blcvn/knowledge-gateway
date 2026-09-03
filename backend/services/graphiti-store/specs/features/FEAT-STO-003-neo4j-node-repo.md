---
id: FEAT-STO-003
title: Neo4j Node Repository Adapter
service: graphiti-store
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement Neo4j node repository adapter cho tất cả node types: EntityNode, EpisodicNode, CommunityNode, SagaNode. Dùng Neo4j Go driver v5 với Cypher queries.

## Scope

### In Scope
- `internal/adapter/driver/neo4j/node_repo.go` — SaveNode, GetNode, GetNodeByName, DeleteNode, ListNodes
- Support cho 4 node labels: Entity, Episodic, Community, Saga
- Parameterized Cypher queries (prevent injection)
- Connection pool management
- Multi-tenant scoping (group_id filter on all queries)

### Out of Scope
- Edge operations (FEAT-STO-004)
- Search operations (FEAT-STO-005)

## Thiết Kế Kỹ Thuật

### Cypher Patterns

```cypher
// SaveNode (MERGE for upsert)
MERGE (n:Entity {uuid: $uuid})
SET n.name = $name, n.group_id = $group_id, n.summary = $summary,
    n.name_embedding = $embedding, n.labels = $labels,
    n.updated_at = datetime()
RETURN n

// GetNode
MATCH (n {uuid: $uuid})
WHERE n.group_id = $group_id
RETURN n

// ListNodes with cursor pagination
MATCH (n:Entity {group_id: $group_id})
WHERE n.uuid > $cursor
RETURN n ORDER BY n.uuid LIMIT $limit
```

## Acceptance Criteria

- [ ] AC-1: SaveNode creates/updates EntityNode with all properties in Neo4j
- [ ] AC-2: GetNode retrieves node by UUID scoped to group_id
- [ ] AC-3: DeleteNode removes node and all connected relationships
- [ ] AC-4: ListNodes supports cursor-based pagination
- [ ] AC-5: All queries parameterized (no string interpolation)
- [ ] AC-6: Connection pool metrics exposed to Prometheus

## Test Requirements
- **Integration tests**: Neo4j testcontainer
- **Minimum coverage**: 80%
