# TASK-MB-011 — `pkg/observability/` & `pkg/middleware/` OTel, Prometheus & Auth Middleware

**Wave:** 4 (Access Layer — xây trước Gateway)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-MB-001 (pkg/config)  
**Ước tính:** 3 giờ  
**Solution tham chiếu:** [SOL-MB-007 §8, §9](../solutions/SOL-MB-007-Shared-Infrastructure.md)

---

## Mục tiêu

Tạo 2 shared packages cho Wave 4:
1. **`pkg/observability/`** — OTel tracer, Prometheus metrics, DB pool monitor, health endpoints
2. **`pkg/middleware/`** — Auth (JWT + project token), logging, tracing, recovery, rate limit, validation

---

## 1. `pkg/observability/`

### File: `pkg/observability/tracer.go`

```go
package observability

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type TracerConfig struct {
    ServiceName string
    Endpoint    string   // OTel Collector endpoint
    Insecure    bool
}

func InitTracer(cfg TracerConfig) (*trace.TracerProvider, error) {
    opts := []otlptracehttp.Option{
        otlptracehttp.WithEndpoint(cfg.Endpoint),
    }
    if cfg.Insecure { opts = append(opts, otlptracehttp.WithInsecure()) }

    exporter, err := otlptracehttp.New(context.Background(), opts...)
    if err != nil { return nil, fmt.Errorf("create OTLP exporter: %w", err) }

    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(cfg.ServiceName),
        )),
        trace.WithSampler(trace.AlwaysSample()),
    )

    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))
    return tp, nil
}

// Span names used across memobase services:
// "process_blobs"           (engine: orchestrator)
// "llm_call.entry_summary"  (engine: LLM call #1)
// "llm_call.extract_topics" (engine: LLM call #2)
// "llm_call.yolo_merge"     (engine: LLM call #3)
// "embed.query"             (context/event: embedding call)
// "db.query.profiles"       (context: PG query)
// "redis.get.profiles"      (context: Redis cache)
// "grpc.{service}.{method}" (all gRPC calls)
```

### File: `pkg/observability/metrics.go`

```go
package observability

var (
    HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "memobase_http_requests_total",
        Help: "Total HTTP requests by path, method, status, and project",
    }, []string{"path", "method", "status", "project_id"})

    HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "memobase_http_request_duration_ms",
        Help:    "HTTP request latency in milliseconds",
        Buckets: []float64{5, 10, 25, 50, 100, 250, 500, 1000},
    }, []string{"path", "method"})

    LLMCalls = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "memobase_llm_calls_total",
        Help: "Total LLM API calls",
    }, []string{"provider", "model", "prompt_type", "status"})

    LLMDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "memobase_llm_duration_ms",
        Help:    "LLM call latency in milliseconds",
        Buckets: []float64{100, 500, 1000, 2000, 5000, 10000, 30000},
    }, []string{"provider", "prompt_type"})

    BufferFlushTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "memobase_buffer_flush_total",
        Help: "Total buffer flushes by project and status",
    }, []string{"project_id", "status"})  // status: success|failed|skipped

    ProfileMergeOps = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "memobase_profile_merge_operations_total",
        Help: "Profile merge operations by action type",
    }, []string{"action"})  // add|update|delete

    EmbeddingCalls = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "memobase_embedding_calls_total",
        Help: "Total embedding API calls",
    }, []string{"provider", "model", "status"})

    CacheHits = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "memobase_cache_hits_total",
        Help: "Cache hits and misses",
    }, []string{"cache", "result"})  // cache: profiles, result: hit|miss

    DBPoolUtilization = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "memobase_db_pool_utilization",
        Help: "DB connection pool utilization ratio (in_use / max_open)",
    }, []string{"service"})
)

// RecordLLMCall records an LLM API call with timing
func RecordLLMCall(provider, promptType string, duration time.Duration, err error) {
    status := "success"
    if err != nil { status = "error" }
    LLMDuration.WithLabelValues(provider, promptType).Observe(float64(duration.Milliseconds()))
    LLMCalls.WithLabelValues(provider, "auto", promptType, status).Inc()
}
```

### File: `pkg/observability/health.go`

```go
// gRPC health + HTTP /healthz endpoint
type HealthServer struct {
    services map[string]func() error  // name → health check fn
}

func NewHealthServer() *HealthServer {
    return &HealthServer{services: make(map[string]func() error)}
}

func (h *HealthServer) Register(name string, check func() error) {
    h.services[name] = check
}

// HTTP handler for /healthz
func (h *HealthServer) HTTPHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        type checkResult struct {
            Status string `json:"status"`
            Error  string `json:"error,omitempty"`
        }
        results := make(map[string]checkResult)
        allOK := true
        for name, check := range h.services {
            if err := check(); err != nil {
                results[name] = checkResult{Status: "unhealthy", Error: err.Error()}
                allOK = false
            } else {
                results[name] = checkResult{Status: "healthy"}
            }
        }
        w.Header().Set("Content-Type", "application/json")
        if !allOK { w.WriteHeader(http.StatusServiceUnavailable) }
        json.NewEncoder(w).Encode(map[string]any{
            "status":   ifThenElse(allOK, "healthy", "unhealthy"),
            "services": results,
        })
    }
}

// Postgres health check
func PostgresHealthCheck(db *pgxpool.Pool) func() error {
    return func() error { return db.Ping(context.Background()) }
}

// Redis health check
func RedisHealthCheck(client *redis.Client) func() error {
    return func() error { return client.Ping(context.Background()).Err() }
}

// NATS health check
func NATSHealthCheck(conn *nats.Conn) func() error {
    return func() error {
        if conn.IsConnected() { return nil }
        return errors.New("nats: not connected")
    }
}
```

### File: `pkg/observability/pool_monitor.go`

```go
func MonitorDBPool(ctx context.Context, pool *pgxpool.Pool, serviceName string) {
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                stats := pool.Stat()
                maxConn := float64(stats.MaxConns())
                if maxConn == 0 { continue }
                utilization := float64(stats.AcquiredConns()) / maxConn
                DBPoolUtilization.WithLabelValues(serviceName).Set(utilization)
                if utilization > 0.8 {
                    slog.Warn("DB pool utilization high",
                        "service", serviceName,
                        "acquired", stats.AcquiredConns(),
                        "max", stats.MaxConns(),
                        "pct", fmt.Sprintf("%.1f%%", utilization*100))
                }
            case <-ctx.Done():
                return
            }
        }
    }()
}
```

---

## 2. `pkg/middleware/`

### File: `pkg/middleware/auth/auth.go`

```go
package auth

// ProjectContext carries authenticated project info through request context
type ProjectContext struct {
    ProjectID string
}

type ContextKey string
const ProjectContextKey ContextKey = "project_context"

func FromContext(ctx context.Context) (*ProjectContext, bool) {
    pc, ok := ctx.Value(ProjectContextKey).(*ProjectContext)
    return pc, ok
}

// ProjectTokenAuth: validates "sk-proj-{projectID}-{secret}" tokens
type ProjectTokenAuthMiddleware struct {
    adminClient AdminClient  // gRPC to memobase-admin
}

func (m *ProjectTokenAuthMiddleware) Handle(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := extractBearerToken(r)
        if token == "" {
            http.Error(w, `{"error":"missing authorization"}`, http.StatusUnauthorized)
            return
        }

        result, err := m.adminClient.ValidateProjectToken(r.Context(), token)
        if err != nil || !result.Valid {
            http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
            return
        }

        ctx := context.WithValue(r.Context(), ProjectContextKey, &ProjectContext{
            ProjectID: result.ProjectID,
        })
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func extractBearerToken(r *http.Request) string {
    auth := r.Header.Get("Authorization")
    if !strings.HasPrefix(auth, "Bearer ") { return "" }
    return strings.TrimPrefix(auth, "Bearer ")
}
```

### File: `pkg/middleware/logging/logging.go`

```go
package logging

func RequestLogger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        next.ServeHTTP(rw, r)
        duration := time.Since(start)

        slog.Info("http request",
            "method",   r.Method,
            "path",     r.URL.Path,
            "status",   rw.statusCode,
            "duration", duration.Milliseconds(),
            "ip",       r.RemoteAddr,
        )

        // Record Prometheus metrics
        observability.HTTPDuration.
            WithLabelValues(r.URL.Path, r.Method).
            Observe(float64(duration.Milliseconds()))
    })
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}
func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}
```

### File: `pkg/middleware/recovery/recovery.go`

```go
package recovery

func PanicRecovery(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rec := recover(); rec != nil {
                slog.Error("panic recovered",
                    "panic", rec,
                    "stack", string(debug.Stack()),
                    "path", r.URL.Path,
                )
                http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

### File: `pkg/middleware/ratelimit/ratelimit.go`

```go
package ratelimit

// Redis sliding window rate limiter
type RateLimiter struct {
    redis  *redis.Client
    limit  int           // requests per window
    window time.Duration // sliding window size
}

func NewRateLimiter(redis *redis.Client, limit int, window time.Duration) *RateLimiter

func (rl *RateLimiter) Handle(keyFunc func(*http.Request) string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := "ratelimit:" + keyFunc(r)

            // Lua script for atomic sliding window check
            count, err := rl.redis.Eval(r.Context(), slidingWindowLua, []string{key},
                rl.window.Milliseconds(), time.Now().UnixMilli(), rl.limit).Int()

            if err != nil {
                // Redis error → allow request (fail open)
                next.ServeHTTP(w, r)
                return
            }

            if count > rl.limit {
                w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.limit))
                http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
                return
            }

            w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(rl.limit-count))
            next.ServeHTTP(w, r)
        })
    }
}

// Sliding window Lua script (atomic ZADD + ZREMRANGEBYSCORE + ZCARD)
var slidingWindowLua = `
    local key = KEYS[1]
    local window = tonumber(ARGV[1])
    local now = tonumber(ARGV[2])
    local limit = tonumber(ARGV[3])
    redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
    redis.call('ZADD', key, now, now)
    redis.call('EXPIRE', key, math.ceil(window/1000))
    return redis.call('ZCARD', key)
`
```

### File: `pkg/middleware/tracing/tracing.go`

```go
package tracing

func OTelMiddleware(serviceName string) func(http.Handler) http.Handler {
    tracer := otel.Tracer(serviceName)
    propagator := otel.GetTextMapPropagator()

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
            spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
            ctx, span := tracer.Start(ctx, spanName,
                oteltrace.WithSpanKind(oteltrace.SpanKindServer),
                oteltrace.WithAttributes(
                    semconv.HTTPMethodKey.String(r.Method),
                    semconv.HTTPURLKey.String(r.URL.String()),
                ),
            )
            defer span.End()
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

---

## Unit Tests

```
TestInitTracer_ValidConfig               → tp created, global tracer set
TestInitTracer_InvalidEndpoint           → exporter creation fails → error
TestHealthServer_AllHealthy              → all checks pass → 200 + "healthy"
TestHealthServer_OneUnhealthy            → 1 check fails → 503 + "unhealthy"
TestHealthServer_NoChecks                → empty → 200 + "healthy"
TestPostgresHealthCheck_Connected        → pool.Ping success → nil
TestPostgresHealthCheck_Disconnected     → ping error → error
TestMonitorDBPool_HighUtilization        → 90% used → Warn log + metric set
TestMonitorDBPool_StopsOnCtxCancel       → ctx cancel → goroutine exits
TestProjectTokenAuth_ValidToken          → admin validates → 200 + ProjectContext in ctx
TestProjectTokenAuth_MissingToken        → no Authorization header → 401
TestProjectTokenAuth_InvalidToken        → admin rejects → 401
TestProjectTokenAuth_SetsContext         → valid → ctx has ProjectContextKey
TestFromContext_WithProjectContext        → set context → retrieve ProjectContext
TestFromContext_EmptyContext             → no key → false
TestRequestLogger_LogsMethod            → GET /path → slog.Info called
TestRequestLogger_RecordsMetrics        → duration metric recorded
TestPanicRecovery_CatchesPanic          → panic("test") → 500 returned, not crash
TestPanicRecovery_NoopOnNormal          → normal handler → 200
TestRateLimiter_BelowLimit              → 5 requests, limit=10 → all 200
TestRateLimiter_ExceedsLimit            → 11 requests, limit=10 → 11th is 429
TestRateLimiter_RedisError_FailOpen     → Redis error → request allowed
TestRateLimiter_SetsHeaders             → X-RateLimit-Remaining set
TestExtractBearerToken_Valid            → "Bearer sk-proj-xxx" → "sk-proj-xxx"
TestExtractBearerToken_NoPrefix         → "Token xxx" → ""
TestExtractBearerToken_Empty            → "" → ""
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
go get go.opentelemetry.io/otel@latest
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp@latest
go get github.com/prometheus/client_golang@latest
go mod tidy

go build ./pkg/observability/...
go build ./pkg/middleware/...
go test ./pkg/observability/... ./pkg/middleware/... -v -count=1 -race
```

---

## Ghi chú triển khai

- `promauto.NewCounterVec`: auto-registers with default Prometheus registry
- Rate limiter: fail open (không block request khi Redis down) — safety over strictness
- `ProjectContext` trong HTTP context: dùng typed key `ContextKey("project_context")` để tránh conflicts
- OTel tracer: `AlwaysSample()` cho dev; production dùng `TraceIDRatioBased(0.1)` (10% sampling)
- Middleware stack (từ ngoài vào trong): `Recovery → Logging → Tracing → Auth → RateLimit → Handler`
- `pkg/middleware/validation/`: struct validation dùng `go-playground/validator` — implement nếu cần
