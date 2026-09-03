# Change Request: CR-CONSOLE-008 — Infrastructure Health Console APIs

**CR ID:** CR-CONSOLE-008
**Component:** `backend/gateway`, `backend/services/vnp-observability`
**Priority:** 🟠 Medium
**Status:** Open
**Version:** v3 / Console
**Feature:** [F24](../../../features/24-infrastructure-health/README.md)
**Depends On:** CR-ENT-006

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P2-03 | Platform Engineer | Không biết database/infra health state |

---

## 2. APIs

### `GET /v1/console/infrastructure/health`

```json
{
  "overall": "degraded",
  "components": {
    "postgres": {
      "status": "healthy", "latency_ms": 3,
      "connections": {"used": 47, "max": 100}
    },
    "neo4j": {"status": "healthy", "latency_ms": 12},
    "redis": {
      "status": "healthy", "latency_ms": 1,
      "memory_used_mb": 256, "memory_max_mb": 2048
    },
    "nats": {
      "status": "healthy", "latency_ms": 2,
      "subjects_count": 127, "messages_pending": 12
    },
    "minio": {"status": "healthy", "latency_ms": 8},
    "neo4j_backup": {"status": "degraded", "error": "Last backup 48h ago"}
  }
}
```

### `GET /v1/console/infrastructure/resources`

```json
{
  "services": [
    {
      "name": "cognee-ingestion",
      "cpu_pct": 34, "memory_mb": 512,
      "goroutines": 47, "gc_pause_ms": 1.2
    }
  ]
}
```

### `GET /v1/console/infrastructure/connections`

DB connection pool stats per service.

### `GET /v1/console/infrastructure/alerts`

```json
{
  "active_alerts": [
    {
      "id": "alert-1",
      "severity": "warning",
      "component": "neo4j_backup",
      "message": "Backup overdue by 24h",
      "since": "2026-09-02T10:00:00Z"
    }
  ]
}
```

---

## 3. Implementation Notes

- Component health: call specific health probes (pg PING, neo4j PING, redis PING, NATS PING)
- Resource stats: runtime.MemStats + pprof goroutine count
- Alerts: evaluate alert rules every 60s, persist to DB

---

## 4. Acceptance Criteria

- [ ] All 5 infra components checked (pg, neo4j, redis, nats, minio)
- [ ] Connection pool stats per service
- [ ] Active alerts with severity (warning/critical)
- [ ] Response < 3s (all checks parallel)
- [ ] `GET /healthz/live` always fast (< 5ms, no downstream)
