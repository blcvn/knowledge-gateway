---
id: TASK-PIP-005
title: Implement Repository Adapters
feature: FEAT-PIP-005
status: Done
---

## Objective
Thực thi implement PostgreSQL + Neo4j repository adapters dựa trên FEAT-PIP-005.

## Tasks
1. Tạo files PostgreSQL migrations (`internal/adapter/repository/postgres/migrations/`).
   - SQL schema cho saga state, episodes, dedup.

2. Tạo file `internal/adapter/repository/postgres/episode_repo.go`
   - Implement Episode CRUD và logic dedup (409 on content_hash collision).
   - Truy vấn scoped by `group_id`.

3. Tạo file `internal/adapter/repository/postgres/saga_repo.go`
   - Implement Saga state persistence và atomic transition.
   - Truy vấn scoped by `group_id`.

4. Tạo file `internal/adapter/repository/neo4j/entity_reader.go`
   - Implement `FindSimilarEntities` (sử dụng cosine index trên name_embedding).
   - Implement `GetEntityByName`.
   - Truy vấn scoped by `group_id`.

5. Tạo file `internal/adapter/repository/neo4j/community_reader.go`
   - Implement community detection queries.
   - Truy vấn scoped by `group_id`.

6. Metrics and Tests
   - Expose PostgreSQL connection pool health metrics cho Prometheus.
   - Viết unit tests với mock databases (sqlmock + neo4j mock).
   - Viết integration tests với PostgreSQL + Neo4j testcontainers.
   - Đảm bảo coverage >= 80%.
