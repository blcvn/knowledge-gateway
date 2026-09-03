# Solution: SOL-CONSOLE-004 — Pipeline Monitor Backend APIs

**CR:** CR-CONSOLE-004
**TDD refs:** `architecture/12-agentmemory-services.md §pipeline-service`, `models/pipeline-service.md`
**Version:** v3/console

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** PipelineHandler: ListRuns/GetRun/GetSteps/ListTemplates
---

## 1. Architecture

Pipeline data comes from:
- `pipeline-service` gRPC → job records, worker heartbeats
- NATS JetStream → queue depths (stream stats API)
- Redis → worker heartbeat keys (TTL-based liveness)

---

## 2. Implementation

```go
// gateway/adapter/handler/pipeline_handler.go [NEW]
type PipelineHandler struct {
    registry port.GRPCRegistry
    nats     *nats.Conn
    redis    *redis.Client
}

// GET /v1/console/pipelines/status
func (h *PipelineHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
    defer cancel()

    type engineStatus struct {
        Status     string `json:"status"`
        ActiveJobs int    `json:"active_jobs"`
        QueueDepth int    `json:"queue_depth"`
        Error      string `json:"error,omitempty"`
    }
    engines := []string{"cognee", "graphiti", "zep", "memobase", "openviking", "supermemory"}
    results := map[string]engineStatus{}
    var mu sync.Mutex
    var wg sync.WaitGroup

    for _, eng := range engines {
        wg.Add(1)
        go func(e string) {
            defer wg.Done()
            // Check active job count from pipeline-service
            conn, err := h.registry.Get("pipeline-service")
            if err != nil {
                mu.Lock(); results[e] = engineStatus{Status: "unavailable"}; mu.Unlock()
                return
            }
            client := pipelinepb.NewPipelineServiceClient(conn)
            stats, err := client.GetEngineStats(ctx, &pipelinepb.EngineStatsRequest{Engine: e})
            status := engineStatus{Status: "healthy"}
            if err != nil { status.Status = "unknown"; status.Error = err.Error() }
            if stats != nil {
                status.ActiveJobs = int(stats.ActiveJobs)
                status.QueueDepth = int(stats.QueueDepth)
                if stats.LastError != "" { status.Status = "degraded"; status.Error = stats.LastError }
            }
            mu.Lock(); results[e] = status; mu.Unlock()
        }(eng)
    }
    wg.Wait()

    overall := "healthy"
    for _, s := range results {
        if s.Status == "unavailable" { overall = "unhealthy" }
        if s.Status == "degraded" && overall != "unhealthy" { overall = "degraded" }
    }
    writeJSON(w, 200, map[string]any{"overall": overall, "engines": results})
}

// GET /v1/console/pipelines/queues
func (h *PipelineHandler) GetQueues(w http.ResponseWriter, r *http.Request) {
    // Query NATS JetStream stream stats
    js, err := h.nats.JetStream()
    if err != nil { writeError(w, 500, "nats_error", err.Error()); return }

    subjects := []string{
        "cognee.ingest", "graphiti.ingest", "zep.ingest",
        "memory.consolidation", "memory.blob.inserted",
    }
    result := map[string]map[string]uint64{}
    for _, subj := range subjects {
        info, err := js.StreamInfo(subjectToStream(subj))
        if err != nil { result[subj] = map[string]uint64{"pending": 0}; continue }
        result[subj] = map[string]uint64{
            "pending":    info.State.Msgs,
            "processing": info.State.NumPending,
        }
    }
    writeJSON(w, 200, map[string]any{"subjects": result})
}

// GET /v1/console/pipelines/workers
func (h *PipelineHandler) GetWorkers(w http.ResponseWriter, r *http.Request) {
    // Workers publish heartbeat to Redis: worker:{name} = {status} with 15s TTL
    keys, _ := h.redis.Keys(r.Context(), "worker:*").Result()
    workers := []map[string]string{}
    for _, key := range keys {
        val, _ := h.redis.Get(r.Context(), key).Result()
        var w map[string]string
        json.Unmarshal([]byte(val), &w)
        workers = append(workers, w)
    }
    writeJSON(w, 200, map[string]any{"workers": workers})
}

// GET /v1/console/pipelines/{engine}/jobs?status=failed&limit=20
func (h *PipelineHandler) GetJobs(w http.ResponseWriter, r *http.Request) {
    engine := chi.URLParam(r, "engine")
    status := r.URL.Query().Get("status")
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit == 0 { limit = 20 }

    conn, _ := h.registry.Get("pipeline-service")
    client  := pipelinepb.NewPipelineServiceClient(conn)
    resp, err := client.ListJobs(r.Context(), &pipelinepb.ListJobsRequest{
        Engine: engine, Status: status, Limit: int32(limit),
    })
    if err != nil { writeError(w, 500, "list_failed", err.Error()); return }
    writeJSON(w, 200, map[string]any{"jobs": resp.Jobs, "total": resp.Total})
}
```

---

## 3. File Changes

| File | Action |
|---|---|
| `gateway/adapter/handler/pipeline_handler.go` | **[NEW]** |
| `gateway/adapter/handler/router.go` | **[MODIFY]** pipeline routes |
