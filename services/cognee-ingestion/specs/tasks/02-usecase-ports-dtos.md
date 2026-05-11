---
id: TASK-ING-02
title: Implement Usecase Ports and DTOs
service: cognee-ingestion
feature: FEAT-ING-001
status: Done
---

## Objective
Define the input/output ports and DTOs for the Usecase layer.

## Files to Create/Update
- `internal/usecase/port/input.go`: Define `FileIngester`, `TextIngester`, `UrlIngester`, `DatasetManager`.
- `internal/usecase/port/output.go`: Define `DatasetRepository`, `DataItemRepository`, `FileStorage`, `TextExtractor`, `EventPublisher`.
- `internal/usecase/dto/request.go`: Define `IngestFileReq`, `IngestTextReq`, `IngestUrlReq`.
- `internal/usecase/dto/response.go`: Define `IngestResult`, `DatasetInfo`.

## Acceptance Criteria
- Ports define clean contracts matching FEAT-ING-001 specifications.
- No dependency on infrastructure types in ports or DTOs.
