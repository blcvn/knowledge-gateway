# Feature 25 — Observability & Tracing

> **Loại:** Operations | **Priority:** High | **Status:** Implemented

## Mô tả

Observability layer cung cấp distributed tracing, metrics, logging, và cost tracking cho toàn bộ VNP Memory. Dựa trên OpenTelemetry standards, với Prometheus cho metrics và structured JSON logging (slog).

---

## Business Logic

### OpenTelemetry Distributed Tracing

Mỗi request được trace end-to-end:
- **Trace**: Một request lifecycle từ gateway → engine → database
- **Span**: Mỗi bước trong lifecycle (e.g., "graphiti.ingest", "neo4j.write", "llm.call")
- **Propagation**: TraceID được propagate qua tất cả gRPC calls → unified trace view
- **Export**: Traces có thể export sang Jaeger, Zipkin, hoặc OTLP collector

### Prometheus Metrics

Key metrics exposed tại port `:8083/metrics`:
- **Latency**: P50/P95/P99 per endpoint và engine
- **Throughput**: Requests/second, memories stored/minute
- **Error Rate**: 4xx/5xx per endpoint
- **LLM Costs**: Tokens consumed per flush, per request
- **Queue Depth**: NATS message counts per subject

### Structured Logging

Tất cả logs theo JSON format (slog):
```json
{
  "time": "2026-06-18T10:30:00Z",
  "level": "INFO",
  "msg": "memory stored",
  "trace_id": "abc123",
  "tenant_id": "tenant-xyz",
  "engine": "cognee",
  "duration_ms": 42
}
```

**Secret Redaction**: Tự động redact API keys, passwords, tokens trong logs.

### Cost Tracking

Track LLM token costs:
- Tokens per LLM call
- Cost per engine (USD estimate)
- Daily/monthly cost trends
- Alert khi cost vượt ngưỡng

### Error Aggregation

- Error counts by type, engine, endpoint
- Error rate trending
- Recent error details với stack traces (nếu applicable)

---

## Dataflow

### Tracing Flow

```
HTTP Request
        │
        ▼
Gateway (RequestID middleware)
        │
        ├── Generate TraceID + RequestID
        ├── Inject into request context
        │
        ▼
Handler → gRPC call to engine
        │
        ├── Propagate TraceID in gRPC metadata
        │
        ▼
Engine service
        │
        ├── Create child span: "engine.process"
        ├── Create grandchild span: "db.query"
        │
        └── Export spans to OTLP collector
                  └── (Jaeger / Grafana Tempo)
```

### Metrics Collection

```
Every request → Middleware (Metrics)
        │
        ├── Record: http_request_duration_seconds{endpoint, method, status}
        ├── Record: memory_store_total{engine, type}
        └── Record: llm_tokens_total{engine, model}
                  │
                  ▼
        Prometheus scrapes /metrics every 15s
                  │
                  ▼
        Grafana dashboards visualize
```

### Console Observability Endpoints

```
GET /v1/console/observability/metrics  → Prometheus metrics summary
GET /v1/console/observability/traces   → Recent trace list
GET /v1/console/observability/traces/{id} → Trace detail (waterfall view)
GET /v1/console/observability/errors   → Error aggregation
GET /v1/console/observability/costs    → LLM cost breakdown
```

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/v1/console/observability/metrics` | Metrics summary |
| `GET` | `/v1/console/observability/traces` | Trace list |
| `GET` | `/v1/console/observability/traces/{id}` | Trace detail |
| `GET` | `/v1/console/observability/errors` | Error aggregation |
| `GET` | `/v1/console/observability/costs` | LLM cost tracking |
| `GET` | `/v1/admin/metrics` | Raw Prometheus metrics |
| `GET` | `/healthz` (port :8083) | Full health + metrics |

---

## Services

| Service | Vai trò |
|---------|---------|
| `obs-service` | Observability infrastructure, OpenTelemetry collector |
| `vnp-observability` | Platform-level observability APIs |
| `observe-service` | Agent-level observation (Feature 08) |

---

## Business Value

### Pain Points được giải quyết

- **PP-P2-02 (Monitoring fragmented)**
- **PP-P8-02 (No quality metrics)**

### Actors hưởng lợi

P2 Platform Engineer, P3 ML/AI Engineer, P8 Product Manager

### Giải pháp tham chiếu

- [S10 — Zero-config Infrastructure](../../bussiness/solutions/S10-infrastructure-simplicity.md)

### ROI / Kết quả đo được

> MTTD 30min → 5min | LLM cost tracking per engine | Error rate breakdown

---

*Xem thêm: [Pain Points](../../bussiness/painpoints/README.md) | [Solutions](../../bussiness/solutions/README.md)*
