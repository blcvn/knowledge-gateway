---
id: FEAT-STO-006
title: Neo4j Bulk Operations + Transaction Management
service: graphiti-store
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement atomic bulk operations (SaveBulk, RollbackBulk, DeleteByGroupID) và transaction management wrapper cho Neo4j driver.

## Scope

### In Scope
- `internal/adapter/driver/neo4j/bulk_repo.go` — SaveBulk, RollbackBulk, DeleteByGroupID
- `internal/adapter/driver/neo4j/transaction.go` — WithTransaction wrapper

## Thiết Kế Kỹ Thuật

### SaveBulk (Atomic)
```go
func (r *Neo4jBulkRepo) SaveBulk(ctx context.Context, nodes []domain.EntityNode, edges []domain.EntityEdge, episode domain.EpisodicNode) error {
    return r.driver.WithTransaction(ctx, func(tx neo4j.ExplicitTransaction) error {
        // 1. Create/update all entity nodes
        // 2. Create episode node
        // 3. Create all edges (RELATES_TO)
        // 4. Create MENTIONS edges (episode → entities)
        return nil  // commit on success, auto-rollback on error
    })
}
```

### RollbackBulk
```cypher
// Delete all nodes/edges created by a specific episode
MATCH (e:Episodic {uuid: $episode_id})-[r:MENTIONS]->(entity)
DELETE r
WITH e
MATCH ()-[rel:RELATES_TO {episode_id: $episode_id}]->()
DELETE rel
WITH e
DELETE e
```

## Acceptance Criteria

- [ ] AC-1: SaveBulk persists nodes + edges + episode in single transaction
- [ ] AC-2: Transaction auto-rolls back on any error (no partial writes)
- [ ] AC-3: RollbackBulk removes all artifacts from a specific episode
- [ ] AC-4: DeleteByGroupID removes ALL data for a tenant (purge)
- [ ] AC-5: WithTransaction supports nested operations with single commit point

## Test Requirements
- **Integration tests**: Neo4j testcontainer, failure injection for rollback
- **Minimum coverage**: 80%
