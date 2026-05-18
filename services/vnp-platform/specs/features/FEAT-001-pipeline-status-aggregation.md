---
id: FEAT-001
title: Pipeline Status Aggregation Service
service: vnp-platform
version: 1.0.0
status: Draft
priority: P1
created: 2026-05-13
updated: 2026-05-13
linked_sol: gateway/SOL-002 (T10)
linked_ux: "ux_spec.md §6.9 Pipelines Console"
---

## Mục Tiêu

Aggregate pipeline job status, queue metrics, và worker health từ tất cả 6 engines.

## Scope

### In Scope
- gRPC `PipelineService.GetJobStatus(engine, job_id)` — single job status
- gRPC `PipelineService.ListJobs(engine, status, cursor)` — list jobs per engine
- gRPC `PipelineService.GetQueueMetrics()` — aggregated queue depth/throughput
- gRPC `PipelineService.GetWorkerStatus()` — worker state per engine

### Out of Scope
- Pipeline template management (future)
- Job execution (each engine manages its own jobs)

## Thiết Kế Kỹ Thuật

### Business Logic

1. Fan-out to each engine's pipeline service:
   - `cognee-pipeline.ListJobs()` / `GetStatus()`
   - `graphiti-pipeline.ListJobs()` / `GetStatus()`
   - `memobase-pipeline.ListJobs()` / `GetStatus()`
   - For ov/zep/sm: query their respective internal job tracking
2. Aggregate queue metrics from NATS JetStream consumer info
3. Worker status from engine health endpoints + internal goroutine counts

### Response Schema

```json
{
  "engines": {
    "cognee": {
      "active_jobs": 3,
      "completed_24h": 142,
      "failed_24h": 2,
      "queue_depth": 41,
      "workers": { "running": 4, "idle": 2 },
      "stages": ["Ingest", "Parse", "Chunk", "Embed", "Extract", "Validate", "Store"]
    }
  },
  "total_active_jobs": 12,
  "total_queue_depth": 150
}
```

## Acceptance Criteria
- [ ] AC-1: Returns job status for all 6 engines within 5s
- [ ] AC-2: Queue metrics include depth, throughput, retry count
- [ ] AC-3: Worker status shows running/idle count per engine
- [ ] AC-4: Supports filtering by engine, job status
- [ ] AC-5: Failed engine returns error status, doesn't block others

## Test Requirements
- Unit tests: Aggregation logic, timeout handling
- Integration tests: Mock engine pipeline gRPC services
- Minimum coverage: 80%
