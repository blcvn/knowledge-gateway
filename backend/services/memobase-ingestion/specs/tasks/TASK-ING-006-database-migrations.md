---
id: TASK-ING-006
title: Implement Database Migrations
service: memobase-ingestion
status: DONE
created: 2026-05-11
---

# Task: Implement Database Migrations

## Objective
Implement the database schema migrations for PostgreSQL as defined in the data model.

## Requirements

1. **general_blobs Table**:
   - `id` (UUID, PK)
   - `user_id` (VARCHAR(255), NOT NULL, FK to users conceptually)
   - `project_id` (VARCHAR(255), NOT NULL, FK to projects conceptually)
   - `blob_type` (VARCHAR(20), NOT NULL)
   - `blob_data` (JSONB, NOT NULL)
   - `add_fields` (JSONB)
   - `created_at` (TIMESTAMPTZ, NOT NULL, DEFAULT NOW())
   - `updated_at` (TIMESTAMPTZ, NOT NULL)

2. **buffer_zones Table**:
   - `id` (UUID, PK)
   - `user_id` (VARCHAR(255), NOT NULL)
   - `project_id` (VARCHAR(255), NOT NULL)
   - `blob_id` (UUID, FK to general_blobs.id)
   - `blob_type` (VARCHAR(20), NOT NULL)
   - `token_size` (INTEGER, NOT NULL)
   - `status` (VARCHAR(20), NOT NULL, DEFAULT 'idle')
   - `created_at` (TIMESTAMPTZ, NOT NULL, DEFAULT NOW())
   - `updated_at` (TIMESTAMPTZ, NOT NULL)

3. **Indexes**:
   - Create index `idx_blobs_user_type` on `general_blobs` `(user_id, project_id, blob_type)`.
   - Create index `idx_buffer_user_status` on `buffer_zones` `(user_id, project_id, blob_type, status)`.
   - Create partial index `idx_buffer_status_idle` on `buffer_zones` `(status) WHERE status='idle'`.

## Constraints
- Use proper migration tools (e.g., golang-migrate).
- Support up and down migrations.
- Respect the composite PK/tenant isolation requirements.
