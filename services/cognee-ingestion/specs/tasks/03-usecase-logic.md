---
id: TASK-ING-03
title: Implement Usecase Logic
service: cognee-ingestion
feature: FEAT-ING-001
status: Done
---

## Objective
Implement the core usecase business logic utilizing the defined ports and DTOs.

## Files to Create/Update
- `internal/usecase/ingest_file.go`: Implement `IngestFile`.
- `internal/usecase/ingest_text.go`: Implement `IngestText`.
- `internal/usecase/ingest_url.go`: Implement `IngestUrl`.
- `internal/usecase/manage_dataset.go`: Implement `ManageDataset`.
- Related `*_test.go` files using `mockgen` for port mocking.

## Acceptance Criteria
- Implement logic flows as described in FEAT-ING-001 (e.g., validate -> hash -> upload -> extract -> persist -> publish).
- Unit tests using mocked ports cover all methods with >= 80% coverage.
- Imports only domain and usecase/port/dto paths.
