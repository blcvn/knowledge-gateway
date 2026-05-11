---
id: TASK-ING-001
title: Implement Domain Layer for Ingestion Service
service: memobase-ingestion
status: DONE
created: 2026-05-11
---

# Task: Implement Domain Layer for Ingestion Service

## Objective
Implement the Domain Layer (Layer 1) for the `memobase-ingestion` service following Clean Architecture principles. This layer must have ZERO external dependencies (except Go standard library).

## Requirements

1. **Blob Entities & Enums**:
   - Define `BlobType` enum/constants (`chat`, `doc`, `summary`).
   - Define `GeneralBlob` struct: `id`, `user_id`, `project_id`, `blob_type`, `blob_data` (JSONB), `add_fields` (JSONB), `created_at`, `updated_at`.

2. **Buffer Zone Entities & FSM**:
   - Define `BufferStatus` enum/constants for FSM states: `idle`, `processing`, `done`, `failed`.
   - Define `BufferZone` struct: `id`, `user_id`, `project_id`, `blob_id`, `blob_type`, `token_size`, `status`, `created_at`, `updated_at`.

3. **Domain Events**:
   - Define `BufferReadyEvent` struct: `user_id`, `project_id`, `buffer_ids` (array of IDs), `blob_type`.

4. **Repository Interfaces**:
   - Define `BlobRepository` interface for CRUD operations on `GeneralBlob`.
   - Define `BufferZoneRepository` interface for FSM transitions and buffer queries (e.g., getting idle entries, updating status to processing).

## Constraints
- No external framework imports in the domain layer.
- Ensure all entities support composite multi-tenancy `(id, project_id)`.
