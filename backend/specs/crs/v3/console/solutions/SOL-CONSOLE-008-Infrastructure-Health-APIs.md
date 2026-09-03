# Solution: SOL-CONSOLE-008 — Infrastructure Health Console APIs

**CR:** CR-CONSOLE-008
**TDD refs:** `architecture/10-data-models-deployment.md`, `architecture/09-shared-packages.md §telemetry`
**Version:** v3/console

**Trạng thái:** 🔄 Partial  
**Ghi chú audit:** InfraHandler: Services/Config/Deploy; health deep-check partial
---

## 1. Architecture

Infrastructure health from:
- **PostgreSQL PING** (`SELECT 1`)
- **Neo4j PING** (bolt protocol health check)
- **Redis PING** command
- **NATS PING** (nats.Status())
- **MinIO** (S3-compatible health endpoint)
- **Alert engine**: evaluate rules → persist to `infra_alerts` table

---

## 2. Implementation

```go
// gateway/adapter/handler/infra_handler.go [NEW]
type InfraHandler struct {
    db    *pgxpool.Pool
    redis *redis.Client
    nats  *nats.Conn
    neo4j port.Neo4jClient
    minio port.MinIOClient
    alertRepo port.InfraAlertRepository
}

// GET /v1/console/infrastructure/health
func (h *InfraHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
    defer cancel()

    type compResult struct {
        Name   string
        Status string
        Extra  map[string]any
        Error  string
    }

    ch := make(chan compResult, 5)

    // Parallel health checks
    go func() {
        start := time.Now()
        _, err := h.db.Exec(ctx, "SELECT 1")
        if err != nil { ch <- compResult{"postgres", "unhealthy", nil, err.Error()}; return }
        // Connection pool stats
        stats := h.db.Stat()
        ch <- compResult{"postgres", "healthy", map[string]any{
            "latency_ms":  time.Since(start).Milliseconds(),
            "connections": map[string]int{"used": int(stats.AcquiredConns()), "max": int(stats.MaxConns())},
        }, ""}
    }()

    go func() {
        start := time.Now()
        err := h.redis.Ping(ctx).Err()
        if err != nil { ch <- compResult{"redis", "unhealthy", nil, err.Error()}; return }
        info, _ := h.redis.Info(ctx, "memory").Result()
        usedMB, maxMB := parseRedisMemory(info)
        ch <- compResult{"redis", "healthy", map[string]any{
            "latency_ms": time.Since(start).Milliseconds(),
            "memory_used_mb": usedMB, "memory_max_mb": maxMB,
        }, ""}
    }()

    go func() {
        start := time.Now()
        if h.nats.Status() != nats.CONNECTED {
            ch <- compResult{"nats", "unhealthy", nil, "disconnected"}; return
        }
        js, _ := h.nats.JetStream()
        streams := 0
        for range js.Streams() { streams++ }
        ch <- compResult{"nats", "healthy", map[string]any{
            "latency_ms": time.Since(start).Milliseconds(), "subjects_count": streams,
        }, ""}
    }()

    go func() {
        start := time.Now()
        if err := h.neo4j.Ping(ctx); err != nil {
            ch <- compResult{"neo4j", "unhealthy", nil, err.Error()}; return
        }
        ch <- compResult{"neo4j", "healthy", map[string]any{
            "latency_ms": time.Since(start).Milliseconds(),
        }, ""}
    }()

    go func() {
        start := time.Now()
        if err := h.minio.HealthCheck(ctx); err != nil {
            ch <- compResult{"minio", "unhealthy", nil, err.Error()}; return
        }
        ch <- compResult{"minio", "healthy", map[string]any{
            "latency_ms": time.Since(start).Milliseconds(),
        }, ""}
    }()

    // Collect results
    components := map[string]any{}
    overall := "healthy"
    for i := 0; i < 5; i++ {
        res := <-ch
        components[res.Name] = map[string]any{
            "status": res.Status, "extra": res.Extra, "error": res.Error,
        }
        if res.Status == "unhealthy" { overall = "unhealthy" }
        if res.Status == "degraded" && overall != "unhealthy" { overall = "degraded" }
    }
    writeJSON(w, 200, map[string]any{"overall": overall, "components": components})
}

// GET /v1/console/infrastructure/alerts
func (h *InfraHandler) GetAlerts(w http.ResponseWriter, r *http.Request) {
    alerts, err := h.alertRepo.GetActive(r.Context())
    if err != nil { writeError(w, 500, "alerts_failed", err.Error()); return }
    writeJSON(w, 200, map[string]any{"active_alerts": alerts})
}

// GET /v1/console/infrastructure/resources
func (h *InfraHandler) GetResources(w http.ResponseWriter, r *http.Request) {
    // Returns Go runtime stats from all services via pprof/metrics
    // Aggregated from Prometheus: go_goroutines, go_memstats_alloc_bytes per service
    conn, _ := h.registry.Get("vnp-observability")
    client  := obspb.NewVnpObservabilityServiceClient(conn)
    resp, err := client.GetResourceStats(r.Context(), &obspb.ResourceStatsRequest{})
    if err != nil { writeError(w, 500, "resources_failed", err.Error()); return }
    writeJSON(w, 200, map[string]any{"services": resp.Services})
}
```

---

## 3. Alert Rule Engine (background goroutine)

```go
// gateway/internal/infra/health_monitor.go [NEW]
type HealthMonitor struct {
    db         *pgxpool.Pool
    infraCheck *InfraChecker
    ticker     *time.Ticker
}

// AlertRule definitions
var alertRules = []struct {
    Component string
    Condition func(status string, extra map[string]any) (bool, string)
    Severity  string
}{
    {"neo4j",   func(s string, e map[string]any) (bool, string) { return s == "unhealthy", "Neo4j connection lost" }, "critical"},
    {"postgres", func(s string, e map[string]any) (bool, string) {
        if m, ok := e["connections"].(map[string]any); ok {
            used := m["used"].(int); max := m["max"].(int)
            if float64(used)/float64(max) > 0.9 {
                return true, fmt.Sprintf("Connection pool %.0f%% full", float64(used)/float64(max)*100)
            }
        }
        return false, ""
    }, "warning"},
}

func (m *HealthMonitor) Start(ctx context.Context) {
    for {
        select {
        case <-m.ticker.C:
            m.evaluate(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (m *HealthMonitor) evaluate(ctx context.Context) {
    // Check health, evaluate rules, upsert active_alerts
    health := m.infraCheck.CheckAll(ctx)
    for _, rule := range alertRules {
        comp := health[rule.Component]
        triggered, msg := rule.Condition(comp.Status, comp.Extra)
        if triggered {
            m.db.Exec(ctx, `
                INSERT INTO infra_alerts (component, severity, message)
                VALUES ($1, $2, $3)
                ON CONFLICT (component, severity) DO UPDATE SET message=$3, updated_at=NOW()`,
                rule.Component, rule.Severity, msg)
        }
    }
}
```

---

## 4. DB Migration

```sql
-- deployment/dev/migrations/0051_infra_alerts.sql
CREATE TABLE IF NOT EXISTS infra_alerts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    component  TEXT NOT NULL,
    severity   TEXT NOT NULL, -- warning|critical
    message    TEXT,
    since      TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (component, severity)
);
```

---

## 5. File Changes

| File | Action |
|---|---|
| `gateway/adapter/handler/infra_handler.go` | **[NEW]** |
| `gateway/internal/infra/health_monitor.go` | **[NEW]** alert rule engine |
| `gateway/adapter/handler/router.go` | **[MODIFY]** infra routes |
| `deployment/dev/migrations/0051_infra_alerts.sql` | **[NEW]** |
