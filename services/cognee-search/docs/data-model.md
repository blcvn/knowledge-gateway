---
id: DOC-S04
service: cognee-search
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
---

# cognee-search — Data Model

> **Database**: Qdrant (vectors), Neo4j (graph traversal), Redis (cache)

## Qdrant Collections (Read-Only)

cognee-search reads from collections created by cognee-cognify:

- `cognee_chunks` — Text chunk embeddings (1536-dim)
- `cognee_entities` — Entity description embeddings (1536-dim)

## Neo4j Queries (Read-Only)

Traversal queries on Entity, Chunk, Community nodes. Uses Cypher for graph completion.

## Redis Cache

| Key Pattern | TTL | Description |
|------------|-----|-------------|
| `search:{tenant_id}:{query_hash}:{type}` | 5min | Cached search results |
| `rag:{tenant_id}:{query_hash}` | 10min | Cached RAG completions |

## Migration History

| Version | Date | Description |
|---------|------|-------------|
| 1.0.0 | 2026-05-09 | Initial: read-only access to cognify collections |
