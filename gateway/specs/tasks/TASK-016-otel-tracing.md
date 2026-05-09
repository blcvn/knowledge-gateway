---
id: TASK-016
title: OTel Distributed Tracing — Jaeger Integration
service: vnp-gateway
version: 1.0.0
status: Ready
priority: P1
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
depends_on: [TASK-012]
estimate: 3h
---

## Mục Tiêu

Implement OpenTelemetry distributed tracing. Span mọi request from middleware → downstream gRPC call. Export to Jaeger.

## Phạm Vi

### Files cần tạo
- `gateway/internal/infra/otel/tracer.go` — OTel tracer setup (OTLP exporter)
- `gateway/internal/infra/otel/propagation.go` — W3C TraceContext propagation
- Update `middleware.go` — inject trace spans
- Update `registry.go` — propagate trace context to gRPC metadata

### Acceptance Criteria

- [ ] AC-1: OTel tracer initialized with OTLP exporter
- [ ] AC-2: Each HTTP request creates a parent span with request_id, tenant_id attributes
- [ ] AC-3: gRPC Forward() creates child span with service, method attributes
- [ ] AC-4: W3C TraceContext propagated in gRPC metadata
- [ ] AC-5: Jaeger UI shows full trace: HTTP → middleware → gRPC → downstream
- [ ] AC-6: Graceful shutdown flushes pending spans

## Verification

```bash
go build ./internal/infra/otel/...
OTEL_ENDPOINT=localhost:4317 go run ./cmd/main.go
# Open Jaeger UI → see traces
```
