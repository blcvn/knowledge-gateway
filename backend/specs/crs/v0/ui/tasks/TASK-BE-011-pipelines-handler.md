# TASK-BE-011 — Console Pipelines Handler (NATS JetStream)

| Field | Value |
|---|---|
| **Task ID** | TASK-BE-011 |
| **Layer** | Backend — Go |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-006 CR-009](../solutions/SOL-006-Adaptive-to-Org-Solutions.md) + [SOL-007 §9](../solutions/SOL-007-Gap-Fixes.md) |
| **Priority** | 🟠 P1 |
| **Depends On** | — |
| **Estimated** | 2.5h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `gateway/internal/adapter/handler/console_pipelines_handler.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

```go
package handler

type ConsolePipelinesHandler struct {
    pipelineSvc PipelineServiceClient  // gRPC → pipeline-service
    js          nats.JetStreamContext  // NATS JetStream management API
}

// GET /v1/console/pipelines/queues
// → Query NATS JetStream stream info để lấy pending messages (= queue depth)
func (h *ConsolePipelinesHandler) GetQueues(w http.ResponseWriter, r *http.Request) {
    // Aggregate pending messages across all engine streams
    var totalDepth, totalRetries int
    var throughput float64

    streams := []string{"memobase.ingest", "graphiti.ingest", "cognee.cognify", "sm.ingest", "ov.ingest"}
    for _, stream := range streams {
        info, err := h.js.StreamInfo(stream)
        if err != nil { continue }
        totalDepth   += int(info.State.Msgs)
        totalRetries += int(info.State.NumDeleted) // approximate retries
    }

    httputil.JSON(w, 200, map[string]any{
        "depth":       totalDepth,
        "throughput":  throughput,
        "retry_count": totalRetries,
    })
}

// GET /v1/console/pipelines/status
func (h *ConsolePipelinesHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
    statuses, _ := h.pipelineSvc.GetAllPipelineStatus(r.Context(), authctx.TenantID(r.Context()))
    httputil.JSON(w, 200, statuses)
}

// GET /v1/console/pipelines/workers
func (h *ConsolePipelinesHandler) GetWorkers(w http.ResponseWriter, r *http.Request) {
    workers, _ := h.pipelineSvc.ListWorkers(r.Context(), authctx.TenantID(r.Context()))
    httputil.JSON(w, 200, workers)
}

// GET /v1/console/pipelines/templates
func (h *ConsolePipelinesHandler) GetTemplates(w http.ResponseWriter, r *http.Request) {
    templates, _ := h.pipelineSvc.ListTemplates(r.Context(), authctx.TenantID(r.Context()))
    httputil.JSON(w, 200, templates)
}

// GET /v1/console/pipelines/{engine}/jobs
func (h *ConsolePipelinesHandler) GetJobs(w http.ResponseWriter, r *http.Request) {
    engine   := r.PathValue("engine")
    tenantID := authctx.TenantID(r.Context())
    jobs, _  := h.pipelineSvc.ListJobs(r.Context(), engine, tenantID)
    // Each job includes: id, engine, type, status, progress (0-100 = items_done/items_total*100), created_at, updated_at
    httputil.JSON(w, 200, jobs)
}

// GET /v1/console/pipelines/{engine}/jobs/{id}
func (h *ConsolePipelinesHandler) GetJob(w http.ResponseWriter, r *http.Request) {
    engine := r.PathValue("engine")
    jobID  := r.PathValue("id")
    job, _ := h.pipelineSvc.GetJob(r.Context(), engine, jobID, authctx.TenantID(r.Context()))
    httputil.JSON(w, 200, job)
}
```

### Routes

```go
mux.HandleFunc("GET /v1/console/pipelines/queues",              authMiddleware(pip.GetQueues))
mux.HandleFunc("GET /v1/console/pipelines/status",              authMiddleware(pip.GetStatus))
mux.HandleFunc("GET /v1/console/pipelines/workers",             authMiddleware(pip.GetWorkers))
mux.HandleFunc("GET /v1/console/pipelines/templates",           authMiddleware(pip.GetTemplates))
mux.HandleFunc("GET /v1/console/pipelines/{engine}/jobs",       authMiddleware(pip.GetJobs))
mux.HandleFunc("GET /v1/console/pipelines/{engine}/jobs/{id}",  authMiddleware(pip.GetJob))
```

---

## Verification

```bash
curl http://localhost:8080/v1/console/pipelines/queues \
  -H "Authorization: Bearer <token>" -H "x-tenant-id: <tid>"
# Expected: {"depth":N,"throughput":X,"retry_count":M}

curl http://localhost:8080/v1/console/pipelines/cognee/jobs \
  -H "Authorization: Bearer <token>" -H "x-tenant-id: <tid>"
# Expected: [{id, engine, status, progress, ...}]
```
