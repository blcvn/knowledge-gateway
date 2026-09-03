# TASK-BE-003 — Console Dashboard Handler: metrics / health / throughput / heatmap

| Field | Value |
|---|---|
| **Task ID** | TASK-BE-003 |
| **Layer** | Backend — Go |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-003](../solutions/SOL-003-Dashboard-Solution.md) |
| **Priority** | 🔴 P0 |
| **Depends On** | — |
| **Estimated** | 4h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `gateway/internal/adapter/handler/console_dashboard_handler.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

```go
package handler

import (
    "context"
    "encoding/json"
    "net/http"
    "sync"
    "time"

    "golang.org/x/sync/errgroup"
)

type ConsoleDashboardHandler struct {
    registry   InProcessRegistry   // 35 services health check
    db         *sql.DB             // PostgreSQL
    neo4j      neo4j.Driver
    redis      *redis.Client       // Cache 20s
    promClient PrometheusClient    // HTTP API client
    nats       *nats.Conn
}

// GET /v1/console/dashboard/metrics
func (h *ConsoleDashboardHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
    tenantID := authctx.TenantID(r.Context())

    // Check Redis cache
    cacheKey := "dashboard:metrics:" + tenantID
    if cached, err := h.redis.Get(r.Context(), cacheKey).Bytes(); err == nil {
        w.Header().Set("Content-Type", "application/json")
        w.Write(cached)
        return
    }

    g, ctx := errgroup.WithContext(r.Context())
    var (
        activeSessions    int
        activeProfiles    int
        memoryVersions    int64
        graphNodes        int64
        graphEdges        int64
        graphGrowth       int64
        recallP50         float64
        recallP95         float64
        errorRatePct      float64
        contextSavingsPct float64
        activeAgents      int
    )

    // Fan-out parallel queries
    g.Go(func() error {
        return h.db.QueryRowContext(ctx,
            `SELECT COUNT(*) FROM sessions WHERE tenant_id = $1 AND status = 'active'`,
            tenantID).Scan(&activeSessions)
    })
    g.Go(func() error {
        return h.db.QueryRowContext(ctx,
            `SELECT COUNT(DISTINCT user_id) FROM memobase_profiles WHERE tenant_id = $1`,
            tenantID).Scan(&activeProfiles)
    })
    g.Go(func() error {
        return h.db.QueryRowContext(ctx,
            `SELECT COUNT(*) FROM sm_memories WHERE tenant_id = $1`,
            tenantID).Scan(&memoryVersions)
    })
    g.Go(func() error {
        // Neo4j counts
        result, err := h.neo4j.ExecuteQueryBookmarked(ctx,
            `MATCH (n) WHERE n.tenant_id = $tenant RETURN count(n) as nodes`,
            map[string]any{"tenant": tenantID}, neo4j.EagerResultTransformer)
        if err != nil { return err }
        graphNodes, _ = result.Records[0].Values[0].(int64)
        return nil
    })
    g.Go(func() error {
        recallP50, _ = h.promClient.QueryScalar(ctx,
            `histogram_quantile(0.50, sum(rate(vnp_recall_latency_ms_bucket{tenant="`+tenantID+`"}[5m])) by (le))`)
        recallP95, _ = h.promClient.QueryScalar(ctx,
            `histogram_quantile(0.95, sum(rate(vnp_recall_latency_ms_bucket{tenant="`+tenantID+`"}[5m])) by (le))`)
        return nil
    })
    g.Go(func() error {
        errors, _ := h.promClient.QueryScalar(ctx,
            `rate(vnp_errors_total{tenant="`+tenantID+`"}[1h])`)
        requests, _ := h.promClient.QueryScalar(ctx,
            `rate(vnp_requests_total{tenant="`+tenantID+`"}[1h])`)
        if requests > 0 {
            errorRatePct = errors / requests * 100
        }
        return nil
    })

    if err := g.Wait(); err != nil {
        // Partial result — tolerate individual failures
    }

    resp := map[string]any{
        "activeAgents":        activeAgents,
        "recallLatencyP50Ms":  recallP50,
        "recallLatencyP95Ms":  recallP95,
        "contextSavingsPct":   contextSavingsPct,
        "graphNodesTotal":     graphNodes,
        "graphEdgesTotal":     graphEdges,
        "graphGrowth24h":      graphGrowth,
        "errorRatePct":        errorRatePct,
        "activeSessions":      activeSessions,
        "activeProfiles":      activeProfiles,
        "memoryVersions":      memoryVersions,
    }

    // Cache 20s
    if b, err := json.Marshal(resp); err == nil {
        h.redis.Set(r.Context(), cacheKey, b, 20*time.Second)
    }
    httputil.JSON(w, 200, resp)
}

// GET /v1/console/dashboard/health
func (h *ConsoleDashboardHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
    engines := []string{"memobase", "graphiti", "zep", "cognee", "supermemory", "openviking", "openviking-cognitive"}

    results := make([]map[string]any, 0, len(engines))
    var mu sync.Mutex
    var wg sync.WaitGroup

    for _, eng := range engines {
        wg.Add(1)
        go func(name string) {
            defer wg.Done()
            svc := h.registry.Get(name)
            status := "Healthy"
            latP50, _ := h.promClient.QueryScalar(r.Context(),
                `histogram_quantile(0.50, rate(vnp_latency_ms_bucket{service="`+name+`"}[5m]))`)
            latP95, _ := h.promClient.QueryScalar(r.Context(),
                `histogram_quantile(0.95, rate(vnp_latency_ms_bucket{service="`+name+`"}[5m]))`)

            // Status mapping
            switch {
            case svc == nil:
                status = "Critical"
            case latP95 >= 500:
                status = "Critical"
            case latP95 >= 200:
                status = "Warning"
            }

            mu.Lock()
            results = append(results, map[string]any{
                "name":          name,
                "status":        status,
                "latencyP50Ms":  latP50,
                "latencyP95Ms":  latP95,
                "lastCheck":     time.Now().UTC().Format(time.RFC3339),
            })
            mu.Unlock()
        }(eng)
    }
    wg.Wait()
    httputil.JSON(w, 200, results)
}

// GET /v1/console/dashboard/throughput?window=1h
func (h *ConsoleDashboardHandler) GetThroughput(w http.ResponseWriter, r *http.Request) {
    window := r.URL.Query().Get("window")
    if window == "" { window = "1h" }

    engines := []string{"memobase", "graphiti", "zep", "cognee", "supermemory"}
    data := map[string]any{}
    for _, eng := range engines {
        ingest, _ := h.promClient.QueryScalar(r.Context(),
            `rate(vnp_ingest_total{engine="`+eng+`"}[`+window+`])`)
        recall, _ := h.promClient.QueryScalar(r.Context(),
            `rate(vnp_recall_total{engine="`+eng+`"}[`+window+`])`)
        data[eng] = map[string]any{"ingestPerSec": ingest, "recallPerSec": recall}
    }
    httputil.JSON(w, 200, map[string]any{"window": window, "engines": data})
}

// GET /v1/console/dashboard/heatmap
func (h *ConsoleDashboardHandler) GetHeatmap(w http.ResponseWriter, r *http.Request) {
    tenantID := authctx.TenantID(r.Context())
    rows, err := h.db.QueryContext(r.Context(),
        `SELECT EXTRACT(HOUR FROM created_at)::int AS hour,
                EXTRACT(DOW FROM created_at)::int AS dow,
                COUNT(*) AS density
         FROM audit_logs
         WHERE tenant_id = $1 AND created_at > NOW() - INTERVAL '7 days'
         GROUP BY hour, dow`,
        tenantID,
    )
    if err != nil {
        httputil.Error(w, "Query failed", "INTERNAL_ERROR", 500)
        return
    }
    defer rows.Close()

    type Point struct {
        X       int `json:"x"`
        Y       int `json:"y"`
        Density int `json:"density"`
    }
    var points []Point
    maxDensity := 0
    for rows.Next() {
        var p Point
        rows.Scan(&p.X, &p.Y, &p.Density)
        if p.Density > maxDensity { maxDensity = p.Density }
        points = append(points, p)
    }
    httputil.JSON(w, 200, map[string]any{
        "points": points, "xLabel": "Hour of day",
        "yLabel": "Day of week", "maxDensity": maxDensity,
    })
}
```

### Routes registration

```go
mux.HandleFunc("GET /v1/console/dashboard/metrics",    authMiddleware(dash.GetMetrics))
mux.HandleFunc("GET /v1/console/dashboard/health",     authMiddleware(dash.GetHealth))
mux.HandleFunc("GET /v1/console/dashboard/throughput", authMiddleware(dash.GetThroughput))
mux.HandleFunc("GET /v1/console/dashboard/heatmap",    authMiddleware(dash.GetHeatmap))
```

---

## Verification

```bash
curl http://localhost:8080/v1/console/dashboard/metrics \
  -H "Authorization: Bearer <token>" -H "x-tenant-id: <tid>"
# Expected: JSON với 11 KPI fields

curl "http://localhost:8080/v1/console/dashboard/throughput?window=5m" \
  -H "Authorization: Bearer <token>" -H "x-tenant-id: <tid>"
# Expected: JSON với engines object
```
