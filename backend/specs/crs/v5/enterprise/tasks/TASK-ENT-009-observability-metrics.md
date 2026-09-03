# TASK-ENT-009 — Prometheus Metrics Setup

| Field | Value |
|---|---|
| **Task ID** | TASK-ENT-009 |
| **Wave** | 3 |
| **Solution** | [SOL-ENT-005](../solutions/SOL-ENT-005-Unified-Observability.md) §1.1 |
| **Component** | `shared/pkg/telemetry/metrics.go` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 4h |

---

## Mục tiêu

Register all vnp_memory_* Prometheus metrics in shared/pkg/telemetry.

---

## Công việc cụ thể

### `shared/pkg/telemetry/metrics.go` [MODIFY]

```go
package telemetry

import "github.com/prometheus/client_golang/prometheus"

var (
    // ─── Memory Operations ────────────────────────────────────────────────
    MemoryStoreDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "vnp_memory_store_duration_ms",
            Help:    "Duration of memory store operation in milliseconds",
            Buckets: []float64{10, 25, 50, 100, 250, 500, 1000, 2500},
        },
        []string{"engine", "type", "tenant"},
    )
    MemoryStoreTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "vnp_memory_store_total",
            Help: "Total memory store operations",
        },
        []string{"engine", "type", "status"}, // status: success|error
    )
    MemoryRecallDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "vnp_memory_recall_duration_ms",
            Help:    "Duration of memory recall in milliseconds",
            Buckets: []float64{50, 100, 250, 500, 1000, 2500, 5000},
        },
        []string{"engine", "tenant"},
    )

    // ─── Observe / AgentMemory ────────────────────────────────────────────
    ObserveHooksTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "vnp_observe_hooks_total",
            Help: "Total hook events observed",
        },
        []string{"hook_type", "agent"},
    )
    SessionDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "vnp_observe_session_duration_seconds",
            Help:    "Agent session duration in seconds",
            Buckets: prometheus.ExponentialBuckets(10, 2, 10),
        },
        []string{"agent"},
    )

    // ─── Engine Health ─────────────────────────────────────────────────────
    EngineHealth = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "vnp_engine_health",
            Help: "Engine health status: 1=healthy, 0=unhealthy",
        },
        []string{"engine"},
    )
    DBConnectionPoolUsed = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "vnp_db_connection_pool_used",
            Help: "Number of active DB connections",
        },
        []string{"db"},
    )
)

// Init registers all metrics — call once at startup
func Init() {
    prometheus.MustRegister(
        MemoryStoreDuration, MemoryStoreTotal, MemoryRecallDuration,
        ObserveHooksTotal, SessionDuration,
        EngineHealth, DBConnectionPoolUsed,
    )
}
```

### Middleware to instrument memory store/recall

```go
// gateway/adapter/handler/memory_handler.go [MODIFY]
// Wrap Store handler:
func (h *MemoryHandler) Store(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    // ... existing logic ...
    telemetry.MemoryStoreDuration.WithLabelValues(engine, req.Type, tenantID).
        Observe(float64(time.Since(start).Milliseconds()))
    telemetry.MemoryStoreTotal.WithLabelValues(engine, req.Type, status).Inc()
}
```

---

## Acceptance Criteria

- [ ] All metrics registered at startup (no panic)
- [ ] `GET :8083/metrics` returns all vnp_memory_* metrics
- [ ] MemoryStoreDuration incremented after each store
- [ ] EngineHealth gauge updated by healthz handler
- [ ] `go test ./shared/pkg/telemetry/...` passes

## Files

```
shared/pkg/telemetry/metrics.go                   [MODIFY — add all metric vars]
shared/pkg/telemetry/init.go                       [NEW/MODIFY — call Init()]
gateway/adapter/handler/memory_handler.go          [MODIFY — instrument]
```

---

**Ghi chú audit:** shared/pkg/telemetry/metrics.go [NEW]: metrics naming convention + labels; gateway/middleware/metrics.go: RequestsTotal/RequestDuration/ActiveConnections/CircuitBreakerState/RateLimitRejected/ResponseSize
