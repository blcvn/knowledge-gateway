---
id: TASK-COG-006
title: Implement PostgreSQL & Qdrant Adapters
feature: FEAT-COG-002
status: Done
---
# Task: Implement PostgreSQL & Qdrant Repositories

## Objective
Implement persistence adapters for job state tracking in PostgreSQL and vector storage in Qdrant.

## Files to Create/Modify
- `internal/adapter/repository/postgres/job_repo.go`
- `internal/adapter/repository/qdrant/vector_repo.go`

## Requirements
- `JobRepository`: Map the domain `CognifyJob` object to the DB schema. Implement standard CRUD operations (Create, Update, Get) to track job state and metadata.
- `VectorRepository`: Connect to Qdrant. Implement operations to upsert text chunk and entity embeddings. 
- Ensure proper tenant isolation in Qdrant by applying a `tenant_id` payload filter on all search/upsert operations.
- Provide integration tests using appropriate testcontainers.
