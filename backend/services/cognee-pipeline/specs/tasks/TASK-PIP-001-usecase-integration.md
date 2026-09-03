---
id: TASK-PIP-001
title: "Implement Domain & Usecase Integration"
service: cognee-pipeline
status: Done
priority: P1
linked_feat: FEAT-PIP-001
---

## Objective
Merge and consolidate the domain and usecase layers of `cognee-ingestion` and `cognee-cognify` into the single `cognee-pipeline` service, utilizing a local function call to trigger cognify instead of NATS events.

## Scope
1. **Domain Layer**:
   - Set up `internal/domain/ingestion/` (Dataset, DataItem, etc.) and `internal/domain/cognify/` (CognifyJob, Entity, Relationship).
   - Resolve imports from existing standalone modules where applicable to reuse entities and logic.
2. **Usecase Layer**:
   - Set up `internal/usecase/ingest/` and `internal/usecase/cognify/`.
   - Modify the ingestion flow (`IngestFileUseCase`) to execute a direct, local goroutine call to `CognifyUseCase` instead of publishing the `cognee.data.ingested` NATS event.
   - Example: `if uc.autoTriggerCognify { go uc.cognifyUseCase.Execute(...) }`.
3. **Ports**:
   - Consolidate required adapter interfaces in `internal/usecase/port/interfaces.go`.

## Acceptance Criteria
- [x] Domain entities for both ingestion and cognify are accessible in `cognee-pipeline`.
- [x] Ingestion usecases can directly invoke cognify usecases without NATS.
- [x] No breaking changes to existing business logic inside the combined usecases.
