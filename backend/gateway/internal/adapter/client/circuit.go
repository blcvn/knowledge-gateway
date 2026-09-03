package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/sony/gobreaker/v2"

	"github.com/vnp-community/vnp-memory/gateway/internal/domain"
)

// CircuitRegistry wraps GRPCRegistry with per-service circuit breakers.
// Implements port.ServiceRegistry.
type CircuitRegistry struct {
	inner    *GRPCRegistry
	breakers map[string]*gobreaker.CircuitBreaker[[]byte]
	logger   *slog.Logger
}

// CircuitConfig holds circuit breaker settings.
type CircuitConfig struct {
	MaxFailures int
	Timeout     time.Duration
	MaxRequests int
}

// NewCircuitRegistry wraps an existing GRPCRegistry with per-service circuit breakers.
func NewCircuitRegistry(inner *GRPCRegistry, cfg CircuitConfig, logger *slog.Logger) *CircuitRegistry {
	cr := &CircuitRegistry{
		inner:    inner,
		breakers: make(map[string]*gobreaker.CircuitBreaker[[]byte]),
		logger:   logger,
	}

	for _, svc := range inner.ConnectedServices() {
		svcName := svc // capture for closure
		cr.breakers[svc] = gobreaker.NewCircuitBreaker[[]byte](gobreaker.Settings{
			Name:        svcName,
			MaxRequests: uint32(cfg.MaxRequests), // requests in half-open
			Interval:    0,                       // no cyclic clearing
			Timeout:     cfg.Timeout,             // duration before half-open
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= uint32(cfg.MaxFailures)
			},
			OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
				logger.Warn("circuit breaker state change",
					"service", name,
					"from", from.String(),
					"to", to.String(),
				)
			},
		})
		logger.Debug("circuit breaker created", "service", svc,
			"max_failures", cfg.MaxFailures,
			"timeout", cfg.Timeout,
			"max_requests", cfg.MaxRequests,
		)
	}

	logger.Info("circuit registry initialized", "breakers", len(cr.breakers))
	return cr
}

// Resolve delegates to the inner registry.
func (cr *CircuitRegistry) Resolve(service string) (*domain.RouteTarget, error) {
	return cr.inner.Resolve(service)
}

// Forward wraps the inner Forward call with a circuit breaker.
func (cr *CircuitRegistry) Forward(ctx context.Context, target *domain.RouteTarget, req []byte) ([]byte, error) {
	return cr.ForwardWithContext(ctx, target, &domain.ForwardRequest{Body: req})
}

// ForwardWithContext wraps the inner ForwardWithContext call with a circuit breaker.
func (cr *CircuitRegistry) ForwardWithContext(ctx context.Context, target *domain.RouteTarget, req *domain.ForwardRequest) ([]byte, error) {
	cb, ok := cr.breakers[target.Service]
	if !ok {
		// No circuit breaker for this service — pass through
		return cr.inner.ForwardWithContext(ctx, target, req)
	}

	result, err := cb.Execute(func() ([]byte, error) {
		return cr.inner.ForwardWithContext(ctx, target, req)
	})

	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			cr.logger.Warn("circuit open, rejecting request",
				"service", target.Service,
				"state", cb.State().String(),
			)
			return nil, domain.ErrCircuitOpen
		}
		return nil, err
	}

	return result, nil
}

// HealthCheck delegates to the inner registry.
func (cr *CircuitRegistry) HealthCheck(service string) (bool, error) {
	return cr.inner.HealthCheck(service)
}

// CircuitState returns the circuit breaker state for a given service.
func (cr *CircuitRegistry) CircuitState(service string) string {
	cb, ok := cr.breakers[service]
	if !ok {
		return "unknown"
	}
	return cb.State().String()
}

// AllCircuitStates returns the circuit breaker states for all services.
func (cr *CircuitRegistry) AllCircuitStates() map[string]string {
	states := make(map[string]string, len(cr.breakers))
	for svc, cb := range cr.breakers {
		states[svc] = cb.State().String()
	}
	return states
}

// CircuitCounts returns the current counts for a circuit breaker.
func (cr *CircuitRegistry) CircuitCounts(service string) string {
	cb, ok := cr.breakers[service]
	if !ok {
		return "unknown"
	}
	counts := cb.Counts()
	return fmt.Sprintf("requests=%d success=%d failure=%d consecutive_success=%d consecutive_failure=%d",
		counts.Requests, counts.TotalSuccesses, counts.TotalFailures,
		counts.ConsecutiveSuccesses, counts.ConsecutiveFailures)
}
