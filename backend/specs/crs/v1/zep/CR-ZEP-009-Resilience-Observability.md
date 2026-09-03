# Change Request: CR-ZEP-009 — Resilience & Observability Infrastructure

**CR ID:** CR-ZEP-009  
**Component:** `pkg/resilience/`, `pkg/observability/`, `pkg/middleware/` [UPGRADE SHARED]  
**Priority:** Medium  
**Status:** In Progress
**Reference:** Zep PRD §7.1-7.3, SRS §8, specs/services/08-shared-packages.md

---

## 1. Mô tả

Nâng cấp infrastructure của VNP Memory theo chuẩn Zep enterprise:

1. **Circuit Breaker**: sony/gobreaker per-downstream-service để ngăn cascade failures.
2. **Exponential Backoff Retry**: 200ms → 30s, max 15 retries cho advisory locks và external calls.
3. **10-Layer Middleware Stack**: CORS, request logging, health check, size limiter, request ID, timeout, real IP, clean path, version header, OTel tracing.
4. **PostgreSQL Advisory Locks**: SHA-256 hash-based locks cho concurrent metadata updates.
5. **Anonymous Telemetry**: Opt-out usage tracking.
6. **Request Constraints**: 5MB max payload, 30s timeout enforced by middleware.

---

## 2. Vấn đề hiện tại

- VNP Memory chưa có Circuit Breaker pattern → cascade failure khi downstream service chết.
- Thiếu standardized retry với exponential backoff.
- Chưa có advisory lock cho concurrent metadata updates → potential race condition.
- Middleware stack chưa đầy đủ (thiếu request ID injection, version header).

---

## 3. Thay đổi đề xuất

### 3.1. [UPGRADE] `pkg/resilience/`

```go
// Circuit Breaker — sony/gobreaker
type CircuitBreaker struct {
    breaker *gobreaker.CircuitBreaker
}

// Per-service: zep-thread, zep-memory, zep-graph, zep-search
var breakers = map[string]*CircuitBreaker{
    "thread": newBreaker("thread", threshold=5, timeout=30*time.Second),
    "graph":  newBreaker("graph", threshold=3, timeout=60*time.Second),
    // ...
}

// Exponential Backoff Retry
// min=200ms, max=30s, max_retries=15
func RetryWithBackoff(ctx context.Context, fn func() error) error {
    backoff := ExponentialBackoff{Min: 200*ms, Max: 30*s, MaxRetries: 15}
    return backoff.Do(ctx, fn)
}
```

### 3.2. [UPGRADE] Gateway Middleware Stack (10 layers, ordered)

```go
// chi router middleware — thứ tự quan trọng
r.Use(
    cors.Handler(cors.Options{AllowedOrigins: []string{"*"}}),     // 1. CORS
    middleware.RequestLogger,                                        // 2. Structured logging
    middleware.Heartbeat("/healthz"),                                // 3. Health check
    middleware.RequestSizeLimiter(5*1024*1024),                     // 4. 5MB max
    middleware.RequestID,                                            // 5. UUID injection
    middleware.Timeout(30*time.Second),                              // 6. 30s timeout
    middleware.RealIP,                                               // 7. Real IP extraction
    middleware.CleanPath,                                            // 8. Clean path
    middleware.SetHeader("X-API-Version", "2.0"),                   // 9. Version header
    otelchi.Middleware("zep-gateway", otelchi.WithChiRoutes(r)),   // 10. OTel tracing
)
```

### 3.3. [UPGRADE] PostgreSQL Advisory Locks

```go
// pkg/metadata/advisory_lock.go
// SHA-256 hash of session/user ID → int64 lock key
func AdvisoryLockKey(id string) int64 {
    hash := sha256.Sum256([]byte(id))
    return int64(binary.BigEndian.Uint64(hash[:8]))
}

// Usage in JSONB metadata update
func (r *MetadataRepo) MergePatch(ctx context.Context, id string, patch map[string]any) error {
    lockKey := AdvisoryLockKey(id)
    _, err := r.db.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockKey)
    defer r.db.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", lockKey)
    // UPDATE SET metadata = metadata || $2::jsonb WHERE id = $3
}
```

### 3.4. [UPGRADE] Structured Request Logging

```go
// Log format per request:
// {"level":"info","method":"POST","path":"/api/v2/sessions/123/memory",
//  "duration_ms":45,"status":200,"request_id":"uuid-v4","bytes":1024}
```

### 3.5. [NEW] Anonymous Telemetry

```go
// Opt-out via config:
// telemetry:
//   disabled: false  # set true to opt out

type TelemetryEvent struct {
    Event     string
    ProjectID string  // hashed, anonymous
    Version   string
}

// Events tracked (anonymous): service_start, api_request_count, error_rate
// No PII sent
```

### 3.6. Request Constraints Enforcement

| Constraint | Value | Layer |
|-----------|-------|-------|
| Max payload | 5MB | Middleware (position 4) |
| Request timeout | 30s | Middleware (position 6) |
| Concurrent metadata lock | SHA-256 advisory | Repository |
| Circuit breaker threshold | 5 failures → open | Resilience package |

---

## 4. Acceptance Criteria

- [ ] Graph service down → circuit breaker opens sau 5 failures → subsequent requests fail fast (không chờ timeout).
- [ ] Concurrent PATCH metadata từ 10 goroutines → chỉ 1 goroutine update tại một thời điểm (advisory lock).
- [ ] Request > 5MB → rejected ngay bởi middleware với 413 status.
- [ ] Request latency > 30s → context cancelled, 504 Gateway Timeout.
- [ ] Mỗi request log có đủ fields: method, path, duration_ms, status, request_id.
- [ ] Telemetry disabled khi `telemetry.disabled: true` trong config.
