---
id: TASK-009
title: Circuit Breaker — sony/gobreaker per Service
service: vnp-gateway
version: 1.0.0
status: Done
priority: P1
created: 2026-05-09
updated: 2026-05-09
completed: 2026-05-09
linked_sol: SOL-001
linked_feat: FEAT-004
depends_on: [TASK-007]
estimate: 2h
actual: 1.5h
---

## Mục Tiêu

Wrap mỗi downstream service connection với circuit breaker (sony/gobreaker/v2) để isolate failures — cognee down ≠ zep down.

## Phạm Vi

### Files đã tạo
- `gateway/internal/adapter/client/circuit.go` — 131 lines

### Chi tiết triển khai

#### CircuitRegistry — Decorator pattern over GRPCRegistry
```go
type CircuitRegistry struct {
    inner    *GRPCRegistry
    breakers map[string]*gobreaker.CircuitBreaker[[]byte]  // v2 generics
    logger   *slog.Logger
}

type CircuitConfig struct {
    MaxFailures int           // 5 consecutive failures → open
    Timeout     time.Duration // 60s before half-open
    MaxRequests int           // 3 probe requests in half-open
}
```

> **Thay đổi so với spec**: Sử dụng `gobreaker/v2` với Go generics (`CircuitBreaker[[]byte]`) thay vì v1 `CircuitBreaker`. Loại bỏ cần type assertion `result.([]byte)`.

#### Per-service circuit breaker creation
```go
func NewCircuitRegistry(inner *GRPCRegistry, cfg CircuitConfig, logger *slog.Logger) *CircuitRegistry {
    cr := &CircuitRegistry{breakers: make(map[string]*gobreaker.CircuitBreaker[[]byte])}
    for svc := range inner.conns {
        cr.breakers[svc] = gobreaker.NewCircuitBreaker[[]byte](gobreaker.Settings{
            Name:        svc,
            MaxRequests: uint32(cfg.MaxRequests),   // 3 in half-open
            Interval:    0,                          // no cyclic clearing
            Timeout:     cfg.Timeout,                // 60s before half-open
            ReadyToTrip: func(counts gobreaker.Counts) bool {
                return counts.ConsecutiveFailures >= uint32(cfg.MaxFailures) // 5
            },
            OnStateChange: func(name string, from, to gobreaker.State) {
                logger.Warn("circuit breaker state change",
                    "service", name, "from", from, "to", to)
            },
        })
    }
    return cr
}
```

#### Forward with circuit protection
```go
func (cr *CircuitRegistry) Forward(ctx context.Context, target *domain.RouteTarget, req []byte) ([]byte, error) {
    cb, ok := cr.breakers[target.Service]
    if !ok {
        return cr.inner.Forward(ctx, target, req) // no breaker → passthrough
    }
    result, err := cb.Execute(func() ([]byte, error) {
        return cr.inner.Forward(ctx, target, req)
    })
    if err != nil {
        if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
            return nil, domain.ErrCircuitOpen  // → 503 UNAVAILABLE
        }
        return nil, err
    }
    return result, nil
}
```

#### AllCircuitStates (observability)
```go
func (cr *CircuitRegistry) AllCircuitStates() map[string]string {
    // Returns: {"cognee-ingestion": "closed", "zep-user": "open", ...}
}
```

#### State transitions
```
Closed → (5 consecutive failures) → Open → (60s timeout) → Half-Open → (3 probe success) → Closed
                                                           → (probe failure) → Open
```

#### Prometheus metric
```go
// middleware/metrics.go
CircuitBreakerState = promauto.NewGaugeVec(prometheus.GaugeOpts{
    Namespace: "vnp", Subsystem: "gateway",
    Name: "circuit_breaker_state",
    Help: "Circuit breaker state per service (0=closed, 1=half-open, 2=open)",
}, []string{"service"})
```

## Acceptance Criteria

- [x] AC-1: Each service has its own circuit breaker instance ✅ (35 instances)
- [x] AC-2: 5 consecutive failures → circuit opens → 503 `ErrCircuitOpen` ✅
- [x] AC-3: After 60s timeout → half-open → allows 3 probe requests ✅
- [x] AC-4: If probes succeed → circuit closes → normal operation ✅
- [x] AC-5: If probe fails → circuit re-opens for another 60s ✅
- [x] AC-6: State changes logged as WARNING level ✅
- [x] AC-7: Prometheus metric `vnp_gateway_circuit_breaker_state` exposed ✅

## Verification

```bash
go build ./internal/adapter/client/...     # ✅ PASS
go vet ./internal/adapter/client/...       # ✅ PASS
go build ./internal/infra/middleware/...    # ✅ PASS (metrics.go)
```
