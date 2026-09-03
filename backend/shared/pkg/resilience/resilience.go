package resilience

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrCircuitOpen = errors.New("resilience: circuit breaker is open")

// ─── CircuitBreaker ────────────────────────────────────────────────────────────

// CircuitBreaker wraps sony/gobreaker with a simplified interface.
type CircuitBreaker struct {
	cb *gobreaker.CircuitBreaker
}

// NewCircuitBreaker creates a circuit breaker that trips when >50% of requests
// fail (minimum 10 requests) and resets after 30 seconds.
func NewCircuitBreaker(name string) *CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,
		Interval:    1 * time.Minute,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < 10 {
				return false
			}
			return float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
		},
	}
	return &CircuitBreaker{cb: gobreaker.NewCircuitBreaker(settings)}
}

// Execute runs fn through the circuit breaker. Returns ErrCircuitOpen when tripped.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	_, err := cb.cb.Execute(func() (any, error) { return nil, fn() })
	if errors.Is(err, gobreaker.ErrOpenState) {
		return ErrCircuitOpen
	}
	return err
}

// State returns the current circuit state (Closed, HalfOpen, Open).
func (cb *CircuitBreaker) State() gobreaker.State { return cb.cb.State() }

// ─── Retry ─────────────────────────────────────────────────────────────────────

// RetryConfig configures exponential backoff retry behavior.
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	IsRetryable  func(error) bool
}

// DefaultRetryConfig retries on transient gRPC errors with exponential backoff.
var DefaultRetryConfig = RetryConfig{
	MaxAttempts:  3,
	InitialDelay: 100 * time.Millisecond,
	MaxDelay:     10 * time.Second,
	Multiplier:   2.0,
	IsRetryable: func(err error) bool {
		code := status.Code(err)
		return code == codes.Unavailable ||
			code == codes.DeadlineExceeded ||
			code == codes.ResourceExhausted
	},
}

// Retry executes fn up to cfg.MaxAttempts times with exponential backoff.
func Retry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	delay := cfg.InitialDelay
	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(delay):
				delay = time.Duration(float64(delay) * cfg.Multiplier)
				if delay > cfg.MaxDelay {
					delay = cfg.MaxDelay
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if cfg.IsRetryable != nil && !cfg.IsRetryable(lastErr) {
			return lastErr
		}
	}
	return fmt.Errorf("after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// ─── Bulkhead ──────────────────────────────────────────────────────────────────

// Bulkhead limits concurrent execution using a semaphore.
type Bulkhead struct {
	sem chan struct{}
	max int
}

// NewBulkhead creates a Bulkhead allowing at most maxConcurrent simultaneous calls.
func NewBulkhead(maxConcurrent int) *Bulkhead {
	return &Bulkhead{sem: make(chan struct{}, maxConcurrent), max: maxConcurrent}
}

// Execute runs fn if a slot is available, blocking until ctx is done otherwise.
func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
	select {
	case b.sem <- struct{}{}:
		defer func() { <-b.sem }()
		return fn()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Active returns the number of currently running goroutines.
func (b *Bulkhead) Active() int { return len(b.sem) }

// Max returns the configured maximum concurrency.
func (b *Bulkhead) Max() int { return b.max }
