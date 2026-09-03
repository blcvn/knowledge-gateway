# Change Request: CR-CONSOLE-001 — Console Dashboard Backend APIs

**CR ID:** CR-CONSOLE-001
**Component:** `backend/gateway`, `backend/services/vnp-observability`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Console
**Feature:** [F15](../../../features/15-console-dashboard/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P2-02 | Platform Engineer | 35+ services → monitoring fragmented |

**Before:** Dashboard không có backend APIs, hardcoded mock data.
**After:** Real-time data từ Prometheus, gRPC health checks, NATS metrics.

---

## 2. APIs

### `GET /v1/console/dashboard/health`
```json
{
  "overall": "degraded",
  "services": {
    "cognee-ingestion": {"status": "healthy", "latency_ms": 12},
    "graphiti-ingestion": {"status": "unhealthy", "error": "timeout"}
  },
  "timestamp": "2026-09-03T..."
}
```

### `GET /v1/console/dashboard/metrics`
```json
{
  "memories_stored_today": 1423,
  "recalls_today": 8901,
  "active_sessions": 12,
  "active_agents": 5,
  "error_rate_pct": 0.8,
  "p95_store_ms": 142,
  "p95_recall_ms": 287
}
```

### `GET /v1/console/dashboard/throughput?range=1h`
```json
{
  "interval_seconds": 60,
  "series": [
    {"ts": "...", "stores": 45, "recalls": 123},
    ...
  ]
}
```

### `GET /v1/console/dashboard/heatmap?days=7`
```json
{
  "engines": ["graphiti", "cognee", "zep", "memobase", "ov", "sm"],
  "days": [
    {"date": "2026-09-01", "counts": {"graphiti": 120, "cognee": 340, ...}},
    ...
  ]
}
```

---

## 3. Implementation Notes

- Health data: call `GET /healthz` internally and parse
- Metrics data: query Prometheus via HTTP API
- Throughput: aggregate from `vnp_memory_store_total` counter
- Heatmap: aggregate from Redis event log (24h buckets)

---

## 4. Acceptance Criteria

- [ ] `/dashboard/health` reflects real healthz state (not mock)
- [ ] `/dashboard/metrics` aggregated from Prometheus
- [ ] `/dashboard/throughput` returns last 1h/6h/24h timeseries
- [ ] `/dashboard/heatmap` shows per-engine activity per day
- [ ] All endpoints cache for 30s (not hit Prometheus on every request)
- [ ] Response < 500ms
