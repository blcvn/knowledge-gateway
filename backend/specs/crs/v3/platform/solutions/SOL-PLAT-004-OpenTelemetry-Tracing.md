# SOL-PLAT-004 — Solution: OpenTelemetry Distributed Tracing

| Field | Value |
|---|---|
| **Solution ID** | SOL-PLAT-004 |
| **CR** | [CR-PLAT-004](../../../../docs/crs/v3/platform/CR-PLAT-004-OpenTelemetry-Tracing.md) |
| **TDD ref** | [09-shared-packages.md §telemetry](../../../tdd/architecture/09-shared-packages.md) · [backend-api-specs.md §12.11-Observability](../../../tdd/backend-api-specs.md) |
| **Status** | Open |
| **Priority** | 🟡 High |

**Trạng thái:** 🔄 Partial  
**Ghi chú audit:** OTel InitProvider + OTLP exporter implemented; HTTP tracing middleware not applied
---

## 1. Phân tích kiến trúc

Theo TDD `09-shared-packages.md §telemetry`, `shared/pkg/telemetry` đã có Prometheus metrics nhưng **OTel distributed tracing chưa implement**. Gateway config đã có `OTELConfig` (từ `01-gateway.md §3`).

Cần implement:
- W3C TraceContext propagation (`traceparent` header) qua HTTP → gRPC
- Span creation tại gateway + injection vào gRPC metadata
- LLM span enrichment (model, tokens, cost)
- OTLP exporter → Jaeger/Grafana Tempo
- Console observability APIs

---

## 2. Giải pháp

### 2.1 `shared/pkg/telemetry/tracer.go` [MODIFY]

```go
package telemetry

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type TracerConfig struct {
    Endpoint    string  // OTLP endpoint (Jaeger / Grafana Tempo)
    ServiceName string
    SampleRate  float64 // 0.0–1.0; 1.0 = 100% sampling
}

// InitTracer sets up OTel TraceProvider with OTLP gRPC exporter
func InitTracer(ctx context.Context, cfg TracerConfig) (func(), error) {
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(cfg.Endpoint),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
    }

    res := resource.NewWithAttributes(
        semconv.SchemaURL,
        semconv.ServiceName(cfg.ServiceName),
        semconv.ServiceVersion("v3"),
    )

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
    )

    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},  // W3C Trace Context (traceparent header)
        propagation.Baggage{},
    ))

    return func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        tp.Shutdown(ctx)
    }, nil
}

// Tracer returns a named tracer for a service/component
func Tracer(name string) trace.Tracer {
    return otel.GetTracerProvider().Tracer(name)
}
```

### 2.2 `gateway/internal/infra/middleware/tracing.go` [NEW]

```go
package middleware

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// TracingMiddleware extracts/creates trace context and creates HTTP span
// Span naming convention: gateway.{method}.{path_template}
func TracingMiddleware(next http.Handler) http.Handler {
    return otelhttp.NewHandler(next, "gateway",
        otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
            // e.g. "gateway.memory.store" from POST /v1/memory/store
            parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/"), "/")
            return fmt.Sprintf("gateway.%s", strings.Join(parts[:min(len(parts), 2)], "."))
        }),
        otelhttp.WithFilter(func(r *http.Request) bool {
            // Skip health checks and metrics from tracing
            return r.URL.Path != "/healthz" && r.URL.Path != "/metrics"
        }),
    )
}

// InjectTraceToGRPC propagates W3C trace context into gRPC metadata
func InjectTraceToGRPC(ctx context.Context) context.Context {
    md, _ := metadata.FromOutgoingContext(ctx)
    if md == nil {
        md = metadata.New(nil)
    }
    // Inject traceparent + tracestate into gRPC metadata
    otel.GetTextMapPropagator().Inject(ctx, &GRPCMetadataCarrier{md: md})
    return metadata.NewOutgoingContext(ctx, md)
}

// GRPCMetadataCarrier implements TextMapCarrier for gRPC metadata
type GRPCMetadataCarrier struct {
    md metadata.MD
}

func (c *GRPCMetadataCarrier) Get(key string) string {
    vals := c.md.Get(key)
    if len(vals) > 0 { return vals[0] }
    return ""
}
func (c *GRPCMetadataCarrier) Set(key, val string) { c.md.Set(key, val) }
func (c *GRPCMetadataCarrier) Keys() []string       { return nil }
```

### 2.3 LLM Span Instrumentation — `shared/pkg/telemetry/llm_span.go` [NEW]

```go
package telemetry

// LLMSpanAttributes adds LLM-specific attributes to a span
// per CR-PLAT-004 §5: model, input_tokens, output_tokens, estimated_cost_usd
func LLMSpanAttributes(span trace.Span, model string, inputTokens, outputTokens int64, costUSD float64) {
    span.SetAttributes(
        attribute.String("llm.model", model),
        attribute.Int64("llm.input_tokens", inputTokens),
        attribute.Int64("llm.output_tokens", outputTokens),
        attribute.Float64("llm.cost_usd", costUSD),
    )
}

// StartLLMSpan creates a child span for an LLM call
func StartLLMSpan(ctx context.Context, model, task string) (context.Context, trace.Span) {
    tr := Tracer("llm")
    ctx, span := tr.Start(ctx, "llm.complete",
        trace.WithAttributes(
            attribute.String("llm.model", model),
            attribute.String("llm.task", task),
        ),
    )
    return ctx, span
}

// StartDBSpan creates a child span for a database query
func StartDBSpan(ctx context.Context, dbType, operation string) (context.Context, trace.Span) {
    tr := Tracer("db")
    ctx, span := tr.Start(ctx, fmt.Sprintf("db.%s.query", dbType),
        trace.WithAttributes(
            attribute.String("db.system", dbType),
            attribute.String("db.operation", operation),
        ),
    )
    return ctx, span
}
```

### 2.4 Secret Redaction — `shared/pkg/telemetry/redaction.go` [NEW]

```go
package telemetry

// RedactedSpan wraps a span and filters out sensitive attribute keys
type RedactedSpan struct {
    inner trace.Span
}

var sensitiveKeys = map[string]bool{
    "http.request.header.authorization": true,
    "http.request.header.x-api-key":     true,
    "x-api-key":                          true,
    "authorization":                      true,
}

func (s *RedactedSpan) SetAttributes(attrs ...attribute.KeyValue) {
    filtered := make([]attribute.KeyValue, 0, len(attrs))
    for _, a := range attrs {
        if !sensitiveKeys[strings.ToLower(string(a.Key))] {
            filtered = append(filtered, a)
        }
    }
    s.inner.SetAttributes(filtered...)
}
```

### 2.5 Console Observability Handler — `gateway/adapter/handler/observability.go` [NEW]

```go
package handler

// Handler for /v1/console/observability/* routes
// Proxies to vnp-observability gRPC service (or queries Jaeger/Tempo directly)

// GET /v1/console/observability/traces
func (h *ObservabilityHandler) ListTraces(w http.ResponseWriter, r *http.Request) {
    limit := queryInt(r, "limit", 100)
    traces, err := h.obsSvc.ListTraces(r.Context(), limit)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "obs_error", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, traces)
}

// GET /v1/console/observability/traces/{id}
func (h *ObservabilityHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
    traceID := chi.URLParam(r, "id")
    trace, err := h.obsSvc.GetTrace(r.Context(), traceID)
    if err != nil {
        writeError(w, http.StatusNotFound, "trace_not_found", traceID)
        return
    }
    writeJSON(w, http.StatusOK, trace)
}

// GET /v1/console/observability/errors
// GET /v1/console/observability/costs
// GET /v1/console/observability/metrics
```

### 2.6 Span Naming Convention (from CR-PLAT-004 §3)

```
gateway.memory.store
  ├── classifier.classify    (LLM span: model, tokens, cost)
  ├── engine.dispatch        → graphiti-ingestion.ingest
  │     ├── llm.extract_entities
  │     ├── db.neo4j.query
  │     └── db.pgvector.upsert_embeddings
  └── nats.publish

gateway.memory.recall
  └── search_hub.fan_out
        ├── graphiti-search.search
        ├── cognee-search.search
        └── rrf.fusion
```

---

## 3. File Changes

| File | Action | Mô tả |
|---|---|---|
| `shared/pkg/telemetry/tracer.go` | MODIFY | Add InitTracer với OTLP exporter + W3C propagation |
| `shared/pkg/telemetry/llm_span.go` | NEW | LLM span attributes + StartLLMSpan helper |
| `shared/pkg/telemetry/redaction.go` | NEW | Secret redaction wrapper cho spans |
| `gateway/internal/infra/middleware/tracing.go` | NEW | HTTP tracing middleware + gRPC metadata injection |
| `gateway/adapter/handler/observability.go` | NEW | Console observability API handlers |
| `gateway/adapter/handler/router.go` | MODIFY | Register `/v1/console/observability/*` routes |
| `deployment/dev/docker-compose.server.yaml` | MODIFY | Add Jaeger/Grafana Tempo container |
| `deployment/dev/grafana/` | NEW | Grafana dashboard provisioning |

---

## 4. Acceptance Criteria

- [ ] TraceID propagated via W3C Trace Context (`traceparent` header HTTP → gRPC metadata)
- [ ] Spans include: service name, operation, latency_ms, error flag
- [ ] LLM spans include: `llm.model`, `llm.input_tokens`, `llm.output_tokens`, `llm.cost_usd`
- [ ] DB spans include: `db.system`, `db.operation`
- [ ] Traces exportable to OTLP endpoint (Jaeger/Tempo)
- [ ] `GET /v1/console/observability/traces` returns last 100 traces
- [ ] Trace detail shows waterfall (parent → child spans with latency bars)
- [ ] Secret Redaction: no API keys, JWT tokens, or passwords in span attributes
- [ ] Health/metrics paths excluded from tracing (no noise)

---

## 5. Dependencies

- OTel Go SDK (`go.opentelemetry.io/otel`)
- `otlptracegrpc` exporter
- `otelhttp` + `otelgrpc` contrib packages
- Jaeger or Grafana Tempo running in deployment stack
- `OTEL_ENDPOINT` env var (e.g. `jaeger:4317`)
- `OTEL_SAMPLE_RATE` env var (default `0.1` = 10% sampling in prod)
