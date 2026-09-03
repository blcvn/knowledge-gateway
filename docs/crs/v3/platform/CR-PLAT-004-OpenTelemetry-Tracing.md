# Change Request: CR-PLAT-004 — OpenTelemetry Distributed Tracing

**CR ID:** CR-PLAT-004
**Component:** `backend/shared/pkg/telemetry`, `backend/gateway`, all services
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Platform
**Feature:** [F25](../../../features/25-observability-tracing/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P2-02 | Platform Engineer | Không trace được request path qua 35+ services |
| PP-P3-01 | ML Engineer | Không biết LLM call nào gây latency spike |

**Before:** Logs phân tán, không correlate được.
**After:** Single trace ID theo request từ gateway → engine → DB → LLM.

---

## 2. Tracing Architecture

```
HTTP Request → Gateway
  │ Generate TraceID + SpanID
  │ Inject into grpc-metadata: "traceparent"
  ▼
Engine Service (gRPC)
  │ Extract TraceID, create child span
  │ Create grandchild span for DB/LLM calls
  ▼
OTLP Collector → Jaeger / Grafana Tempo
```

---

## 3. Span Naming Convention

```
gateway.memory.store
  ├── classifier.classify (LLM)
  ├── engine.dispatch → graphiti-ingestion.ingest
  │     ├── llm.extract_entities
  │     ├── neo4j.upsert_nodes
  │     └── pgvector.upsert_embeddings
  └── nats.publish

gateway.memory.recall
  └── search_hub.fan_out
        ├── graphiti-search.search
        ├── cognee-search.search
        └── rrf.fusion
```

---

## 4. Observability Console APIs

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/console/observability/traces` | Recent traces list |
| `GET` | `/v1/console/observability/traces/{id}` | Trace waterfall detail |
| `GET` | `/v1/console/observability/errors` | Error aggregation |
| `GET` | `/v1/console/observability/costs` | LLM cost breakdown |
| `GET` | `/v1/console/observability/metrics` | Metrics summary |

---

## 5. Key Spans to Instrument

- `gateway.auth` — JWT/API key validation
- `gateway.memory.{store|recall|forget}` — top-level memory ops
- `engine.{name}.{operation}` — per engine operation
- `llm.complete` — every LLM call (with model, tokens, cost)
- `db.{postgres|neo4j}.query` — database queries

---

## 6. Acceptance Criteria

- [ ] TraceID propagated via W3C Trace Context (`traceparent` header)
- [ ] Spans include: service name, operation, latency, error flag
- [ ] LLM spans include: model, input_tokens, output_tokens, estimated_cost_usd
- [ ] Traces exportable to OTLP endpoint (Jaeger/Tempo)
- [ ] `GET /v1/console/observability/traces` returns last 100 traces
- [ ] Trace detail shows waterfall (parent → child spans)
- [ ] Secret Redaction: no API keys/tokens in span attributes
