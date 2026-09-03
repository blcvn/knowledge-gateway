# TASK-BE-010 — Console Observability Handler

| Field | Value |
|---|---|
| **Task ID** | TASK-BE-010 |
| **Layer** | Backend — Go |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-006 CR-008](../solutions/SOL-006-Adaptive-to-Org-Solutions.md) + [SOL-007 §8](../solutions/SOL-007-Gap-Fixes.md) |
| **Priority** | 🟠 P1 |
| **Depends On** | — |
| **Estimated** | 3h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `vnp-platform/migrations/0007_create_error_aggregates.sql` |
| CREATE | `gateway/internal/adapter/handler/console_observability_handler.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

### Migration: `0007_create_error_aggregates.sql`

```sql
-- +migrate Up
CREATE TABLE IF NOT EXISTS error_aggregates (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    service         TEXT        NOT NULL,
    message         TEXT        NOT NULL,
    message_hash    TEXT        NOT NULL,    -- MD5 để group identical errors
    count           INT         NOT NULL DEFAULT 1,
    last_occurrence TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    stack           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, message_hash)
);

CREATE INDEX idx_errors_tenant ON error_aggregates(tenant_id, last_occurrence DESC);

-- +migrate Down
DROP TABLE IF EXISTS error_aggregates;
```

### Handler: `console_observability_handler.go`

```go
package handler

type ConsoleObservabilityHandler struct {
    promClient PromClient      // Prometheus HTTP API
    traceRepo  TraceRepository // OTEL trace store (PostgreSQL hoặc Jaeger API)
    db         *sql.DB         // error_aggregates
    bifrost    BifrostClient   // LLM cost tracking
}

// GET /v1/console/observability/metrics
// Returns: {latency: MetricPoint[], error_rate: MetricPoint[], throughput: MetricPoint[]}
func (h *ConsoleObservabilityHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Query Prometheus time-series (last 1 hour, 5-minute steps)
    latency, _   := h.promClient.QueryRange(ctx, `histogram_quantile(0.95, rate(vnp_recall_latency_ms_bucket[5m]))`, "1h", "5m")
    errRate, _   := h.promClient.QueryRange(ctx, `rate(vnp_errors_total[5m])`, "1h", "5m")
    throughput, _ := h.promClient.QueryRange(ctx, `rate(vnp_requests_total[5m])`, "1h", "5m")

    httputil.JSON(w, 200, map[string]any{
        "latency":    mapToMetricPoints(latency, "p95"),
        "error_rate": mapToMetricPoints(errRate, "error_rate"),
        "throughput": mapToMetricPoints(throughput, "throughput"),
    })
}

// GET /v1/console/observability/traces
// Query params: service, status, operation, from, to, limit
func (h *ConsoleObservabilityHandler) GetTraces(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()
    filters := map[string]string{}
    for _, k := range []string{"service", "status", "operation", "from", "to"} {
        if v := q.Get(k); v != "" { filters[k] = v }
    }
    limit := 50
    if l, _ := strconv.Atoi(q.Get("limit")); l > 0 { limit = l }
    traces, _ := h.traceRepo.Query(r.Context(), filters, limit)
    httputil.JSON(w, 200, traces)
}

// GET /v1/console/observability/traces/{id}
func (h *ConsoleObservabilityHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
    span, err := h.traceRepo.Get(r.Context(), r.PathValue("id"))
    if err != nil { httputil.Error(w, "Not found", "NOT_FOUND", 404); return }
    httputil.JSON(w, 200, span)
}

// GET /v1/console/observability/errors
func (h *ConsoleObservabilityHandler) GetErrors(w http.ResponseWriter, r *http.Request) {
    tenantID := authctx.TenantID(r.Context())
    q := r.URL.Query()
    where := "WHERE tenant_id = $1"; args := []any{tenantID}
    if svc := q.Get("service"); svc != "" { where += " AND service = $2"; args = append(args, svc) }

    rows, _ := h.db.QueryContext(r.Context(),
        `SELECT id, message, service, count, last_occurrence, stack
         FROM error_aggregates `+where+` ORDER BY last_occurrence DESC LIMIT 50`, args...)
    // scan and return
}

// GET /v1/console/observability/costs
func (h *ConsoleObservabilityHandler) GetCosts(w http.ResponseWriter, r *http.Request) {
    tenantID := authctx.TenantID(r.Context())
    costs, _ := h.bifrost.GetCostSummary(r.Context(), tenantID)
    httputil.JSON(w, 200, costs)
}
```

### Routes

```go
mux.HandleFunc("GET /v1/console/observability/metrics",      authMiddleware(obs.GetMetrics))
mux.HandleFunc("GET /v1/console/observability/traces",       authMiddleware(obs.GetTraces))
mux.HandleFunc("GET /v1/console/observability/traces/{id}",  authMiddleware(obs.GetTrace))
mux.HandleFunc("GET /v1/console/observability/errors",       authMiddleware(obs.GetErrors))
mux.HandleFunc("GET /v1/console/observability/costs",        authMiddleware(obs.GetCosts))
```
