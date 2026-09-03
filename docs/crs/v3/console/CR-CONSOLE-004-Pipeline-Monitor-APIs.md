# Change Request: CR-CONSOLE-004 — Pipeline Monitor Backend APIs

**CR ID:** CR-CONSOLE-004
**Component:** `backend/gateway`, `backend/services/pipeline-service`, `backend/services/vnp-observability`
**Priority:** 🟠 Medium
**Status:** Open
**Version:** v3 / Console
**Feature:** [F23](../../../features/23-pipeline-monitor/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P2-05 | Platform Engineer | Pipeline failures im lặng — không biết |

---

## 2. APIs

### `GET /v1/console/pipelines/status`

```json
{
  "overall": "healthy",
  "engines": {
    "cognee": {"status": "healthy", "active_jobs": 3, "queue_depth": 12},
    "graphiti": {"status": "degraded", "active_jobs": 0, "queue_depth": 0, "error": "LLM timeout"}
  },
  "consolidation": {
    "status": "healthy", "pending_sessions": 5, "last_run_ago_seconds": 120
  }
}
```

### `GET /v1/console/pipelines/queues`

NATS queue depths per subject:
```json
{
  "subjects": {
    "cognee.ingest": {"pending": 12, "processing": 3},
    "graphiti.ingest": {"pending": 0, "processing": 1},
    "memory.consolidation": {"pending": 5, "processing": 1}
  }
}
```

### `GET /v1/console/pipelines/workers`

```json
{
  "workers": [
    {"id": "worker-1", "engine": "cognee", "status": "busy", "current_job": "job-abc"},
    {"id": "worker-2", "engine": "graphiti", "status": "idle"}
  ]
}
```

### `GET /v1/console/pipelines/{engine}/jobs?status=failed&limit=20`

Job list for specific engine.

### `GET /v1/console/pipelines/{engine}/jobs/{id}`

Job detail: input, output, error, duration, retry count.

---

## 3. Implementation Notes

- Queue depths: query NATS JetStream stream stats
- Job records: pipeline-service DB
- Worker status: heartbeat via NATS (10s TTL)

---

## 4. Acceptance Criteria

- [ ] Queue depths real-time from NATS JetStream
- [ ] Failed jobs: error message + retry count
- [ ] Per-engine status aggregated
- [ ] Job detail shows full input/output/error
