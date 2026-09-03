---
id: FEAT-ING-004
title: PostgreSQL Repository — Saga State + Episodes
service: graphiti-ingestion
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

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

- [ ] AC-1: Saga state transitions persisted atomically
- [ ] AC-2: Episode dedup prevents duplicate ingestion by content_hash
- [ ] AC-3: All queries scoped by group_id
- [ ] AC-4: Migrations versioned and idempotent
- [ ] AC-5: Connection pool metrics exposed

## Test Requirements
- **Integration tests**: PostgreSQL testcontainer
- **Minimum coverage**: 80%
