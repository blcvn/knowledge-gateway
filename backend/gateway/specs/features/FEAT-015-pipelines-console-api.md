---
id: FEAT-015
title: Pipelines Console API
service: vnp-gateway
version: 1.0.0
status: Draft
priority: P1
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
linked_ux: "ux_spec.md §6.9 Pipelines Console"
---

## Mục Tiêu

REST APIs cho Pipelines Console — DAG view per engine, queue monitoring, worker status, pipeline templates. Cho phép observe ingestion & processing pipelines across all 6 engines.

## Bối Cảnh Nghiệp Vụ

Pipelines Console (UX §6.9) cần:
1. **DAG View** — pipeline stages per engine (like Airflow/Temporal)
2. **Queue Monitoring** — queue depth, retries, failures, throughput per engine
3. **Worker Status** — realtime worker state per engine
4. **Pipeline Templates** — pre-built ingestion templates

Pipeline stages per engine:
- Cognee: Ingest → Parse → Chunk → Embed → Extract → Validate → Store
- Graphiti: Episode → Extract → Deduplicate → Graph → Community
- Zep: Message → Graph Ingestion → Context Assembly
- OpenViking: Data → Parse → Chunk → Embed → L0/L1/L2 Context
- Memobase: Blob → Buffer → Extract → Merge → Profile → Cache
- Supermemory: Document → Memory → Version → Search Index

## Scope

### In Scope
- `GET /v1/console/pipelines/status` — All engines pipeline overview
- `GET /v1/console/pipelines/{engine}` — Engine-specific pipeline status
- `GET /v1/console/pipelines/{engine}/jobs` — Active/recent jobs
- `GET /v1/console/pipelines/{engine}/jobs/{id}` — Job detail with stage progress
- `GET /v1/console/pipelines/queues` — Queue metrics across engines
- `GET /v1/console/pipelines/workers` — Worker status across engines
- `GET /v1/console/pipelines/templates` — Available pipeline templates

### Out of Scope
- Pipeline execution trigger (engine responsibility)
- Pipeline template creation (admin only, future scope)

## Thiết Kế Kỹ Thuật

### API Contract

#### GET `/v1/console/pipelines/status`

**Response (200):**
```json
{
  "engines": {
    "cognee": {
      "active_jobs": 3,
      "completed_24h": 142,
      "failed_24h": 2,
      "avg_duration_ms": 45000,
      "stages": ["ingest", "parse", "chunk", "embed", "extract", "validate", "store"]
    },
    "graphiti": {
      "active_jobs": 1,
      "completed_24h": 89,
      "failed_24h": 0,
      "avg_duration_ms": 32000,
      "stages": ["episode", "extract", "deduplicate", "graph", "community"]
    }
  }
}
```

#### GET `/v1/console/pipelines/{engine}/jobs/{id}`

**Response (200):**
```json
{
  "job_id": "job_abc123",
  "engine": "cognee",
  "status": "running",
  "stages": [
    { "name": "ingest", "status": "completed", "duration_ms": 200, "items_processed": 1 },
    { "name": "parse", "status": "completed", "duration_ms": 1500, "items_processed": 12 },
    { "name": "chunk", "status": "running", "duration_ms": null, "items_processed": 8, "items_total": 12 },
    { "name": "embed", "status": "pending", "duration_ms": null, "items_processed": 0 },
    { "name": "extract", "status": "pending", "duration_ms": null, "items_processed": 0 },
    { "name": "validate", "status": "pending", "duration_ms": null, "items_processed": 0 },
    { "name": "store", "status": "pending", "duration_ms": null, "items_processed": 0 }
  ],
  "started_at": "2026-05-13T12:00:00Z",
  "updated_at": "2026-05-13T12:01:45Z"
}
```

#### GET `/v1/console/pipelines/queues`

**Response (200):**
```json
{
  "queues": {
    "cognee": { "depth": 41, "retries": 3, "failures_24h": 2, "throughput_per_min": 12.5 },
    "graphiti": { "depth": 12, "retries": 0, "failures_24h": 0, "throughput_per_min": 8.3 },
    "zep": { "depth": 3, "retries": 0, "failures_24h": 0, "throughput_per_min": 15.0 },
    "openviking": { "depth": 84, "retries": 5, "failures_24h": 1, "throughput_per_min": 3.2 },
    "memobase": { "depth": 7, "retries": 0, "failures_24h": 0, "throughput_per_min": 20.1 },
    "supermemory": { "depth": 15, "retries": 1, "failures_24h": 0, "throughput_per_min": 5.5 }
  }
}
```

### Internal Architecture
- **Handler:** `adapter/http/pipeline_handler.go`
- **Proxy to:** `vnp-platform` (pipeline aggregation), individual engine pipeline services
- **Source:** NATS JetStream consumer info for queue metrics
- Worker status: gRPC health checks + NATS consumer group info

## Acceptance Criteria
- [ ] AC-1: Pipeline status returns overview for all 6 engines with stage definitions
- [ ] AC-2: Job detail shows per-stage progress with timing
- [ ] AC-3: Queue metrics reflect actual NATS JetStream state
- [ ] AC-4: Worker status shows active/idle workers per engine
- [ ] AC-5: Pipeline templates list available ingestion patterns
- [ ] AC-6: All endpoints require admin role

## Test Requirements
- Unit tests: Pipeline stage mapping, queue metrics aggregation
- Integration tests: Mock vnp-platform + NATS consumer info
- Minimum coverage: 80%
