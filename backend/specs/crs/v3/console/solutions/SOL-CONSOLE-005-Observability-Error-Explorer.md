# Solution: SOL-CONSOLE-005 — Observability Error Explorer APIs

**CR:** CR-CONSOLE-005
**TDD refs:** `architecture/09-shared-packages.md §telemetry`, `architecture/08-platform-services.md §VNP Observability`
**Version:** v3/console

---

## 1. Architecture

Error/trace data from:
- **Structured logs** → PostgreSQL `error_log` table (persisted by `vnp-observability`)
- **LLM cost** → Prometheus counter `vnp_llm_cost_usd_total{engine, model}`
- **Traces** → OTLP backend (Jaeger HTTP API or Grafana Tempo)

---

## 2. Error Aggregation

```go
// gateway/adapter/handler/observability_handler.go [NEW]
type ObservabilityHandler struct {
    prometheus   port.PrometheusClient
    db           *pgxpool.Pool
    traceBackend port.TraceBackendClient
}

// GET /v1/console/observability/errors?range=1h&engine=cognee
func (h *ObservabilityHandler) GetErrors(w http.ResponseWriter, r *http.Request) {
    rangeStr := r.URL.Query().Get("range")
    engine   := r.URL.Query().Get("engine")
    tenantID := tenant.FromContext(r.Context())
    if rangeStr == "" { rangeStr = "1h" }

    // Parse range: "1h" → NOW - 1h
    since := parseRange(rangeStr)

    query := `
        SELECT error_type, engine, trace_id, message, created_at
        FROM error_log
        WHERE tenant_id = $1 AND created_at >= $2
        ` + func() string { if engine != "" { return "AND engine = $3" }; return "" }()

    args := []any{tenantID, since}
    if engine != "" { args = append(args, engine) }

    rows, err := h.db.Query(r.Context(), query, args...)
    if err != nil { writeError(w, 500, "query_failed", err.Error()); return }
    defer rows.Close()

    byType   := map[string]int{}
    byEngine := map[string]int{}
    recent   := []map[string]any{}

    for rows.Next() {
        var errType, eng, traceID, message string
        var createdAt time.Time
        rows.Scan(&errType, &eng, &traceID, &message, &createdAt)
        byType[errType]++; byEngine[eng]++
        if len(recent) < 20 {
            recent = append(recent, map[string]any{
                "error_type": errType, "engine": eng,
                "trace_id": traceID, "message": message,
                "timestamp": createdAt,
            })
        }
    }

    total := 0
    for _, v := range byType { total += v }
    writeJSON(w, 200, map[string]any{
        "total_errors": total, "by_type": byType,
        "by_engine": byEngine, "recent_errors": recent,
    })
}

// GET /v1/console/observability/costs?range=7d
func (h *ObservabilityHandler) GetCosts(w http.ResponseWriter, r *http.Request) {
    rangeStr := r.URL.Query().Get("range")
    if rangeStr == "" { rangeStr = "7d" }
    ctx := r.Context()

    engines := []string{"cognee", "graphiti", "zep", "memobase", "openviking", "supermemory"}
    byEngine := map[string]map[string]any{}

    for _, eng := range engines {
        costQuery  := fmt.Sprintf(`sum(vnp_llm_cost_usd_total{engine="%s"})`, eng)
        tokInQuery := fmt.Sprintf(`sum(vnp_llm_tokens_total{engine="%s", direction="in"})`, eng)
        tokOutQuery := fmt.Sprintf(`sum(vnp_llm_tokens_total{engine="%s", direction="out"})`, eng)

        cost, _   := h.prometheus.QueryScalar(ctx, costQuery)
        tokIn, _  := h.prometheus.QueryScalar(ctx, tokInQuery)
        tokOut, _ := h.prometheus.QueryScalar(ctx, tokOutQuery)
        byEngine[eng] = map[string]any{"usd": cost, "tokens_in": tokIn, "tokens_out": tokOut}
    }

    // Per-day series
    dailySeries, _ := h.prometheus.QueryRange(ctx,
        `sum(rate(vnp_llm_cost_usd_total[24h])) * 86400`,
        rangeStr, "24h")

    // Total
    total, _ := h.prometheus.QueryScalar(ctx, `sum(vnp_llm_cost_usd_total)`)

    writeJSON(w, 200, map[string]any{
        "total_usd": total, "by_engine": byEngine, "by_day": dailySeries,
    })
}

// GET /v1/console/observability/traces?limit=50&has_error=true&engine=cognee
func (h *ObservabilityHandler) GetTraces(w http.ResponseWriter, r *http.Request) {
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit == 0 { limit = 50 }
    hasError := r.URL.Query().Get("has_error") == "true"
    engine   := r.URL.Query().Get("engine")
    tenantID := tenant.FromContext(r.Context())

    traces, err := h.traceBackend.QueryTraces(r.Context(), &TraceQuery{
        TenantID: tenantID, Limit: limit,
        HasError: hasError, Engine: engine,
        FromTime: time.Now().Add(-24 * time.Hour),
    })
    if err != nil { writeError(w, 500, "traces_failed", err.Error()); return }
    writeJSON(w, 200, map[string]any{"traces": traces})
}

// GET /v1/console/observability/traces/{id}
func (h *ObservabilityHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
    traceID := chi.URLParam(r, "id")
    trace, err := h.traceBackend.GetTrace(r.Context(), traceID)
    if err != nil { writeError(w, 404, "trace_not_found", ""); return }
    writeJSON(w, 200, trace)
}
```

---

## 3. Error Log Table (Migration)

```sql
-- deployment/dev/migrations/0049_error_log.sql
CREATE TABLE IF NOT EXISTS error_log (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  TEXT NOT NULL,
    error_type TEXT NOT NULL,
    engine     TEXT,
    trace_id   TEXT,
    message    TEXT,
    stack      TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_error_log_tenant_time ON error_log(tenant_id, created_at DESC);

-- auto-purge: keep 30 days
CREATE OR REPLACE FUNCTION purge_old_errors() RETURNS void AS $$
    DELETE FROM error_log WHERE created_at < NOW() - INTERVAL '30 days';
$$ LANGUAGE sql;
```

---

## 4. Error Publishing (in error handler middleware)

```go
// shared/pkg/telemetry/error_publisher.go [NEW]
// Called from recovery middleware when a handler panics or returns 5xx
func PublishError(ctx context.Context, db *pgxpool.Pool, errType, engine, traceID, message string) {
    tenantID := tenant.FromContext(ctx)
    db.Exec(ctx, `INSERT INTO error_log (tenant_id, error_type, engine, trace_id, message) VALUES ($1,$2,$3,$4,$5)`,
        tenantID, errType, engine, traceID, message)
}
```

---

## 5. File Changes

| File | Action |
|---|---|
| `gateway/adapter/handler/observability_handler.go` | **[NEW]** |
| `gateway/internal/port/trace_backend.go` | **[NEW]** TraceBackendClient interface |
| `gateway/internal/adapter/jaeger/client.go` | **[NEW]** Jaeger HTTP API client |
| `shared/pkg/telemetry/error_publisher.go` | **[NEW]** |
| `deployment/dev/migrations/0049_error_log.sql` | **[NEW]** |
| `gateway/adapter/handler/router.go` | **[MODIFY]** observability routes |
