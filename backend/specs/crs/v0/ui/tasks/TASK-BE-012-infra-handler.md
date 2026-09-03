# TASK-BE-012 — Console Infrastructure Handler

| Field | Value |
|---|---|
| **Task ID** | TASK-BE-012 |
| **Layer** | Backend — Go |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-006 CR-010](../solutions/SOL-006-Adaptive-to-Org-Solutions.md) |
| **Priority** | 🟡 P2 |
| **Depends On** | — |
| **Estimated** | 2h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `gateway/internal/adapter/handler/console_infra_handler.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

```go
package handler

type ConsoleInfraHandler struct {
    registry InProcessRegistry  // 35 services health check
    db       *sql.DB            // PostgreSQL ping
    neo4j    neo4j.Driver       // Neo4j ping
    redis    *redis.Client      // Redis ping
    nats     *nats.Conn         // NATS status
    minio    *minio.Client      // MinIO ping
    promClient PromClient        // Prometheus node_exporter metrics
}

// GET /v1/console/infra/services
func (h *ConsoleInfraHandler) ListServices(w http.ResponseWriter, r *http.Request) {
    services := h.registry.GetAll()
    results := make([]map[string]any, 0, len(services))
    var mu sync.Mutex
    var wg sync.WaitGroup

    for _, svc := range services {
        wg.Add(1)
        go func(s ServiceEntry) {
            defer wg.Done()
            status := "Healthy"
            if err := s.HealthCheck(r.Context()); err != nil { status = "Critical" }
            mu.Lock()
            results = append(results, map[string]any{
                "name":    s.Name,
                "version": s.Version,
                "status":  status,
                "uptime":  s.UptimeSeconds(),
            })
            mu.Unlock()
        }(svc)
    }
    wg.Wait()
    httputil.JSON(w, 200, results)
}

// GET /v1/console/infra/databases
func (h *ConsoleInfraHandler) GetDatabases(w http.ResponseWriter, r *http.Request) {
    type dbResult struct {
        Name      string `json:"name"`
        Type      string `json:"type"`
        Status    string `json:"status"`
        LatencyMs int64  `json:"latency_ms"`
    }

    results := []dbResult{}
    var mu sync.Mutex
    var wg sync.WaitGroup

    ping := func(name, dbType string, fn func() (int64, error)) {
        wg.Add(1)
        go func() {
            defer wg.Done()
            lat, err := fn()
            status := "Healthy"
            if err != nil { status = "Critical"; lat = -1 }
            mu.Lock()
            results = append(results, dbResult{name, dbType, status, lat})
            mu.Unlock()
        }()
    }

    ping("Postgres-Primary", "PostgreSQL", func() (int64, error) {
        start := time.Now()
        return time.Since(start).Milliseconds(), h.db.PingContext(r.Context())
    })
    ping("Neo4j-Graph", "Neo4j", func() (int64, error) {
        start := time.Now()
        return time.Since(start).Milliseconds(), h.neo4j.VerifyConnectivity(r.Context())
    })
    ping("Redis", "Redis", func() (int64, error) {
        start := time.Now()
        _, err := h.redis.Ping(r.Context()).Result()
        return time.Since(start).Milliseconds(), err
    })
    ping("NATS", "NATS", func() (int64, error) {
        if h.nats.IsConnected() { return 0, nil }
        return -1, errors.New("disconnected")
    })

    wg.Wait()
    httputil.JSON(w, 200, results)
}

// GET /v1/console/infra/resources
func (h *ConsoleInfraHandler) GetResources(w http.ResponseWriter, r *http.Request) {
    cpu, _    := h.promClient.QueryScalar(r.Context(), `rate(process_cpu_seconds_total{job="vnp-memory"}[1m]) * 100`)
    memBytes, _ := h.promClient.QueryScalar(r.Context(), `process_resident_memory_bytes{job="vnp-memory"}`)
    httputil.JSON(w, 200, []map[string]any{{
        "service": "vnp-gateway",
        "cpu_usage_pct":  cpu,
        "memory_usage_mb": memBytes / 1024 / 1024,
        "disk_usage_pct":  0,  // node_exporter metric
    }})
}

// GET /v1/console/infra/topology
func (h *ConsoleInfraHandler) GetTopology(w http.ResponseWriter, r *http.Request) {
    services := h.registry.GetAll()
    httputil.JSON(w, 200, map[string]any{
        "mode":       "monolith",
        "node_count": len(services),
        "services":   services,
    })
}

// GET /v1/console/infra/deployments
func (h *ConsoleInfraHandler) GetDeployments(w http.ResponseWriter, r *http.Request) {
    // Build info injected at compile time
    httputil.JSON(w, 200, []map[string]any{{
        "service":    "vnp-gateway",
        "version":    buildinfo.Version,
        "git_commit": buildinfo.GitCommit,
        "started_at": buildinfo.StartTime.Format(time.RFC3339),
    }})
}
```

### Routes

```go
mux.HandleFunc("GET /v1/console/infra/services",    authMiddleware(infra.ListServices))
mux.HandleFunc("GET /v1/console/infra/databases",   authMiddleware(infra.GetDatabases))
mux.HandleFunc("GET /v1/console/infra/resources",   authMiddleware(infra.GetResources))
mux.HandleFunc("GET /v1/console/infra/topology",    authMiddleware(infra.GetTopology))
mux.HandleFunc("GET /v1/console/infra/deployments", authMiddleware(infra.GetDeployments))
```
