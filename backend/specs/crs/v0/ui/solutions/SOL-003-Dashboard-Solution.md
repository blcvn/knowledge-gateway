# SOL-003 — Solution: Dashboard API (CR-002)

| Field | Value |
|---|---|
| **Solution ID** | SOL-003 |
| **CR** | [CR-002 — Dashboard](../CR-002-DASHBOARD.md) |
| **Architecture ref** | §2.2 Services Inventory · §4.2 Console Routes FEAT-006 · §6.3 Cross-Engine Recall · §8 Infra Dependencies |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Implemented** | 2026-06-17 |

---

## 1. Phân tích kiến trúc

Dashboard cần tổng hợp dữ liệu từ **nhiều nguồn khác nhau** trong một monolith:

| Metric nhóm | Nguồn kiến trúc |
|---|---|
| Engine health (7 engines) | `InProcessRegistry` → gRPC health check đến 35 services (§2.1) |
| KPI metrics (latency, error rate) | `Prometheus` (§8) qua `obs-service` |
| Throughput per engine | NATS JetStream metrics + Prometheus counters |
| Graph stats (nodes/edges) | `graphiti-store` → Neo4j (§8) |
| Active sessions / profiles | PostgreSQL (§8) |
| Memory heatmap (activity) | `vnp-event` → UserEvent table (§5.2 Event Domain) |

Console route `FEAT-006` ↔ paths `/v1/console/dashboard/*` (§4.2) đã được đăng ký trong router.

---

## 2. Giải pháp Backend

### 2.1 Handler (`gateway/internal/adapter/handler/console_dashboard_handler.go`)

```go
type ConsoleDashboardHandler struct {
    registry    InProcessRegistry   // gọi health check đến 35 services qua bufconn
    obsClient   ObsServiceClient    // Prometheus wrapper từ obs-service
    natsClient  NATSClient          // JetStream metrics
    db          *sql.DB             // PostgreSQL cho session/profile count
    graphClient GraphitiStoreClient // Neo4j node/edge count qua graphiti-store gRPC
    eventClient VNPEventClient      // vnp-event service cho heatmap
}

// GET /v1/console/dashboard/metrics
func (h *ConsoleDashboardHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    tenantID := authctx.TenantID(ctx)

    // Fan-out parallel queries
    g, gctx := errgroup.WithContext(ctx)

    var (
        latencyP50, latencyP95 float64
        errorRate, savings      float64
        nodeCount, edgeCount    int64
        growth24h               int64
        sessionCount            int64
        profileCount            int64
        versionCount            int64
    )

    g.Go(func() error {
        m, err := h.obsClient.GetLatencyPercentiles(gctx)
        latencyP50, latencyP95 = m.P50, m.P95
        return err
    })
    g.Go(func() error {
        nodeCount, edgeCount, growth24h, _ = h.graphClient.GetStats(gctx, tenantID)
        return nil
    })
    g.Go(func() error {
        sessionCount, _ = h.db.CountActiveSessions(gctx, tenantID)
        profileCount, _ = h.db.CountActiveProfiles(gctx, tenantID)
        return nil
    })
    // ... (tolerate individual failures)
    _ = g.Wait()

    httputil.JSON(w, 200, KPIData{
        RecallLatencyP50Ms: latencyP50,
        RecallLatencyP95Ms: latencyP95,
        GraphNodesTotal:    nodeCount,
        GraphEdgesTotal:    edgeCount,
        GraphGrowth24h:     growth24h,
        ActiveSessions:     sessionCount,
        ActiveProfiles:     profileCount,
    })
}
```

### 2.2 Engine Health — via InProcessRegistry

```go
// GET /v1/console/dashboard/health
func (h *ConsoleDashboardHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Gọi gRPC Health protocol đến mỗi service thông qua bufconn
    engines := []string{
        "memobase", "supermemory", "graphiti",
        "cognee", "zep", "openviking", "kgs",
    }

    results := make([]EngineHealthResponse, 0, len(engines))
    for _, name := range engines {
        conn, _ := h.registry.GetConn(name) // bufconn — zero latency
        client := grpc_health_v1.NewHealthClient(conn)
        resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})

        latencyP50, latencyP95 := h.obsClient.GetEngineLatency(ctx, name)
        queueDepth := h.natsClient.GetConsumerPending(ctx, name)
        uptime := h.registry.GetUptime(name)

        status := "Healthy"
        if err != nil || resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
            status = "Critical"
        } else if latencyP95 >= 500 {
            status = "Warning"
        } else if latencyP95 >= 200 {
            status = "Warning"
        }

        results = append(results, EngineHealthResponse{
            Name:          name,
            Role:          engineRoleMap[name],
            Status:        status,
            LatencyP50Ms:  latencyP50,
            LatencyP95Ms:  latencyP95,
            QueueDepth:    queueDepth,
            UptimeSeconds: int(uptime.Seconds()),
            LastCheck:     time.Now().UTC().Format(time.RFC3339),
        })
    }

    httputil.JSON(w, 200, results)
}
```

**JSON casing quan trọng** (phải khớp TypeScript `EngineHealth` type):
```go
type EngineHealthResponse struct {
    Name          string  `json:"name"`
    Role          string  `json:"role"`
    Status        string  `json:"status"`
    LatencyP50Ms  float64 `json:"latencyP50Ms"`
    LatencyP95Ms  float64 `json:"latencyP95Ms"`
    QueueDepth    int64   `json:"queueDepth"`
    UptimeSeconds int     `json:"uptimeSeconds"`
    LastCheck     string  `json:"lastCheck"`
}
```

### 2.3 Throughput — Prometheus Rate Queries

```go
// GET /v1/console/dashboard/throughput?window=1h
func (h *ConsoleDashboardHandler) GetThroughput(w http.ResponseWriter, r *http.Request) {
    window := r.URL.Query().Get("window")
    if window == "" { window = "1h" }

    // Query Prometheus
    // vnp_memory_operations_total{engine="memobase",operation="ingest"}
    data, _ := h.obsClient.QueryThroughputByEngine(r.Context(), window)

    httputil.JSON(w, 200, ThroughputData{
        Window:  window,
        Engines: data, // map[EngineType]MemoryFlowMetrics
    })
}
```

Prometheus metrics cần được expose bởi mỗi service:
```
vnp_memory_ingest_total{engine="memobase"} counter
vnp_memory_recall_total{engine="graphiti"} counter
vnp_memory_embed_total{engine="cognee"}    counter
```

### 2.4 Heatmap — vnp-event UserEvent table

```go
// GET /v1/console/dashboard/heatmap
// Dùng UserEvent table từ vnp-event domain (§5.2)
func (h *ConsoleDashboardHandler) GetHeatmap(w http.ResponseWriter, r *http.Request) {
    // SQL: COUNT events GROUP BY EXTRACT(DOW FROM created_at), EXTRACT(HOUR FROM created_at)
    points, _ := h.eventClient.GetActivityHeatmap(r.Context(), authctx.TenantID(r.Context()))
    httputil.JSON(w, 200, HeatmapResponse{Points: points})
}
```

---

## 3. Giải pháp Frontend

### 3.1 Cập nhật `useDashboard.ts`

```typescript
// Xóa mock imports, thêm refetchInterval
import { useQuery } from '@tanstack/react-query';
import { dashboardService } from '../services/dashboard.service';

export function useEngineHealth() {
    return useQuery({
        queryKey: ['dashboard', 'health'],
        queryFn: () => dashboardService.getHealth(),
        staleTime: 15_000,
        refetchInterval: 30_000,   // Poll 30s
        refetchIntervalInBackground: false,
    });
}

export function useMetrics() {
    return useQuery({
        queryKey: ['dashboard', 'metrics'],
        queryFn: () => dashboardService.getMetrics(),
        staleTime: 30_000,
        refetchInterval: 60_000,
    });
}

export function useThroughput(window = '1h') {
    return useQuery({
        queryKey: ['dashboard', 'throughput', window],
        queryFn: () => dashboardService.getThroughput(window),
        staleTime: 30_000,
        refetchInterval: 30_000,
    });
}

export function useDashboardHeatmap() {
    return useQuery({
        queryKey: ['dashboard', 'heatmap'],
        queryFn: () => dashboardService.getHeatmap(),
        staleTime: 300_000,  // 5 phút
    });
}
```

---

## 4. Caching Strategy (Backend)

Để tránh overload Prometheus và Neo4j mỗi 30s:

```go
// Dùng Redis (§8 Infrastructure) để cache response
func (h *ConsoleDashboardHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
    cacheKey := fmt.Sprintf("console:dashboard:health:%s", tenantID)

    // Try Redis cache first (TTL 20s)
    if cached, err := h.redis.Get(ctx, cacheKey); err == nil {
        w.Header().Set("X-Cache", "HIT")
        w.Write(cached)
        return
    }

    // Compute health...
    result := h.computeHealth(ctx)

    // Cache 20s
    h.redis.Set(ctx, cacheKey, result, 20*time.Second)
    httputil.JSON(w, 200, result)
}
```
