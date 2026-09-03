# Change Request: CR-CONSOLE-005 — Observability Error Explorer APIs

**CR ID:** CR-CONSOLE-005
**Component:** `backend/gateway`, `backend/services/vnp-observability`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Console
**Feature:** [F25](../../../features/25-observability-tracing/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P2-02 | Platform Engineer | Error logs phân tán, không aggregate được |
| PP-P8-02 | Product Manager | Không biết LLM cost theo engine |

---

## 2. APIs

### `GET /v1/console/observability/errors?range=1h&engine=cognee`

```json
{
  "total_errors": 47,
  "error_rate_pct": 2.3,
  "by_type": {
    "LLMTimeout": 12,
    "Neo4jConnectionFailed": 3,
    "RateLimitExceeded": 32
  },
  "by_engine": {
    "cognee": 15, "graphiti": 2, "zep": 30
  },
  "recent_errors": [
    {
      "error_id": "err-abc",
      "type": "LLMTimeout",
      "engine": "cognee",
      "trace_id": "trace-xyz",
      "message": "LLM call timed out after 30s",
      "timestamp": "2026-09-03T10:05:00Z"
    }
  ]
}
```

### `GET /v1/console/observability/costs?range=7d`

```json
{
  "total_usd": 12.47,
  "by_engine": {
    "cognee": {"usd": 4.12, "tokens_in": 1200000, "tokens_out": 340000},
    "graphiti": {"usd": 3.21, "tokens_in": 890000, "tokens_out": 210000}
  },
  "by_day": [
    {"date": "2026-09-01", "usd": 1.82},
    {"date": "2026-09-02", "usd": 2.10}
  ],
  "alert_threshold_usd": 50.0,
  "projected_monthly_usd": 380.0
}
```

### `GET /v1/console/observability/traces?limit=50&has_error=true`

```json
{
  "traces": [
    {
      "trace_id": "abc123",
      "operation": "memory.store",
      "duration_ms": 1240,
      "has_error": true,
      "error": "LLMTimeout",
      "timestamp": "..."
    }
  ]
}
```

### `GET /v1/console/observability/traces/{id}`

Full trace waterfall: parent → child spans with latency breakdown.

---

## 3. Implementation Notes

- Errors: aggregate from structured logs (slog → loki or postgres error_log table)
- Costs: read from `vnp_llm_cost_usd_total` Prometheus counter
- Traces: read from OTLP backend (Jaeger API or Tempo API)

---

## 4. Acceptance Criteria

- [ ] Error aggregation: by type, by engine, by time range
- [ ] Cost breakdown: per engine, per day, projected monthly
- [ ] Trace list: filter by has_error, engine, time range
- [ ] Trace detail: waterfall with all child spans
- [ ] Cost alert when approaching threshold
