# Solution: SOL-CONSOLE-001 — Console Dashboard Backend APIs

**CR:** CR-CONSOLE-001
**TDD refs:** `architecture/08-platform-services.md §VNP Search Hub`, `architecture/01-gateway.md §4`
**Version:** v3/console

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** Dashboard: Health/Metrics/Throughput/Heatmap + Prometheus client
---

## 1. Architecture

Dashboard aggregates from 3 sources:
- **Prometheus** (via HTTP API at `:9090`) → metrics
- **healthz** (gateway internal call to `:8083`) → service health
- **Redis** (NATS event log) → throughput/heatmap

Gateway exposes `GET /v1/console/dashboard/*` routes, served by `ConsoleHandler`.

---

## 2. Implementation

```go
// gateway/adapter/handler/console_handler.go [NEW/MODIFY]
type ConsoleHandler struct {
    prometheus port.PrometheusClient
    registry   port.GRPCRegistry
    redis      *redis.Client
}

// GET /v1/console/dashboard/health
func (h *ConsoleHandler) DashboardHealth(w http.ResponseWriter, r *http.Request) {
    // Call internal healthz
    resp, err := http.Get("http://localhost:8083/healthz")
    if err != nil { writeError(w, 500, "health_unavailable", err.Error()); return }
    defer resp.Body.Close()
    var health map[string]any
    json.NewDecoder(resp.Body).Decode(&health)
    writeJSON(w, 200, health)
}

// GET /v1/console/dashboard/metrics
func (h *ConsoleHandler) DashboardMetrics(w http.ResponseWriter, r *http.Request) {
    tenantID := tenant.FromContext(r.Context())
    ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
    defer cancel()

    type metricResult struct{ name string; val float64 }
    ch := make(chan metricResult, 8)

    queries := map[string]string{
        "memories_stored_today": `increase(vnp_memory_store_total{status="success"}[24h])`,
        "recalls_today":          `increase(vnp_memory_recall_duration_ms_count[24h])`,
        "error_rate_pct":         `rate(http_requests_total{status=~"5.."}[5m]) * 100`,
        "p95_store_ms":           `histogram_quantile(0.95, vnp_memory_store_duration_ms_bucket)`,
        "p95_recall_ms":          `histogram_quantile(0.95, vnp_memory_recall_duration_ms_bucket)`,
    }
    for name, query := range queries {
        go func(n, q string) {
            val, _ := h.prometheus.QueryScalar(ctx, q)
            ch <- metricResult{n, val}
        }(name, query)
    }

    // Active sessions from NATS/Redis
    go func() {
        count, _ := h.redis.SCard(ctx, fmt.Sprintf("active_sessions:%s", tenantID)).Result()
        ch <- metricResult{"active_sessions", float64(count)}
    }()

    result := map[string]any{}
    for i := 0; i < len(queries)+1; i++ {
        m := <-ch
        result[m.name] = m.val
    }
    writeJSON(w, 200, result)
}

// GET /v1/console/dashboard/throughput?range=1h
func (h *ConsoleHandler) DashboardThroughput(w http.ResponseWriter, r *http.Request) {
    rangeStr := r.URL.Query().Get("range")
    if rangeStr == "" { rangeStr = "1h" }

    // Prometheus range query
    series, err := h.prometheus.QueryRange(r.Context(),
        `rate(vnp_memory_store_total[1m]) * 60`,
        rangeStr, "1m")
    if err != nil { writeError(w, 500, "metrics_error", err.Error()); return }
    writeJSON(w, 200, map[string]any{"interval_seconds": 60, "series": series})
}

// GET /v1/console/dashboard/heatmap?days=7
func (h *ConsoleHandler) DashboardHeatmap(w http.ResponseWriter, r *http.Request) {
    days := 7
    if d := r.URL.Query().Get("days"); d != "" { fmt.Sscanf(d, "%d", &days) }
    tenantID := tenant.FromContext(r.Context())

    engines := []string{"graphiti", "cognee", "zep", "memobase", "openviking", "supermemory"}
    result := []map[string]any{}

    for i := 0; i < days; i++ {
        date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
        counts := map[string]int64{}
        for _, eng := range engines {
            key := fmt.Sprintf("heatmap:%s:%s:%s", tenantID, eng, date)
            n, _ := h.redis.Get(r.Context(), key).Int64()
            counts[eng] = n
        }
        result = append(result, map[string]any{"date": date, "counts": counts})
    }
    writeJSON(w, 200, map[string]any{"engines": engines, "days": result})
}
```

---

## 3. Prometheus Client Port

```go
// gateway/internal/port/prometheus.go [NEW]
type PrometheusClient interface {
    QueryScalar(ctx context.Context, query string) (float64, error)
    QueryRange(ctx context.Context, query, duration, step string) ([]SeriesPoint, error)
}

// gateway/internal/adapter/prometheus/client.go [NEW]
// Calls Prometheus HTTP API: GET /api/v1/query?query=...
```

---

## 4. Heatmap Redis Update (in memory store handler)

```go
// gateway/adapter/handler/memory_handler.go [MODIFY]
// After successful store dispatch:
go func() {
    date := time.Now().Format("2006-01-02")
    key := fmt.Sprintf("heatmap:%s:%s:%s", tenantID, engine, date)
    h.redis.Incr(context.Background(), key)
    h.redis.Expire(context.Background(), key, 8*24*time.Hour) // keep 8 days
}()
```

---

## 5. File Changes

| File | Action |
|---|---|
| `gateway/adapter/handler/console_handler.go` | **[NEW]** |
| `gateway/internal/port/prometheus.go` | **[NEW]** |
| `gateway/internal/adapter/prometheus/client.go` | **[NEW]** |
| `gateway/adapter/handler/router.go` | **[MODIFY]** dashboard routes |
| `gateway/adapter/handler/memory_handler.go` | **[MODIFY]** heatmap Redis update |
