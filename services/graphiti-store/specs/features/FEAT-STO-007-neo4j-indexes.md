---
id: FEAT-STO-007
title: Neo4j Index Management
service: graphiti-store
version: 1.0.0
status: Ready
priority: P1
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement index management adapter cho Neo4j: vector indexes (cosine), fulltext indexes (BM25), composite indexes, và range indexes cho bi-temporal queries.

## Scope

- `internal/adapter/driver/neo4j/index_repo.go` — BuildIndices, DropIndices, ListIndices

### Indexes to Create

| Index Name | Type | Target | Properties |
|-----------|------|--------|-----------|
| `entity_name_embedding` | Vector (cosine) | Entity | name_embedding |
| `edge_fact_embedding` | Vector (cosine) | RELATES_TO | fact_embedding |
| `entity_name_fulltext` | Fulltext | Entity | name, summary |
| `edge_fact_fulltext` | Fulltext | RELATES_TO | name, fact |
| `entity_group_id` | Range | Entity | group_id |
| `edge_temporal` | Composite | RELATES_TO | group_id, valid_at, invalid_at |

## Acceptance Criteria

- [ ] AC-1: BuildIndices creates all 6 indexes idempotently (IF NOT EXISTS)
- [ ] AC-2: DropIndices removes all indexes for a group (cleanup)
- [ ] AC-3: ListIndices returns current index definitions
- [ ] AC-4: Vector index uses correct dimension from config

## Test Requirements
- **Integration tests**: Neo4j testcontainer
- **Minimum coverage**: 70%
