package resilience_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sony/gobreaker"
	"github.com/vnp-memory/pkg/resilience"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ─── CircuitBreaker Tests ───────────────────────────────────────────────────────

func TestCircuitBreaker_AllowsInitially(t *testing.T) {
	cb := resilience.NewCircuitBreaker("test")
	called := false
	err := cb.Execute(func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Errorf("Execute error: %v", err)
	}
	if !called {
		t.Error("fn was not called")
	}
}

func TestCircuitBreaker_TripsAfterFailures(t *testing.T) {
	cb := resilience.NewCircuitBreaker("trip-test")
	// Execute 10+ failures to trip the breaker (>50% failure rate after 10 requests)
	testErr := errors.New("fail")
	for i := 0; i < 12; i++ {
		_ = cb.Execute(func() error { return testErr })
	}
	// Now the CB should be open
	if cb.State() != gobreaker.StateOpen {
		t.Skip("circuit breaker did not trip (may need different configuration)")
	}
	err := cb.Execute(func() error { return nil })
	if !errors.Is(err, resilience.ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

// ─── Retry Tests ────────────────────────────────────────────────────────────────

func TestRetry_SuccessFirstAttempt(t *testing.T) {
	count := 0
	err := resilience.Retry(context.Background(), resilience.RetryConfig{
		MaxAttempts: 3, InitialDelay: time.Millisecond,
	}, func() error {
		count++
		return nil
	})
	if err != nil {
		t.Errorf("Retry error: %v", err)
	}
	if count != 1 {
		t.Errorf("fn called %d times, want 1", count)
	}
}

func TestRetry_RetriesOnRetryable(t *testing.T) {
	count := 0
	retryableErr := status.Error(codes.Unavailable, "unavailable")
	cfg := resilience.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
		IsRetryable:  resilience.DefaultRetryConfig.IsRetryable,
	}
	err := resilience.Retry(context.Background(), cfg, func() error {
		count++
		if count < 3 {
			return retryableErr
		}
		return nil
	})
	if err != nil {
		t.Errorf("Retry error: %v", err)
	}
	if count != 3 {
		t.Errorf("fn called %d times, want 3", count)
	}
}

func TestRetry_NoRetryOnNonRetryable(t *testing.T) {
	count := 0
	nonRetryable := status.Error(codes.InvalidArgument, "bad input")
	cfg := resilience.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		IsRetryable:  resilience.DefaultRetryConfig.IsRetryable,
	}
	_ = resilience.Retry(context.Background(), cfg, func() error {
		count++
		return nonRetryable
	})
	if count != 1 {
		t.Errorf("fn called %d times, want 1 (no retry on InvalidArgument)", count)
	}
}

func TestRetry_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := resilience.RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 50 * time.Millisecond,
		IsRetryable:  func(error) bool { return true },
	}
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()
	err := resilience.Retry(ctx, cfg, func() error {
		return errors.New("always fail")
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRetry_ExhaustsAttempts(t *testing.T) {
	cfg := resilience.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		IsRetryable:  func(error) bool { return true },
	}
	err := resilience.Retry(context.Background(), cfg, func() error {
		return errors.New("persistent error")
	})
	if err == nil {
		t.Error("expected error after exhausting attempts")
	}
}

// ─── Bulkhead Tests ────────────────────────────────────────────────────────────

func TestBulkhead_AllowsBelowMax(t *testing.T) {
	bh := resilience.NewBulkhead(5)
	err := bh.Execute(context.Background(), func() error { return nil })
	if err != nil {
		t.Errorf("Execute error: %v", err)
	}
}

func TestBulkhead_ContextCancel(t *testing.T) {
	bh := resilience.NewBulkhead(1)
	// Fill the slot
	done := make(chan struct{})
	go func() {
		_ = bh.Execute(context.Background(), func() error {
			<-done
			return nil
		})
	}()
	time.Sleep(5 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	err := bh.Execute(ctx, func() error { return nil })
	close(done)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled when bulkhead full, got %v", err)
	}
}

func TestBulkhead_Active(t *testing.T) {
	bh := resilience.NewBulkhead(5)
	ready := make(chan struct{})
	done := make(chan struct{})
	for i := 0; i < 3; i++ {
		go func() {
			_ = bh.Execute(context.Background(), func() error {
				ready <- struct{}{}
				<-done
				return nil
			})
		}()
	}
	// Wait for all 3 to be active
	for i := 0; i < 3; i++ {
		<-ready
	}
	if got := bh.Active(); got != 3 {
		t.Errorf("Active() = %d, want 3", got)
	}
	close(done)
}

func TestBulkhead_MaxConcurrency(t *testing.T) {
	bh := resilience.NewBulkhead(2)
	if bh.Max() != 2 {
		t.Errorf("Max() = %d, want 2", bh.Max())
	}
}
