# TASK-PLAT-011 — HTTP Tracing Middleware & gRPC Propagation

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-011 |
| **Wave** | 3 (Observability) |
| **Solution** | [SOL-PLAT-004](../solutions/SOL-PLAT-004-OpenTelemetry-Tracing.md) §2.2 |
| **Component** | `gateway/internal/infra/middleware/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-PLAT-010 |
| **Estimated** | 2h |

---

## Mục tiêu

Implement OTel HTTP tracing middleware cho gateway. Span naming convention: `gateway.{resource}.{operation}`. Filter out health/metrics paths.

---

## Công việc cụ thể

### 1. Tạo `gateway/internal/infra/middleware/tracing.go` [NEW]

```go
package middleware

import (
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel"
)

// tracingSkipPaths: don't trace these paths (reduces noise)
var tracingSkipPaths = map[string]bool{
    "/healthz":  true,
    "/metrics":  true,
    "/readyz":   true,
    "/":         true,
}

// TracingMiddleware creates OTel spans for all HTTP requests
// Span naming: gateway.memory.store, gateway.cognee.search, etc.
func TracingMiddleware(next http.Handler) http.Handler {
    return otelhttp.NewHandler(next, "gateway",
        otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
            return formatSpanName(r.URL.Path)
        }),
        otelhttp.WithFilter(func(r *http.Request) bool {
            return !tracingSkipPaths[r.URL.Path]
        }),
        otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
    )
}

// formatSpanName converts URL path to span name
// /v1/memory/store → gateway.memory.store
// /v1/cognee/search → gateway.cognee.search
// /v1/console/dashboard/health → gateway.console.dashboard
func formatSpanName(path string) string {
    path = strings.TrimPrefix(path, "/v1/")
    parts := strings.Split(path, "/")
    if len(parts) >= 2 {
        return fmt.Sprintf("gateway.%s", strings.Join(parts[:2], "."))
    }
    if len(parts) == 1 && parts[0] != "" {
        return fmt.Sprintf("gateway.%s", parts[0])
    }
    return "gateway.unknown"
}

// AddTenantToSpan adds tenant_id attribute to current span
func AddTenantToSpan(ctx context.Context, tenantID string) {
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(attribute.String("tenant.id", tenantID))
}

// PropagateToGRPC injects W3C traceparent into outgoing gRPC metadata
func PropagateToGRPC(ctx context.Context) context.Context {
    return telemetry.InjectToGRPC(ctx)
}
```

### 2. Modify `gateway/adapter/client/` gRPC clients [MODIFY] — inject trace context

```go
// In each gRPC client call (e.g., graphiti-ingestion, cognee-search, etc.),
// inject trace context before the call:

func (c *GraphitiIngestionClient) Ingest(ctx context.Context, req *ingestpb.IngestRequest) (*ingestpb.IngestResponse, error) {
    ctx = middleware.PropagateToGRPC(ctx) // ← inject traceparent into gRPC metadata
    return c.client.Ingest(ctx, req)
}
```

### 3. Modify `gateway/adapter/handler/router.go` [MODIFY] — apply tracing middleware

```go
// Apply tracing as the outermost middleware (before auth)
r.Use(middleware.TracingMiddleware)
r.Use(authMiddleware)
r.Use(rateLimitMiddleware)
// ...
```

### 4. Unit test `gateway/internal/infra/middleware/tracing_test.go` [NEW]

```go
func TestFormatSpanName(t *testing.T) {
    tests := []struct {
        path     string
        expected string
    }{
        {"/v1/memory/store", "gateway.memory.store"},
        {"/v1/cognee/search", "gateway.cognee.search"},
        {"/v1/console/dashboard/health", "gateway.console.dashboard"},
        {"/healthz", "gateway.unknown"}, // filtered out anyway
        {"/v1/auth/login", "gateway.auth.login"},
    }
    for _, tt := range tests {
        assert.Equal(t, tt.expected, formatSpanName(tt.path))
    }
}
```

---

## Acceptance Criteria

- [ ] OTel span created for each HTTP request to `/v1/*`
- [ ] `/healthz`, `/metrics` paths excluded from tracing
- [ ] Span names follow convention: `gateway.{resource}.{operation}`
- [ ] W3C `traceparent` header propagated to all downstream gRPC calls
- [ ] Tenant ID added as span attribute (`tenant.id`)
- [ ] Spans exported to Jaeger (visible in Jaeger UI)
- [ ] `go build ./gateway/...` passes

## Files

```
gateway/internal/infra/middleware/tracing.go        [NEW]
gateway/internal/infra/middleware/tracing_test.go   [NEW]
gateway/adapter/client/*.go                         [MODIFY — inject trace context]
gateway/adapter/handler/router.go                   [MODIFY — apply tracing middleware]
```
