---
id: FEAT-STO-005
title: Neo4j Search Primitives — Cosine, Fulltext, BFS
service: graphiti-store
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement 3 search primitives trong Neo4j: cosine similarity (vector index), fulltext search (BM25 index), và BFS graph traversal.

## Scope

### In Scope
- `internal/adapter/driver/neo4j/search_repo.go`
- CosineSimilaritySearch: vector index query trên name_embedding / fact_embedding
- FulltextSearch: BM25 index query trên name, summary, fact fields
- BFSSearch: variable-length path traversal to configurable depth
- All searches scoped by group_id

### Out of Scope
- Reranking (graphiti-search responsibility)
- Embedding generation (graphiti-pipeline responsibility)

## Thiết Kế Kỹ Thuật

### Cosine Similarity

```cypher
CALL db.index.vector.queryNodes($index_name, $limit, $embedding)
YIELD node, score
WHERE node.group_id = $group_id
RETURN node, score ORDER BY score DESC
```

### Fulltext Search (BM25)

```cypher
CALL db.index.fulltext.queryNodes($index_name, $query)
YIELD node, score
WHERE node.group_id = $group_id
RETURN node, score ORDER BY score DESC LIMIT $limit
```

### BFS Traversal

```cypher
MATCH path = (start:Entity {uuid: $start_id})-[*1..$depth]-(connected)
WHERE ALL(r IN relationships(path) WHERE r.group_id = $group_id)
  AND connected.group_id = $group_id
RETURN DISTINCT connected, min(length(path)) AS distance
ORDER BY distance ASC LIMIT $limit
```

## Acceptance Criteria

- [ ] AC-1: CosineSimilaritySearch returns top-K entities ordered by embedding similarity
- [ ] AC-2: FulltextSearch returns BM25-ranked results from name/summary/fact
- [ ] AC-3: BFSSearch traverses graph to specified depth, returns connected entities
- [ ] AC-4: All searches enforce group_id scoping
- [ ] AC-5: Search results include score/distance for ranking

## Test Requirements
- **Integration tests**: Neo4j testcontainer with pre-loaded data + indexes
- **Minimum coverage**: 80%
