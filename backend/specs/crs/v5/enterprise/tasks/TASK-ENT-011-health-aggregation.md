# TASK-ENT-011 — Aggregated Health Check (42 Services)

| Field | Value |
|---|---|
| **Task ID** | TASK-ENT-011 |
| **Wave** | 3 |
| **Solution** | [SOL-ENT-006](../solutions/SOL-ENT-006-Infrastructure-Health.md) §1.1 |
| **Component** | `gateway/adapter/handler/health_handler.go` |
| **Priority** | 🟠 Medium |
| **Depends On** | TASK-ENT-009 |
| **Estimated** | 4h |

---

## Mục tiêu

`GET /healthz` aggregates health from all 42 services + 5 infra components in < 3s.

---

## Công việc cụ thể

### `gateway/adapter/handler/health_handler.go` [MODIFY]

```go
type HealthStatus struct {
    Status    string `json:"status"`     // healthy|degraded|unhealthy
    LatencyMs int64  `json:"latency_ms,omitempty"`
    Error     string `json:"error,omitempty"`
}

type HealthResponse struct {
    Status         string                  `json:"status"`
    Version        string                  `json:"version"`
    UptimeSeconds  int64                   `json:"uptime_seconds"`
    Services       map[string]HealthStatus `json:"services"`
    Infrastructure map[string]HealthStatus `json:"infrastructure"`
}

var startTime = time.Now()

func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
    defer cancel()

    var wg sync.WaitGroup
    services := map[string]HealthStatus{}
    infra    := map[string]HealthStatus{}
    mu := sync.Mutex{}

    // Check all registered gRPC services
    for name, conn := range h.registry.All() {
        wg.Add(1)
        go func(n string, c *grpc.ClientConn) {
            defer wg.Done()
            start := time.Now()
            status := HealthStatus{}
            if err := grpcHealthCheck(ctx, c); err != nil {
                status = HealthStatus{Status: "unhealthy", Error: err.Error()}
                telemetry.EngineHealth.WithLabelValues(n).Set(0)
            } else {
                status = HealthStatus{Status: "healthy", LatencyMs: time.Since(start).Milliseconds()}
                telemetry.EngineHealth.WithLabelValues(n).Set(1)
            }
            mu.Lock(); services[n] = status; mu.Unlock()
        }(name, conn)
    }

    // Infrastructure checks
    infraChecks := map[string]func() HealthStatus{
        "postgres": func() HealthStatus { return checkPostgres(ctx, h.db) },
        "neo4j":    func() HealthStatus { return checkNeo4j(ctx, h.neo4j) },
        "redis":    func() HealthStatus { return checkRedis(ctx, h.redis) },
        "nats":     func() HealthStatus { return checkNATS(h.nats) },
        "minio":    func() HealthStatus { return checkMinIO(ctx, h.minio) },
    }
    for name, check := range infraChecks {
        wg.Add(1)
        go func(n string, c func() HealthStatus) {
            defer wg.Done()
            s := c()
            mu.Lock(); infra[n] = s; mu.Unlock()
        }(name, check)
    }
    wg.Wait()

    // Aggregate status
    overall := "healthy"
    for _, s := range services {
        if s.Status == "unhealthy" { overall = "unhealthy"; break }
        if s.Status == "degraded" { overall = "degraded" }
    }

    resp := HealthResponse{
        Status: overall, Version: version.Version,
        UptimeSeconds: int64(time.Since(startTime).Seconds()),
        Services: services, Infrastructure: infra,
    }

    code := http.StatusOK
    if overall == "unhealthy" { code = http.StatusServiceUnavailable }
    writeJSON(w, code, resp)
}

// Lightweight liveness — no downstream checks
func (h *HealthHandler) LivenessCheck(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, 200, map[string]string{"status": "alive"})
}

// Readiness — all critical services healthy
func (h *HealthHandler) ReadinessCheck(w http.ResponseWriter, r *http.Request) { ... }
```

### Router registration

```go
r.Get("/healthz",       healthHandler.Healthz)
r.Get("/healthz/live",  healthHandler.LivenessCheck)
r.Get("/healthz/ready", healthHandler.ReadinessCheck)
```

---

## Acceptance Criteria

- [ ] /healthz responds in < 3s (all checks concurrent)
- [ ] Per-service status with latency_ms
- [ ] Per-infra status (postgres, neo4j, redis, nats, minio)
- [ ] HTTP 503 if any service unhealthy
- [ ] /healthz/live responds immediately (< 5ms)
- [ ] /healthz/ready returns unhealthy if critical services down
- [ ] EngineHealth gauge updated per check

## Files

```
gateway/adapter/handler/health_handler.go  [MODIFY — aggregate all 42]
gateway/adapter/handler/router.go          [MODIFY — /healthz/live, /healthz/ready]
```
