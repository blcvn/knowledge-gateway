# TASK-CONSOLE-004 — Sessions Explorer & Pipeline Monitor Handlers

| Field | Value |
|---|---|
| **Task ID** | TASK-CONSOLE-004 |
| **Wave** | 2 |
| **Solution** | [SOL-CONSOLE-003](../solutions/SOL-CONSOLE-003-Sessions-Explorer-APIs.md), [SOL-CONSOLE-004](../solutions/SOL-CONSOLE-004-Pipeline-Monitor-APIs.md) |
| **Component** | `gateway/adapter/handler/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 4h |

---

## Mục tiêu

Tạo `SessionsHandler` (proxy to observe-service) và `PipelineHandler` (NATS JetStream + pipeline-service).

---

## Công việc cụ thể

### 1. Tạo `gateway/adapter/handler/sessions_handler.go` [NEW]

6 endpoints, tất cả proxy tới `observe-service` gRPC:
- `GET /v1/console/sessions` → `ListSessions(status, limit, offset)`
- `GET /v1/console/sessions/{id}` → `GetSession`
- `GET /v1/console/sessions/{id}/timeline` → `GetObservations`
- `GET /v1/console/sessions/{id}/diff` → `GetMemoryDiff`
- `GET /v1/console/sessions/{id}/working-memory` → `GetWorkingMemory`
- `GET /v1/console/sessions/{id}/user-summary` → `GetSessionSummary`

### 2. Tạo `gateway/adapter/handler/pipeline_handler.go` [NEW]

5 endpoints:
- `GET /v1/console/pipelines/status` → parallel engine stats from pipeline-service gRPC
- `GET /v1/console/pipelines/queues` → NATS JetStream stream stats
- `GET /v1/console/pipelines/workers` → Redis `worker:*` heartbeat keys
- `GET /v1/console/pipelines/{engine}/jobs` → pipeline-service.ListJobs
- `GET /v1/console/pipelines/{engine}/jobs/{id}` → pipeline-service.GetJob

NATS JetStream helper:
```go
func (h *PipelineHandler) getQueueDepth(js nats.JetStreamContext, subject string) (uint64, error) {
    info, err := js.StreamInfo(subjectToStream(subject))
    if err != nil { return 0, err }
    return info.State.Msgs, nil
}
```

### 3. Routes `router.go` [MODIFY]

```go
// Sessions
r.Get("/v1/console/sessions",                          sessionsHandler.ListSessions)
r.Get("/v1/console/sessions/{id}",                     sessionsHandler.GetSession)
r.Get("/v1/console/sessions/{id}/timeline",            sessionsHandler.GetTimeline)
r.Get("/v1/console/sessions/{id}/diff",                sessionsHandler.GetMemoryDiff)
r.Get("/v1/console/sessions/{id}/working-memory",      sessionsHandler.GetWorkingMemory)
r.Get("/v1/console/sessions/{id}/user-summary",        sessionsHandler.GetUserSummary)

// Pipelines
r.Get("/v1/console/pipelines/status",                  pipelineHandler.GetStatus)
r.Get("/v1/console/pipelines/queues",                  pipelineHandler.GetQueues)
r.Get("/v1/console/pipelines/workers",                 pipelineHandler.GetWorkers)
r.Get("/v1/console/pipelines/{engine}/jobs",           pipelineHandler.GetJobs)
r.Get("/v1/console/pipelines/{engine}/jobs/{id}",      pipelineHandler.GetJob)
```

---

## Acceptance Criteria

- [ ] `GET /sessions?status=live` → only live sessions
- [ ] `GET /sessions/{id}/timeline` → events ordered by index
- [ ] `GET /pipelines/status` → per-engine health (3s timeout)
- [ ] `GET /pipelines/queues` → real NATS JetStream pending counts
- [ ] Worker liveness: only workers with non-expired Redis key shown

## Files

```
gateway/adapter/handler/sessions_handler.go   [NEW]
gateway/adapter/handler/pipeline_handler.go   [NEW]
gateway/adapter/handler/router.go             [MODIFY]
```
