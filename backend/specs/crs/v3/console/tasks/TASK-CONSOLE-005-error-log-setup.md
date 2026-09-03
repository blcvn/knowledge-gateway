# TASK-CONSOLE-005 — Error Log DB + Observability Error Handler

| Field | Value |
|---|---|
| **Task ID** | TASK-CONSOLE-005 |
| **Wave** | 2 |
| **Solution** | [SOL-CONSOLE-005](../solutions/SOL-CONSOLE-005-Observability-Error-Explorer.md) §2,§3 |
| **Component** | `gateway/adapter/handler/`, `shared/pkg/telemetry/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 3h |

---

## Mục tiêu

Setup `error_log` table + error publisher + `ObservabilityHandler` (errors, costs, traces).

---

## Công việc cụ thể

### 1. DB Migration `deployment/dev/migrations/0049_error_log.sql` [NEW]

```sql
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
```

### 2. Tạo `shared/pkg/telemetry/error_publisher.go` [NEW]

```go
package telemetry

func PublishError(ctx context.Context, db *pgxpool.Pool, errType, engine, traceID, message string) {
    tenantID := tenant.FromContext(ctx)
    if tenantID == "" { return }
    db.Exec(ctx, `INSERT INTO error_log (tenant_id, error_type, engine, trace_id, message) VALUES ($1,$2,$3,$4,$5)`,
        tenantID, errType, engine, traceID, message)
}
```

### 3. Tạo `gateway/internal/port/trace_backend.go` [NEW]

```go
type TraceQuery struct {
    TenantID string; Limit int; HasError bool; Engine string; FromTime time.Time
}
type TraceItem struct {
    TraceID string; Operation string; DurationMs int64; HasError bool; Error string; Timestamp time.Time
}
type SpanItem struct {
    SpanID string; ParentID string; Operation string; DurationMs int64; HasError bool; Tags map[string]string
}
type TraceDetail struct { TraceID string; Spans []SpanItem }

type TraceBackendClient interface {
    QueryTraces(ctx context.Context, q *TraceQuery) ([]TraceItem, error)
    GetTrace(ctx context.Context, traceID string) (*TraceDetail, error)
}
```

### 4. Tạo `gateway/adapter/handler/observability_handler.go` [NEW]

Implement:
- `GET /v1/console/observability/errors` → query error_log with filters
- `GET /v1/console/observability/costs` → Prometheus `vnp_llm_cost_usd_total`
- `GET /v1/console/observability/traces` → trace backend query
- `GET /v1/console/observability/traces/{id}` → trace detail

### 5. Routes `router.go` [MODIFY]

```go
r.Get("/v1/console/observability/errors",       obsHandler.GetErrors)
r.Get("/v1/console/observability/costs",        obsHandler.GetCosts)
r.Get("/v1/console/observability/traces",       obsHandler.GetTraces)
r.Get("/v1/console/observability/traces/{id}",  obsHandler.GetTrace)
```

---

## Acceptance Criteria

- [ ] Error log table: INSERT works from PublishError
- [ ] `/observability/errors?range=1h` → errors from last 1h
- [ ] `/observability/costs?range=7d` → per-engine USD totals
- [ ] `/observability/traces?has_error=true` → only errored traces

## Files

```
deployment/dev/migrations/0049_error_log.sql              [NEW]
shared/pkg/telemetry/error_publisher.go                   [NEW]
gateway/internal/port/trace_backend.go                    [NEW]
gateway/internal/adapter/jaeger/client.go                 [NEW]
gateway/adapter/handler/observability_handler.go          [NEW]
gateway/adapter/handler/router.go                         [MODIFY]
```
