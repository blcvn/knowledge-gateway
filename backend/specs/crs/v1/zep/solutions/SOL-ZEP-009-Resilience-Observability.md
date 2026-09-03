# Solution: SOL-ZEP-009 — Resilience & Observability Infrastructure

**CR ID:** CR-ZEP-009  
**Solution ID:** SOL-ZEP-009  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Nâng cấp `pkg/` (shared packages) của VNP Memory với: **Circuit Breaker** (sony/gobreaker per-service), **Exponential Backoff Retry** (200ms→30s, max 15), **10-Layer Gateway Middleware Stack** (chi), **PostgreSQL Advisory Locks** (SHA-256 hash-based), và **Anonymous Telemetry** với opt-out. Tất cả thay đổi là shared packages — tái sử dụng bởi tất cả Zep services.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| `pkg/` shared packages | `pkg/` | Có: các utilities cơ bản |
| Gateway middleware | `gateway/infra/middleware/` | Có: auth, ratelimit — thiếu 10 layers đầy đủ |
| OpenTelemetry | `pkg/otel/` hoặc `services/obs-service/` | Có: tracing/metrics cơ bản |
| Redis cache | Infrastructure | Có |

### Gap phân tích

- Không có Circuit Breaker → cascade failure khi graph service down
- Thiếu standardized exponential backoff retry
- Middleware stack chưa đầy đủ (thiếu: request ID, version header, request size limiter)
- Advisory lock chưa có shared package (mỗi service tự implement — duplicate code)
- Không có anonymous telemetry với opt-out

---

## 3. Thiết kế Giải pháp

### 3.1. Cấu trúc Packages Mới

```
pkg/
├── resilience/
│   ├── circuit_breaker.go    # sony/gobreaker wrapper
│   └── retry.go              # Exponential backoff (200ms→30s, max 15)
├── metadata/
│   └── advisory_lock.go      # SHA-256 hash → int64 → pg_advisory_lock
├── middleware/
│   └── stack.go              # 10-layer chi middleware stack
└── telemetry/
    └── tracker.go            # Anonymous usage tracking (opt-out)
```

### 3.2. Circuit Breaker Package

```go
// pkg/resilience/circuit_breaker.go

package resilience

import (
    "context"
    "fmt"
    "time"

    "github.com/sony/gobreaker"
)

type CircuitBreakerConfig struct {
    Name        string
    MaxFailures uint32        // failures before tripping (default: 5)
    Timeout     time.Duration // how long to stay open (default: 30s)
    OnStateChange func(name string, from, to gobreaker.State)
}

type CircuitBreaker struct {
    breaker *gobreaker.CircuitBreaker
    name    string
}

func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
    settings := gobreaker.Settings{
        Name:        cfg.Name,
        MaxRequests: 1,         // 1 trial request in half-open state
        Interval:    10 * time.Second,
        Timeout:     cfg.Timeout,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            return counts.ConsecutiveFailures >= cfg.MaxFailures
        },
        OnStateChange: func(name string, from, to gobreaker.State) {
            slog.Warn("circuit breaker state change",
                "breaker", name,
                "from", from.String(),
                "to", to.String(),
            )
            if cfg.OnStateChange != nil {
                cfg.OnStateChange(name, from, to)
            }
        },
    }
    return &CircuitBreaker{
        breaker: gobreaker.NewCircuitBreaker(settings),
        name:    cfg.Name,
    }
}

// Execute runs fn through the circuit breaker
func (cb *CircuitBreaker) Execute(fn func() (any, error)) (any, error) {
    result, err := cb.breaker.Execute(fn)
    if err == gobreaker.ErrOpenState {
        return nil, fmt.Errorf("circuit breaker %q is open: %w", cb.name, ErrCircuitOpen)
    }
    return result, err
}

// Predefined circuit breakers per Zep service
type ZepCircuitBreakers struct {
    Thread *CircuitBreaker
    Memory *CircuitBreaker
    Graph  *CircuitBreaker  // More sensitive: graph extraction is slow
    Search *CircuitBreaker
    User   *CircuitBreaker
}

func NewZepCircuitBreakers() *ZepCircuitBreakers {
    return &ZepCircuitBreakers{
        Thread: NewCircuitBreaker(CircuitBreakerConfig{Name: "zep-thread", MaxFailures: 5, Timeout: 30 * time.Second}),
        Memory: NewCircuitBreaker(CircuitBreakerConfig{Name: "zep-memory", MaxFailures: 5, Timeout: 30 * time.Second}),
        Graph:  NewCircuitBreaker(CircuitBreakerConfig{Name: "zep-graph", MaxFailures: 3, Timeout: 60 * time.Second}),
        Search: NewCircuitBreaker(CircuitBreakerConfig{Name: "zep-search", MaxFailures: 5, Timeout: 30 * time.Second}),
        User:   NewCircuitBreaker(CircuitBreakerConfig{Name: "zep-user", MaxFailures: 5, Timeout: 30 * time.Second}),
    }
}
```

### 3.3. Exponential Backoff Retry

```go
// pkg/resilience/retry.go

package resilience

import (
    "context"
    "errors"
    "math"
    "math/rand"
    "time"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
    MinDelay   time.Duration  // default: 200ms
    MaxDelay   time.Duration  // default: 30s
    MaxRetries int            // default: 15
    Jitter     bool           // add random jitter to avoid thundering herd
}

var DefaultRetryConfig = RetryConfig{
    MinDelay:   200 * time.Millisecond,
    MaxDelay:   30 * time.Second,
    MaxRetries: 15,
    Jitter:     true,
}

// RetryWithBackoff executes fn with exponential backoff retry
// Retry only on ErrRetryable (caller wraps retriable errors)
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, fn func() error) error {
    var lastErr error

    for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
        if err := fn(); err == nil {
            return nil
        } else {
            lastErr = err
            // Don't retry on non-retriable errors
            if !errors.Is(err, ErrRetryable) {
                return err
            }
        }

        if attempt == cfg.MaxRetries {
            break
        }

        // Exponential backoff: min * 2^attempt, capped at max
        delay := time.Duration(float64(cfg.MinDelay) * math.Pow(2, float64(attempt)))
        if delay > cfg.MaxDelay { delay = cfg.MaxDelay }

        // Add jitter (±25% of delay)
        if cfg.Jitter {
            jitter := time.Duration(rand.Int63n(int64(delay) / 4))
            delay += jitter
        }

        select {
        case <-time.After(delay):
        case <-ctx.Done():
            return ctx.Err()
        }
    }

    return fmt.Errorf("max retries (%d) exceeded: %w", cfg.MaxRetries, lastErr)
}

var ErrRetryable = errors.New("retriable error")
var ErrCircuitOpen = errors.New("circuit breaker open")
```

### 3.4. PostgreSQL Advisory Locks (Shared Package)

```go
// pkg/metadata/advisory_lock.go

package metadata

import (
    "context"
    "crypto/sha256"
    "database/sql"
    "encoding/binary"
    "fmt"
)

// AdvisoryLockKey converts any string ID into a stable int64 for PostgreSQL advisory lock
// Uses SHA-256 first 8 bytes as int64 — collision probability: 1/(2^64)
func AdvisoryLockKey(id string) int64 {
    hash := sha256.Sum256([]byte(id))
    return int64(binary.BigEndian.Uint64(hash[:8]))
}

// WithAdvisoryLock acquires a session-level advisory lock, runs fn in a transaction,
// then releases the lock.
//
// Pattern used for:
// - Session metadata PATCH (CR-ZEP-001 §3.5)
// - Message metadata PATCH (CR-ZEP-002 §3.6)
// - User metadata PATCH (CR-ZEP-008)
//
// Lock granularity: session-level (released when connection closes)
func WithAdvisoryLock(ctx context.Context, db *sql.DB, lockID string, fn func(*sql.Tx) error) error {
    lockKey := AdvisoryLockKey(lockID)

    conn, err := db.Conn(ctx)
    if err != nil { return fmt.Errorf("get db connection: %w", err) }
    defer conn.Close()

    // Acquire session-level advisory lock
    if _, err = conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
        return fmt.Errorf("acquire advisory lock (id=%s, key=%d): %w", lockID, lockKey, err)
    }
    defer conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", lockKey)

    tx, err := conn.BeginTx(ctx, nil)
    if err != nil { return fmt.Errorf("begin transaction: %w", err) }

    if err := fn(tx); err != nil {
        tx.Rollback()
        return err
    }
    return tx.Commit()
}

// MergeJSONBMetadata performs a JSONB merge-patch under advisory lock
// SQL: UPDATE SET metadata = metadata || $patch::jsonb WHERE id = $id
func MergeJSONBMetadata(ctx context.Context, db *sql.DB, table, idColumn, id string, patch map[string]any) error {
    return WithAdvisoryLock(ctx, db, id, func(tx *sql.Tx) error {
        patchJSON, err := json.Marshal(patch)
        if err != nil { return err }

        query := fmt.Sprintf(`
            UPDATE %s
            SET metadata = metadata || $1::jsonb,
                updated_at = NOW()
            WHERE %s = $2
              AND deleted_at IS NULL
        `, table, idColumn)

        _, err = tx.ExecContext(ctx, query, patchJSON, id)
        return err
    })
}
```

### 3.5. 10-Layer Gateway Middleware Stack

```go
// pkg/middleware/stack.go

package middleware

import (
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/go-chi/cors"
    "github.com/riandyrn/otelchi"
)

// ZepMiddlewareConfig configures the middleware stack
type ZepMiddlewareConfig struct {
    ServiceName    string
    APIVersion     string        // e.g. "2.0"
    MaxPayloadMB   int           // default: 5
    RequestTimeout time.Duration // default: 30s
    AllowedOrigins []string      // CORS origins
}

// RegisterZepMiddleware applies the 10-layer middleware stack to a chi router
// Layer order is CRITICAL — each layer wraps the ones below it
func RegisterZepMiddleware(r chi.Router, cfg ZepMiddlewareConfig) {
    if cfg.MaxPayloadMB == 0 { cfg.MaxPayloadMB = 5 }
    if cfg.RequestTimeout == 0 { cfg.RequestTimeout = 30 * time.Second }
    if cfg.APIVersion == "" { cfg.APIVersion = "2.0" }

    r.Use(
        // Layer 1: CORS — must be first to handle preflight requests
        cors.Handler(cors.Options{
            AllowedOrigins:   cfg.AllowedOrigins,
            AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
            AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID"},
            AllowCredentials: true,
        }),

        // Layer 2: Structured request logging
        structuredRequestLogger(),

        // Layer 3: Health check bypass (no auth, no logging)
        middleware.Heartbeat("/healthz"),

        // Layer 4: Request size limiter (5MB default)
        middleware.RequestSize(int64(cfg.MaxPayloadMB)*1024*1024),

        // Layer 5: Request ID injection (UUID v4)
        middleware.RequestID,

        // Layer 6: Request timeout (30s default)
        middleware.Timeout(cfg.RequestTimeout),

        // Layer 7: Real IP extraction (X-Real-IP, X-Forwarded-For)
        middleware.RealIP,

        // Layer 8: URL path cleaning (remove double slashes, trailing slashes)
        middleware.CleanPath,

        // Layer 9: API version header injection
        middleware.SetHeader("X-API-Version", cfg.APIVersion),

        // Layer 10: OpenTelemetry tracing (must be last — wraps all layers)
        otelchi.Middleware(cfg.ServiceName, otelchi.WithChiRoutes(r)),
    )
}

// structuredRequestLogger returns a chi-compatible structured logger
func structuredRequestLogger() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

            next.ServeHTTP(ww, r)

            slog.Info("request",
                "method",     r.Method,
                "path",       r.URL.Path,
                "status",     ww.Status(),
                "duration_ms", time.Since(start).Milliseconds(),
                "bytes",      ww.BytesWritten(),
                "request_id", middleware.GetReqID(r.Context()),
                "remote_ip",  r.RemoteAddr,
            )
        })
    }
}
```

### 3.6. Anonymous Telemetry (Opt-Out)

```go
// pkg/telemetry/tracker.go

package telemetry

import (
    "context"
    "crypto/sha256"
    "fmt"
    "net/http"
    "time"
)

type TelemetryConfig struct {
    Disabled  bool
    ProjectID string    // will be hashed before sending
    Version   string
    Endpoint  string    // telemetry collection endpoint
}

type TelemetryEvent struct {
    Event     string    `json:"event"`
    ProjectID string    `json:"project_id"` // SHA-256 hash — no PII
    Version   string    `json:"version"`
    Timestamp time.Time `json:"ts"`
}

type Tracker struct {
    cfg    TelemetryConfig
    client *http.Client
}

func NewTracker(cfg TelemetryConfig) *Tracker {
    return &Tracker{cfg: cfg, client: &http.Client{Timeout: 5 * time.Second}}
}

// Track sends an anonymous telemetry event
// No-op when disabled
// No PII: project_id is SHA-256 hashed before sending
func (t *Tracker) Track(ctx context.Context, event string) {
    if t.cfg.Disabled { return }

    // Hash project ID to ensure no PII linkage
    h := sha256.Sum256([]byte(t.cfg.ProjectID))
    hashedProjectID := fmt.Sprintf("%x", h)[:16]

    evt := TelemetryEvent{
        Event:     event,
        ProjectID: hashedProjectID,
        Version:   t.cfg.Version,
        Timestamp: time.Now(),
    }

    // Fire-and-forget: don't block request processing
    go func() {
        body, _ := json.Marshal(evt)
        req, _ := http.NewRequestWithContext(
            context.Background(), "POST", t.cfg.Endpoint, bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        resp, err := t.client.Do(req)
        if err != nil { return }
        defer resp.Body.Close()
    }()
}

// Events tracked (anonymous, no PII):
// - "service_start"
// - "api_request"
// - "graph_extraction"
// - "search_query"
```

---

## 4. Config Integration

```yaml
# apps/memory/configs/config.yaml — thêm sections

# Circuit breaker settings
resilience:
  circuit_breaker:
    graph:
      max_failures: 3
      timeout_seconds: 60
    default:
      max_failures: 5
      timeout_seconds: 30

# Gateway middleware
gateway:
  max_payload_mb: 5
  request_timeout_seconds: 30
  api_version: "2.0"
  cors:
    allowed_origins: ["*"]

# Anonymous telemetry (opt-out)
telemetry:
  disabled: false    # set true to opt out
  endpoint: "https://telemetry.vnpmemory.io/events"
```

---

## 5. Usage Examples

### Circuit Breaker in gRPC Client

```go
// gateway/adapter/grpc_client.go

type ZepGatewayClient struct {
    breakers *resilience.ZepCircuitBreakers
    // ...
}

func (c *ZepGatewayClient) GetMemory(ctx context.Context, req *MemoryRequest) (*Memory, error) {
    result, err := c.breakers.Memory.Execute(func() (any, error) {
        return c.memoryClient.GetMemory(ctx, req)
    })
    if errors.Is(err, resilience.ErrCircuitOpen) {
        // Fast fail — don't wait for timeout
        return nil, ErrMemoryServiceUnavailable
    }
    return result.(*Memory), err
}
```

### Advisory Lock in Thread Service

```go
// services/zep-thread/internal/infra/postgres/session_repo.go

// Uses shared pkg/metadata/advisory_lock.go
func (r *SessionRepo) MergePatchMetadata(ctx context.Context, sessionID string, patch map[string]any) error {
    return metadata.MergeJSONBMetadata(ctx, r.db, "sessions", "session_id", sessionID, patch)
}
```

### Retry in Graph Service

```go
// services/zep-graph/internal/adapter/graphiti/client.go

func (c *GraphitiClient) PutMemory(ctx context.Context, req PutMemoryRequest) (*GraphitiResponse, error) {
    var result *GraphitiResponse

    err := resilience.RetryWithBackoff(ctx, resilience.DefaultRetryConfig, func() error {
        resp, err := c.doRequest(ctx, req)
        if err != nil {
            if isRateLimitError(err) { return fmt.Errorf("%w: %v", resilience.ErrRetryable, err) }
            return err  // non-retriable
        }
        result = resp
        return nil
    })

    return result, err
}
```

---

## 6. Request Constraints Summary

| Constraint | Value | Implementation |
|-----------|-------|---------------|
| Max payload | 5MB | `middleware.RequestSize(5*1024*1024)` — Layer 4 |
| Request timeout | 30s | `middleware.Timeout(30s)` — Layer 6 |
| Circuit breaker threshold | 5 failures (graph: 3) | `gobreaker.CircuitBreaker` |
| Circuit breaker timeout | 30s (graph: 60s) | `gobreaker.Settings.Timeout` |
| Advisory lock collision | 1/(2^64) | SHA-256 first 8 bytes |
| Retry max delay | 30s | `MaxDelay: 30s` |
| Retry max attempts | 15 | `MaxRetries: 15` |
| Retry initial delay | 200ms | `MinDelay: 200ms` |

---

## 7. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | `pkg/resilience/circuit_breaker.go` (sony/gobreaker) | 1 ngày |
| **P2** | `pkg/resilience/retry.go` (exponential backoff + jitter) | 1 ngày |
| **P3** | `pkg/metadata/advisory_lock.go` (shared package) | 1 ngày |
| **P4** | `pkg/middleware/stack.go` (10-layer chi stack) | 1.5 ngày |
| **P5** | `pkg/telemetry/tracker.go` (opt-out anonymous) | 1 ngày |
| **P6** | Integrate circuit breaker vào tất cả gRPC clients | 1 ngày |
| **P7** | Replace per-service advisory lock với shared package | 0.5 ngày |
| **P8** | Config YAML integration + tests | 1 ngày |

**Tổng:** ~8 ngày (Wave 1 — foundation, ưu tiên cao nhất)

---

## 8. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| Graph service down → circuit breaker opens sau 5 failures | MaxFailures:5 → ErrCircuitOpen |
| Concurrent PATCH từ 10 goroutines → 1 tại một thời điểm | pg_advisory_lock() + transaction |
| Request > 5MB → 413 | middleware.RequestSize(5MB) — Layer 4 |
| Request > 30s → 504 Gateway Timeout | middleware.Timeout(30s) — Layer 6 |
| Mỗi request log có đủ fields | structuredRequestLogger(): method, path, status, duration_ms, request_id |
| Telemetry disabled khi config=true | `if t.cfg.Disabled { return }` no-op |
