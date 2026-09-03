---
id: TASK-RES-003
service: ov-resource
status: Done
---

# TASK-RES-003: Implement Infrastructure Persistence

## Objective
Implement PostgreSQL repositories (`internal/infra/persistence/`) and data models based exactly on `data-model.md`.

## Requirements
1. **`ov_resources` Table & Repository**:
   - Fields: `id` (UUID, PK), `account_id` (NOT NULL), `source_path`, `target_path`, `filename`, `mime_type`, `parser_type`, `chunk_count` (default 0), `total_tokens` (default 0), `content_hash` (NOT NULL), `status` (pending/processing/completed/failed), `error_message`, `parse_duration_ms`, `ingested_at`, `created_at`.
   - Implement CRUD, status transitions, and deduplication logic leveraging `content_hash`.
2. **`ov_watch_tasks` Table & Repository**:
   - Fields: `id` (UUID, PK), `account_id`, `source_path`, `target_path`, `patterns` (JSONB default '["**/*"]'), `poll_interval_ms` (BIGINT default 30000), `status` (active/paused/deleted), `last_poll_at`, `files_tracked`, `created_at`, `updated_at`.
   - Implement query methods for polling scheduler.
3. **Index Strategy**:
   - Create `idx_resources_account` (`account_id`), `idx_resources_hash` (`account_id`, `content_hash`) for deduplication.
   - Create `idx_resources_status` (`status`) for queue processing.
   - Create `idx_watch_active` (`status`, `last_poll_at`) for polling efficiency.
4. **Data Isolation**:
   - Enforce `account_id` tenant scoping on all queries.

## Dependencies
- PostgreSQL Database.
