---
id: TASK-ING-004
title: Implement PostgreSQL Repository — Saga State + Episodes
service: graphiti-ingestion
type: task
status: done
priority: P0
created: 2026-05-11
dependencies: [TASK-ING-002]
estimated_time: 4h
linked_feat: FEAT-ING-004
---

## Objective
Implement PostgreSQL repository adapters cho saga state persistence và episode metadata/dedup.

## Scope
- `internal/adapter/repository/postgres/saga_repo.go` — Create, Get, Update saga state
- `internal/adapter/repository/postgres/episode_repo.go` — Episode CRUD + dedup lookup
- `internal/adapter/repository/postgres/migrations/` — SQL migrations

### Tables
| Table | Purpose |
|-------|---------|
| `graphiti_saga_state` | Saga step tracking, status, retry counts |
| `graphiti_episodes` | Episode metadata |
| `graphiti_episode_dedup` | Content hash → episode_id dedup index |

## Acceptance Criteria
- [x] Saga state transitions persisted atomically
- [x] Episode dedup prevents duplicate ingestion by content_hash
- [x] All queries scoped by group_id
- [x] Migrations versioned and idempotent
- [x] Connection pool metrics exposed

## Test Requirements
- Integration tests: PostgreSQL testcontainer
- Minimum coverage: 80%
