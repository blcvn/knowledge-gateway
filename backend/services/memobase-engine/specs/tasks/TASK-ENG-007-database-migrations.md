---
id: TASK-ENG-007
title: Implement Database Migrations (pgvector, profiles, events)
service: memobase-engine
layer: Database
status: Done
---

# Task: Implement Database Migrations

## Objective
Implement the database schema migrations based on the `data-model.md` and `runbook.md` requirements.

## Requirements
1. **Extensions**:
   - Ensure `CREATE EXTENSION IF NOT EXISTS vector;` is executed.

2. **Tables**:
   - Create `user_profiles` table with composite PK `(id, project_id)`.
   - Create `user_events` table with `vector` embedding type.
   - Create `user_event_gists` table with `vector` embedding type.

3. **Indexes**:
   - Create B-tree index `idx_profiles_user` on `user_profiles(user_id, project_id)`.
   - Create B-tree index `idx_events_user` on `user_events(user_id, project_id)`.
   - Create pgvector index `idx_events_embedding` on `user_events.embedding` using HNSW/IVFFlat.
   - Create B-tree index `idx_gists_event` on `user_event_gists(user_id, project_id, event_id)`.
   - Create pgvector index `idx_gists_embedding` on `user_event_gists.embedding` using HNSW/IVFFlat.

## Constraints
- Align strictly with the table structure and constraints defined in `data-model.md`.
