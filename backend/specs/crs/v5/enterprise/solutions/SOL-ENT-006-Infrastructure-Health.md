# SOL-ENT-006 — Solution: Infrastructure Health & Aggregated Healthz

| Field | Value |
|---|---|
| **Solution ID** | SOL-ENT-006 |
| **CR** | [CR-ENT-006](../../../../docs/crs/v5/enterprise/CR-ENT-006-Infrastructure-Health.md) |
| **TDD ref** | [01-gateway.md](../../../tdd/architecture/01-gateway.md) §Health |
| **Status** | ✅ Implemented |
| **Priority** | 🟠 Medium |

---

## 1. Phân tích

Gateway đã có `GET :8083/healthz`. Cần aggregate health từ tất cả 42 services + infra components.

### 1.1 `gateway/adapter/handler/health_handler.go` [MODIFY]

```go
type HealthHandler struct {
    registry port.ServiceRegistry
    db       *pgxpool.Pool
    neo4j    neo4j.Driver
    redis    *redis.Client
    nats     *nats.Conn
    minio    *minio.Client
}

func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
    defer cancel()

    var wg sync.WaitGroup
    services := map[string]HealthStatus{}
    mu := sync.Mutex{}

    // Check all registered services
    for name, conn := range h.registry.All() {
        wg.Add(1)
        go func(n string, c *grpc.ClientConn) {
            defer wg.Done()
            start := time.Now()
            err := grpcPing(ctx, c)
            mu.Lock()
            if err != nil {
                services[n] = HealthStatus{Status: "unhealthy", Error: err.Error()}
            } else {
                services[n] = HealthStatus{Status: "healthy", LatencyMs: time.Since(start).Milliseconds()}
            }
            mu.Unlock()
        }(name, conn)
    }

    // Check infrastructure
    infra := map[string]HealthStatus{}
    wg.Add(5)
    go func() { defer wg.Done(); infra["postgres"] = checkPostgres(ctx, h.db) }()
    go func() { defer wg.Done(); infra["neo4j"] = checkNeo4j(ctx, h.neo4j) }()
    go func() { defer wg.Done(); infra["redis"] = checkRedis(ctx, h.redis) }()
    go func() { defer wg.Done(); infra["nats"] = checkNATS(h.nats) }()
    go func() { defer wg.Done(); infra["minio"] = checkMinIO(ctx, h.minio) }()
    wg.Wait()

    // Aggregate status
    overall := "healthy"
    for _, s := range services {
        if s.Status == "unhealthy" { overall = "unhealthy"; break }
        if s.Status == "degraded"  { overall = "degraded" }
    }

    resp := HealthResponse{
        Status: overall, Version: version.Version,
        UptimeSeconds: int64(time.Since(startTime).Seconds()),
        Services: services, Infrastructure: infra,
    }

    statusCode := http.StatusOK
    if overall == "unhealthy" { statusCode = http.StatusServiceUnavailable }

    writeJSON(w, statusCode, resp)
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `gateway/adapter/handler/health_handler.go` | MODIFY — aggregate all 42 services |
| `gateway/domain/health.go` | NEW — HealthStatus, HealthResponse types |
| `apps/memory/main.go` | VERIFY — :8083/healthz registered |

---

## 3. Acceptance Criteria

- [ ] `GET :8083/healthz` responds in < 3s (concurrent checks)
- [ ] Returns per-service status + latency_ms
- [ ] Returns per-infra status (postgres, neo4j, redis, nats, minio)
- [ ] HTTP 503 if any service unhealthy
- [ ] `GET :8083/healthz/live` lightweight liveness (no downstream checks)
- [ ] `GET :8083/healthz/ready` readiness (all services healthy)

---

**Ghi chú audit:** gateway ObservabilityServer: /healthz/deep (16 services), /healthz, /readyz, /metrics (Prometheus) endpoints
