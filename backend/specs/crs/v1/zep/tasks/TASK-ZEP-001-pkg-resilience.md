# TASK-ZEP-001 — pkg/resilience: Circuit Breaker & Retry

**Task ID:** TASK-ZEP-001  
**Wave:** 1 (Foundation)  
**Solution:** [SOL-ZEP-009](../solutions/SOL-ZEP-009-Resilience-Observability.md)  
**Ước tính:** 3h  
**Priority:** Critical — foundation, mọi service khác depend vào

---

## Mục tiêu

Tạo `pkg/resilience/` với 2 files:
1. `circuit_breaker.go` — wrapper cho `sony/gobreaker` với per-service config
2. `retry.go` — Exponential backoff retry (200ms → 30s, max 15 retries, jitter)

---

## Input Context

- **Dep:** `github.com/sony/gobreaker` (thêm vào `go.mod`)
- **Target path:** `pkg/resilience/`
- **Used by:** tất cả Zep gRPC clients trong gateway + services

---

## Công việc cụ thể

### 1. Tạo `pkg/resilience/circuit_breaker.go`

```go
package resilience

import (
    "fmt"
    "log/slog"
    "time"
    "github.com/sony/gobreaker"
)

// CircuitBreakerConfig định nghĩa cấu hình cho một circuit breaker
type CircuitBreakerConfig struct {
    Name          string
    MaxFailures   uint32        // consecutive failures trước khi mở (default: 5)
    Timeout       time.Duration // thời gian ở trạng thái open (default: 30s)
    OnStateChange func(name string, from, to gobreaker.State)
}

// CircuitBreaker wraps sony/gobreaker với logging
type CircuitBreaker struct {
    breaker *gobreaker.CircuitBreaker
    name    string
}

// ErrCircuitOpen là sentinel error khi circuit đang open
var ErrCircuitOpen = fmt.Errorf("circuit breaker open")

// NewCircuitBreaker tạo một circuit breaker mới
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker { ... }

// Execute chạy fn qua circuit breaker
// Trả về ErrCircuitOpen nếu breaker đang open
func (cb *CircuitBreaker) Execute(fn func() (any, error)) (any, error) { ... }

// ZepCircuitBreakers chứa các breakers cho từng Zep service
type ZepCircuitBreakers struct {
    Thread *CircuitBreaker  // MaxFailures:5, Timeout:30s
    Memory *CircuitBreaker  // MaxFailures:5, Timeout:30s
    Graph  *CircuitBreaker  // MaxFailures:3, Timeout:60s (sensitive — extraction slow)
    Search *CircuitBreaker  // MaxFailures:5, Timeout:30s
    User   *CircuitBreaker  // MaxFailures:5, Timeout:30s
}

// NewZepCircuitBreakers khởi tạo tất cả circuit breakers cho Zep
func NewZepCircuitBreakers() *ZepCircuitBreakers { ... }
```

### 2. Tạo `pkg/resilience/retry.go`

```go
package resilience

import (
    "context"
    "errors"
    "fmt"
    "math"
    "math/rand"
    "time"
)

// RetryConfig định nghĩa hành vi retry
type RetryConfig struct {
    MinDelay   time.Duration // default: 200ms
    MaxDelay   time.Duration // default: 30s
    MaxRetries int           // default: 15
    Jitter     bool          // ±25% để tránh thundering herd
}

// DefaultRetryConfig cấu hình mặc định cho advisory lock và external calls
var DefaultRetryConfig = RetryConfig{
    MinDelay:   200 * time.Millisecond,
    MaxDelay:   30 * time.Second,
    MaxRetries: 15,
    Jitter:     true,
}

// ErrRetryable là wrapper để đánh dấu lỗi có thể retry
var ErrRetryable = errors.New("retriable error")

// RetryWithBackoff thực thi fn với exponential backoff
// Chỉ retry nếu lỗi wrap ErrRetryable
// delay = MinDelay * 2^attempt (capped at MaxDelay) ± jitter
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, fn func() error) error { ... }
```

### 3. Tạo `pkg/resilience/resilience_test.go`

Test cases:
- `TestCircuitBreaker_OpensAfterMaxFailures`: 5 failures → state = Open
- `TestCircuitBreaker_FastFailOnOpen`: request khi Open → trả về ErrCircuitOpen ngay
- `TestRetryWithBackoff_SuccessOnThirdAttempt`: fail 2 lần rồi succeed
- `TestRetryWithBackoff_NonRetryableError`: non-retryable → không retry, return ngay
- `TestRetryWithBackoff_ContextCancellation`: context cancel → stop retry
- `TestRetryWithBackoff_ExceedsMaxRetries`: all retries fail → return error

---

## Acceptance Criteria

- [ ] `go build ./pkg/resilience/...` không có lỗi
- [ ] `go test ./pkg/resilience/...` 100% pass
- [ ] Circuit breaker opens sau đúng MaxFailures consecutive failures
- [ ] `ErrCircuitOpen` được trả về khi breaker đang open (không block)
- [ ] Retry: attempt 0 delay=200ms, attempt 1=400ms, attempt 2=800ms (capped at 30s)
- [ ] Jitter: delay thực tế ≈ base ± 25%
- [ ] Retry KHÔNG chạy khi error không wrap ErrRetryable
- [ ] Context cancel dừng retry loop ngay lập tức

---

## Files tạo ra

```
pkg/resilience/
├── circuit_breaker.go
├── retry.go
└── resilience_test.go
```

## Sau khi hoàn thành

Chạy: `go build ./pkg/resilience/... && go test ./pkg/resilience/...`
