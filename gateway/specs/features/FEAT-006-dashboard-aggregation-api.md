---
id: FEAT-006
title: Dashboard Aggregation API
service: vnp-gateway
version: 1.0.0
status: Draft
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
linked_ux: "ux_spec.md §6.1 Dashboard / Overview"
---

## Mục Tiêu

Cung cấp REST API endpoints cho Dashboard màn hình Overview của Console UI:
- Aggregated health check cho tất cả 6 engines + KGS
- Realtime throughput metrics (ingest/sec, recall/sec, embeddings/sec)
- Engine-specific latency và queue depth
- Top-level KPI cards data

## Bối Cảnh Nghiệp Vụ

Console UI Dashboard (UX §6.1) cần:
1. **Engine Health Grid** — trạng thái 7 engines (6 memory + KGS)
2. **KPI Cards** — Active Agents, Recall Latency, Token Savings, Graph Growth, Error Rate
3. **Memory Flow Visualization** — throughput per engine
4. **AI Memory Heatmap** — density, retrieval frequency, stale memories

## Scope

### In Scope
- `GET /v1/console/dashboard/health` — Aggregated health status
- `GET /v1/console/dashboard/metrics` — KPI metrics
- `GET /v1/console/dashboard/throughput` — Per-engine throughput
- `GET /v1/console/dashboard/heatmap` — Memory heatmap data

### Out of Scope
- WebSocket streaming (covered in FEAT-012)
- Historical trend analytics (future scope)

## Thiết Kế Kỹ Thuật

### API Contract

#### GET `/v1/console/dashboard/health`

**Response (200):**
```json
{
  "engines": [
    {
      "name": "cognee",
      "role": "Semantic Memory",
      "status": "healthy|degraded|unhealthy",
      "latency_p50_ms": 823,
      "latency_p95_ms": 1250,
      "queue_depth": 41,
      "uptime_seconds": 86400,
      "last_check": "2026-05-13T12:00:00Z"
    }
  ],
  "overall_status": "healthy",
  "total_services": 35,
  "healthy_services": 34
}
```

#### GET `/v1/console/dashboard/metrics`

**Response (200):**
```json
{
  "active_agents": 12,
  "recall_latency_p50_ms": 180,
  "recall_latency_p95_ms": 420,
  "context_savings_pct": 67.5,
  "graph_nodes_total": 125000,
  "graph_edges_total": 890000,
  "graph_growth_24h": 1234,
  "error_rate_pct": 0.3,
  "active_sessions": 42,
  "active_profiles": 890,
  "memory_versions": 3400
}
```

#### GET `/v1/console/dashboard/throughput`

**Query params:** `?window=5m|15m|1h|24h`

**Response (200):**
```json
{
  "window": "5m",
  "engines": {
    "cognee": { "ingest_per_sec": 12.5, "recall_per_sec": 45.2, "embed_per_sec": 8.1 },
    "graphiti": { "ingest_per_sec": 8.3, "recall_per_sec": 22.1, "embed_per_sec": 0 },
    "zep": { "ingest_per_sec": 15.0, "recall_per_sec": 30.5, "embed_per_sec": 0 },
    "openviking": { "ingest_per_sec": 3.2, "recall_per_sec": 10.8, "embed_per_sec": 2.1 },
    "memobase": { "ingest_per_sec": 20.1, "recall_per_sec": 18.3, "embed_per_sec": 0 },
    "supermemory": { "ingest_per_sec": 5.5, "recall_per_sec": 12.0, "embed_per_sec": 3.4 }
  }
}
```

### Internal Architecture

1. **Handler:** `adapter/http/dashboard_handler.go`
2. **Usecase:** `usecase/dashboard.go` — fan-out health checks to all services
3. **Port:** `port/input.go` — add `DashboardUseCase` interface
4. **Client:** Use existing `ServiceRegistry` + add `HealthCheck()` method

### Business Logic

- Fan-out health checks concurrently to all 7 engine groups
- Timeout per engine: 5s
- If engine unreachable → status = "unhealthy", don't block other engines
- Metrics scraped from Prometheus via `vnp-platform` aggregation
- Queue depth from NATS JetStream consumer info

## Acceptance Criteria

- [ ] AC-1: `GET /health` returns status for all 7 engine groups within 5s
- [ ] AC-2: `GET /metrics` returns all KPI fields with non-null values
- [ ] AC-3: `GET /throughput` respects `window` parameter
- [ ] AC-4: Unreachable engine returns `unhealthy` status, others still return data
- [ ] AC-5: All endpoints require `admin` role in auth context

## Test Requirements
- Unit tests: DashboardUseCase fan-out logic, timeout handling
- Integration tests: Mock gRPC services, verify aggregation
- Minimum coverage: 80%
