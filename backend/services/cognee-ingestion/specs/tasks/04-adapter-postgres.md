---
id: TASK-ING-04
title: Implement PostgreSQL Repositories
service: cognee-ingestion
feature: FEAT-ING-002
status: Done
---

## Objective
Implement PostgreSQL adapters for dataset and data item repositories.

## Files to Create/Update
- `internal/adapter/repository/postgres/dataset_repo.go`: Implement `DatasetRepository`.
- `internal/adapter/repository/postgres/dataitem_repo.go`: Implement `DataItemRepository`.
- Create PostgreSQL schemas for `datasets` and `data_items` based on FEAT-ING-002 (with RLS policies).
- Related `*_test.go` files utilizing `testcontainers`.

## Acceptance Criteria
- Repositories correctly implement the usecase output ports.
- Queries account for Row Level Security (RLS) via the `tenant_id` context.
- Integration tests using testcontainers pass with >= 80% coverage.
