---
id: TASK-002
title: Update Docker Compose and Kubernetes Manifests
service: deploy
version: 1.0.0
status: Ready
priority: P2
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Update Docker Compose (dev) and Kubernetes (prod) manifests to reflect the 18-service consolidated architecture.

## Scope

### In Scope
- Remove 17 eliminated service containers from docker-compose.yml
- Add 7 new consolidated service containers
- Update service discovery / DNS entries
- Update NATS stream configuration (29 → 17 subjects)
- Remove Qdrant service from infrastructure
- Update port mappings per consolidated port allocation

### Out of Scope
- Helm chart values (separate task if using Helm)

## Thiết Kế Kỹ Thuật

### Docker Compose Changes

```yaml
# REMOVE containers
# vnp-admin, vnp-event, ov-admin, zep-admin, sm-auth, sm-analytics, sm-project
# cognee-ingestion, cognee-cognify
# graphiti-ingestion, graphiti-knowledge
# memobase-ingestion, memobase-engine
# zep-user, zep-thread, zep-memory (replaced by zep-core)
# ov-fs, ov-crypto, ov-resource (replaced by ov-storage)
# sm-document, sm-memory, sm-profile (replaced by sm-engine)
# sm-mcp (absorbed into gateway)
# qdrant (infrastructure)

# ADD containers
# vnp-platform:9050
# cognee-pipeline:9011
# graphiti-pipeline:9021
# memobase-pipeline:9031
# ov-storage:9051
# zep-core:9061
# sm-engine:9071
```

### NATS Stream Update

```
# Remove subjects (now internal function calls)
- cognee.data.ingested
- cognee.cognify.started / completed
- graphiti.saga.step.*
- memobase.buffer.ready
- memobase.engine.started
- ov.crypto.key.rotated
- zep.thread.created / ended
- zep.user.created
- sm.document.chunked
- sm.memory.extracted
- sm.profile.trait.updated

# Keep subjects (cross-service events)
+ cognee.pipeline.completed
+ graphiti.episode.completed
+ memobase.pipeline.completed / memobase.profile.changed
+ ov.content.written / ov.content.deleted / ov.resource.ingested / ov.session.*
+ zep.memory.messages.ingested / zep.user.deleted / zep.graph.* / zep.search.*
+ sm.engine.document.created / sm.engine.memory.created / sm.engine.memory.forgotten
+ sm.connector.synced
+ admin.tenant.created / admin.tenant.deleted
```

## Acceptance Criteria

- [ ] AC-1: `docker-compose up` starts exactly 18 service containers + infrastructure
- [ ] AC-2: No Qdrant container in docker-compose
- [ ] AC-3: NATS streams configured with 17 subjects (down from 29)
- [ ] AC-4: All services healthy and reachable via gateway
