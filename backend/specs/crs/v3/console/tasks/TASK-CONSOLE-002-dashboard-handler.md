# TASK-CONSOLE-002 — Console Dashboard Handler

| Field | Value |
|---|---|
| **Task ID** | TASK-CONSOLE-002 |
| **Wave** | 2 |
| **Solution** | [SOL-CONSOLE-001](../solutions/SOL-CONSOLE-001-Dashboard-APIs.md) §2 |
| **Component** | `gateway/adapter/handler/console_handler.go` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-CONSOLE-001 |
| **Estimated** | 3h |

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** DashboardHandler: Health/Metrics/Throughput/Heatmap handlers (console.go:55+)
---

## Mục tiêu

Implement 4 dashboard endpoints: health, metrics, throughput, heatmap.

---

## Công việc cụ thể

### 1. Tạo `gateway/adapter/handler/console_handler.go` [NEW]

Implement 4 methods per SOL-CONSOLE-001 §2:
- `DashboardHealth`: call internal healthz
- `DashboardMetrics`: parallel Prometheus queries + Redis active sessions
- `DashboardThroughput`: Prometheus range query
- `DashboardHeatmap`: Redis INCR keys per engine per date

### 2. Heatmap Redis update trong `memory_handler.go` [MODIFY]

```go
// After successful store response:
go func() {
    date := time.Now().Format("2006-01-02")
    key := fmt.Sprintf("heatmap:%s:%s:%s", tenantID, engine, date)
    h.redis.Incr(ctx, key)
    h.redis.Expire(ctx, key, 8*24*time.Hour)
}()
```

### 3. Routes trong `router.go` [MODIFY]

```go
r.Get("/v1/console/dashboard/health",     consoleHandler.DashboardHealth)
r.Get("/v1/console/dashboard/metrics",    consoleHandler.DashboardMetrics)
r.Get("/v1/console/dashboard/throughput", consoleHandler.DashboardThroughput)
r.Get("/v1/console/dashboard/heatmap",    consoleHandler.DashboardHeatmap)
```

### 4. Response caching middleware (30s TTL)

```go
// For dashboard endpoints: cache response 30s
func CacheMiddleware(ttl time.Duration) func(http.Handler) http.Handler {
    cache := sync.Map{}
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := r.URL.String()
            if cached, ok := cache.Load(key); ok {
                entry := cached.(*cacheEntry)
                if time.Since(entry.ts) < ttl {
                    w.Header().Set("Content-Type", "application/json")
                    w.Header().Set("X-Cache", "HIT")
                    w.Write(entry.body); return
                }
            }
            // ... capture response and cache it
        })
    }
}
```

---

## Acceptance Criteria

- [ ] `/dashboard/health` → real healthz data (not mock)
- [ ] `/dashboard/metrics` → Prometheus aggregated values
- [ ] `/dashboard/heatmap` → Redis heatmap per engine per day
- [ ] Responses cached 30s (X-Cache: HIT on 2nd request)
- [ ] All 4 endpoints < 500ms

## Files

```
gateway/adapter/handler/console_handler.go  [NEW]
gateway/adapter/handler/memory_handler.go   [MODIFY] heatmap update
gateway/adapter/handler/router.go           [MODIFY] dashboard routes
```
