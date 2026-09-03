---
id: TASK-PIPE-005
title: Implement Database Schema & Migrations
layer: database
status: Done
---

## Objective
Implement the initial PostgreSQL database schema migrations based on `data-model.md` and Domain entities.

## Requirements
1. **Schema Definition**: Create SQL migration files for `Blobs`, `BufferState`, `Profiles`, and `EventGists`.
2. **Vector Extension**: Ensure `pgvector` extension is enabled and vector columns are appropriately sized for Bifrost embeddings.
3. **Indexes Strategy**: Define and create indexes based on query patterns, explicitly ensuring indexing on `tenant_id` for isolation.

## Constraints
- Do not implement application logic here, only raw SQL migration files.
