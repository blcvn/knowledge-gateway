---
id: TASK-STO-003
title: Implement Repository and Event Adapters for ov-storage
service: ov-storage
status: Done
---

# TASK-STO-003: Implement Repository and Event Adapters

## Objective
Implement persistence (PostgreSQL + VikingFS) and messaging (NATS) adapters for the `ov-storage` service based on the defined data models.

## Requirements
1. **PostgreSQL Adapters**:
   - Implement metadata repositories mapping to tables: `ov_files`, `ov_file_relations`, `ov_account_keys`, `ov_api_key_hashes`, `ov_resources`, `ov_watch_tasks`.
2. **VikingFS Adapters**:
   - Implement the Go-native VikingFS tiered storage (L0/L1/L2) interfaces.
3. **Event Adapters**:
   - Implement a NATS publisher for events: `ov.content.written`, `ov.content.deleted`, `ov.resource.ingested`.

## Acceptance Criteria
- [x] PostgreSQL adapters use standard drivers and execute queries matching `data-model.md` schemas.
- [x] VikingFS storage layer successfully persists byte streams to local/tiered storage.
- [x] NATS adapter can reliably publish the required messaging contracts to `ov-search`.
