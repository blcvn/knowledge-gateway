# TASK-PLAT-010 — OTel Tracer Init & OTLP Exporter

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-010 |
| **Wave** | 3 (Observability) |
| **Solution** | [SOL-PLAT-004](../solutions/SOL-PLAT-004-OpenTelemetry-Tracing.md) §2.1 |
| **Component** | `shared/pkg/telemetry/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 3h |

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** shared/pkg/telemetry/otel.go: InitProvider() with OTLP gRPC exporter + propagators
---

## Mục tiêu

Extend `shared/pkg/telemetry` để support OTel distributed tracing với OTLP gRPC exporter và W3C TraceContext propagation.

---

## Công việc cụ thể

### 1. Modify `shared/pkg/telemetry/tracer.go` [MODIFY/NEW]

```go
package telemetry

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
    "go.opentelemetry.io/otel/trace"
)

type TracerConfig struct {
    Endpoint    string  // OTLP gRPC endpoint (e.g. "jaeger:4317")
    ServiceName string  // e.g. "vnp-gateway", "graphiti-ingestion"
    SampleRate  float64 // 0.0–1.0; use 0.1 in prod, 1.0 in dev
    Insecure    bool    // skip TLS for local dev
}

// InitTracer sets up global OTel TraceProvider with OTLP exporter
// Returns shutdown func — call on service shutdown
func InitTracer(ctx context.Context, cfg TracerConfig) (shutdownFn func(), err error) {
    opts := []otlptracegrpc.Option{
        otlptracegrpc.WithEndpoint(cfg.Endpoint),
    }
    if cfg.Insecure {
        opts = append(opts, otlptracegrpc.WithInsecure())
    }

    exporter, err := otlptracegrpc.New(ctx, opts...)
    if err != nil {
        return nil, fmt.Errorf("create OTLP exporter: %w", err)
    }

    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName(cfg.ServiceName),
            semconv.ServiceVersion("v3"),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("create resource: %w", err)
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
    )

    // Set global provider + W3C TraceContext propagator
    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(
        propagation.NewCompositeTextMapPropagator(
            propagation.TraceContext{}, // traceparent header
            propagation.Baggage{},
        ),
    )

    return func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        _ = tp.Shutdown(ctx)
    }, nil
}

// Tracer returns a named tracer (use package/service name)
func Tracer(name string) trace.Tracer {
    return otel.GetTracerProvider().Tracer(name)
}

// GRPCMetadataCarrier implements TextMapCarrier for gRPC metadata propagation
type GRPCMetadataCarrier struct {
    MD metadata.MD
}

func (c *GRPCMetadataCarrier) Get(key string) string {
    vals := c.MD.Get(key)
    if len(vals) > 0 {
        return vals[0]
    }
    return ""
}

func (c *GRPCMetadataCarrier) Set(key, val string) {
    c.MD.Set(key, val)
}

func (c *GRPCMetadataCarrier) Keys() []string {
    return nil
}

// InjectToGRPC injects current trace context into outgoing gRPC metadata
func InjectToGRPC(ctx context.Context) context.Context {
    md, ok := metadata.FromOutgoingContext(ctx)
    if !ok {
        md = metadata.New(nil)
    }
    carrier := &GRPCMetadataCarrier{MD: md}
    otel.GetTextMapPropagator().Inject(ctx, carrier)
    return metadata.NewOutgoingContext(ctx, carrier.MD)
}

// ExtractFromGRPC extracts trace context from incoming gRPC metadata
func ExtractFromGRPC(ctx context.Context) context.Context {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return ctx
    }
    carrier := &GRPCMetadataCarrier{MD: md}
    return otel.GetTextMapPropagator().Extract(ctx, carrier)
}
```

### 2. Update `go.mod` in `shared/pkg/telemetry/` [MODIFY] — add OTel deps

```
require (
    go.opentelemetry.io/otel v1.28.0
    go.opentelemetry.io/otel/sdk v1.28.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0
    go.opentelemetry.io/otel/trace v1.28.0
    go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.53.0
    go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.53.0
)
```

### 3. Modify `deployment/dev/docker-compose.server.yaml` [MODIFY] — add Jaeger

```yaml
services:
  jaeger:
    image: jaegertracing/all-in-one:1.58
    ports:
      - "4317:4317"   # OTLP gRPC
      - "16686:16686" # Jaeger UI
    environment:
      - COLLECTOR_OTLP_ENABLED=true
```

---

## Acceptance Criteria

- [ ] `InitTracer()` sets global OTel provider with OTLP gRPC exporter
- [ ] W3C `traceparent` header propagated via `propagation.TraceContext{}`
- [ ] `InjectToGRPC()` / `ExtractFromGRPC()` work for HTTP → gRPC trace propagation
- [ ] Sampler: 10% in prod (`OTEL_SAMPLE_RATE=0.1`), 100% in dev (`OTEL_SAMPLE_RATE=1.0`)
- [ ] `go work sync` passes after adding OTel deps to go.mod
- [ ] Jaeger service added to docker-compose

## Files

```
shared/pkg/telemetry/tracer.go                        [MODIFY]
shared/pkg/telemetry/go.mod                           [MODIFY — add OTel deps]
deployment/dev/docker-compose.server.yaml             [MODIFY — add Jaeger]
```
