---
id: FEAT-SEA-005
title: Redis Cache + NATS Cache Invalidation
service: graphiti-search
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement Redis cache adapter cho search results (TTL-based) và NATS subscriber cho cache invalidation khi có episode mới được ingest.

## Scope

- `internal/adapter/cache/redis_cache.go` — Get/Set with TTL, key = hash(query + group_id)
- `internal/adapter/event/nats_subscriber.go` — Subscribe `graphiti.episode.ingested` → invalidate cache per group_id

### Cache Strategy

| Aspect | Design |
|--------|--------|
| Key format | `search:{group_id}:{sha256(query+methods+rerankers)}` |
| TTL | Configurable (default 5min) |
| Invalidation | On `graphiti.episode.ingested` → delete all keys matching `search:{group_id}:*` |
| Serialization | Protobuf for compact encoding |

## Acceptance Criteria

- [ ] AC-1: Cache hit returns results in <10ms (no search execution)
- [ ] AC-2: Cache miss executes full search pipeline and stores result
- [ ] AC-3: NATS subscriber invalidates all cache entries for affected group_id
- [ ] AC-4: Cache keys scoped by group_id (tenant isolation)
- [ ] AC-5: Redis unavailability degrades gracefully (search still works, just uncached)
- [ ] AC-6: Cache hit ratio metric exposed to Prometheus

## Test Requirements
- **Integration tests**: Redis testcontainer + NATS testcontainer
- **Minimum coverage**: 80%
