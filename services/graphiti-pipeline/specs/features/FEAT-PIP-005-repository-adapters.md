---
id: FEAT-PIP-005
title: Repository Adapters — PostgreSQL + Neo4j Reader
service: graphiti-pipeline
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement PostgreSQL repository adapters (saga state, episodes, dedup) và Neo4j reader adapters (entity/edge similarity for resolution).

## Scope

### In Scope
- `internal/adapter/repository/postgres/episode_repo.go` — Episode CRUD + dedup
- `internal/adapter/repository/postgres/saga_repo.go` — Saga state persistence + transition
- `internal/adapter/repository/postgres/migrations/` — SQL migration files
- `internal/adapter/repository/neo4j/entity_reader.go` — FindSimilarEntities, GetEntityByName
- `internal/adapter/repository/neo4j/community_reader.go` — Community detection queries

### Out of Scope
- Neo4j write operations (via graphiti-store service)

## Acceptance Criteria

- [ ] AC-1: Saga state transitions persisted atomically in PostgreSQL
- [ ] AC-2: Episode dedup prevents duplicate ingestion (409 on content_hash collision)
- [ ] AC-3: Neo4j entity similarity search uses cosine index on name_embedding
- [ ] AC-4: All queries scoped by group_id (multi-tenant isolation)
- [ ] AC-5: Connection pool health metrics exposed to Prometheus
- [ ] AC-6: SQL migrations versioned and idempotent

## Test Requirements
- **Unit tests**: Repository methods with mock database (sqlmock + neo4j mock)
- **Integration tests**: PostgreSQL + Neo4j testcontainers
- **Minimum coverage**: 80%
