# Change Request: CR-ENT-006 — Infrastructure Health & Aggregated Healthz

**CR ID:** CR-ENT-006
**Component:** `backend/gateway`, `backend/apps/memory`
**Priority:** 🟠 Medium
**Status:** Open
**Version:** v5 / Enterprise & Operations
**Solution:** [S10 — Infrastructure Simplicity](../../../bussiness/solutions/S10-infrastructure-simplicity.md)
**Features:** [F24](../../../features/24-infrastructure-health/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P2-01 | Platform Engineer | 35+ services → 35+ health endpoints phải check riêng lẻ |
| PP-P2-02 | Platform Engineer | Không biết service nào down khi có lỗi |

**Before:** `curl service1:port/health && curl service2:port/health && ...` (35 commands)
**After:** 1 `GET /healthz` → aggregated status tất cả 35+ services.

---

## 2. Health Response Format

```json
GET :8083/healthz
→ {
    "status": "degraded",         // healthy | degraded | unhealthy
    "version": "1.0.0",
    "uptime_seconds": 3600,
    "services": {
      "cognee-ingestion":     {"status": "healthy",   "latency_ms": 2},
      "cognee-search":        {"status": "healthy",   "latency_ms": 3},
      "graphiti-ingestion":   {"status": "healthy",   "latency_ms": 1},
      "memobase-engine":      {"status": "unhealthy", "error": "LLM timeout"},
      "ov-fs":                {"status": "healthy",   "latency_ms": 5},
      "zep-memory":           {"status": "healthy",   "latency_ms": 4}
    },
    "infrastructure": {
      "postgres":  {"status": "healthy",   "latency_ms": 1},
      "neo4j":     {"status": "healthy",   "latency_ms": 8},
      "redis":     {"status": "healthy",   "latency_ms": 0},
      "nats":      {"status": "healthy",   "latency_ms": 0},
      "minio":     {"status": "degraded",  "error": "disk usage 87%"}
    }
  }
```

**HTTP status codes:**
- `200` → all healthy
- `207` → degraded (some services unhealthy)
- `503` → critical services down

---

## 3. Thay đổi đề xuất

### 3.1 `backend/gateway/internal/adapter/handler/health_handler.go` [MODIFY]

```go
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
    services := h.registry.ListServices()
    results := make(map[string]ServiceHealth, len(services))
    
    var wg sync.WaitGroup
    for _, svc := range services {
        wg.Add(1)
        go func(name string) {
            defer wg.Done()
            start := time.Now()
            conn := h.registry.Lookup(name)
            err := grpc_health.Check(r.Context(), conn)
            results[name] = ServiceHealth{
                Status:    healthStatus(err),
                LatencyMs: time.Since(start).Milliseconds(),
                Error:     errStr(err),
            }
        }(svc)
    }
    wg.Wait()
    
    overall := aggregateStatus(results)
    writeJSON(w, httpStatus(overall), HealthResponse{
        Status:   overall,
        Services: results,
        Infra:    h.checkInfra(r.Context()),
    })
}
```

---

## 4. Acceptance Criteria

- [ ] `GET :8083/healthz` trả aggregated status tất cả 35+ services
- [ ] Parallel check — không sequential (≤ 200ms total)
- [ ] 3 HTTP status: 200 (all healthy), 207 (degraded), 503 (critical down)
- [ ] Infrastructure check: postgres, neo4j, redis, nats, minio
- [ ] Prometheus metric: `vnp_engine_health{engine}` gauge (1=up, 0=down)
- [ ] Graceful: nếu 1 service không respond, đánh dấu unhealthy, tiếp tục
