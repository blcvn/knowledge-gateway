// Package port defines the output interfaces for the gateway usecase layer.
// Output ports are driven BY usecases (outbound) — usecases call these methods,
// infrastructure provides the concrete implementations.
package port

import (
	"context"

	"github.com/vnp-community/vnp-memory/gateway/domain"
)

// ServiceRegistry manages downstream gRPC connections and request forwarding.
type ServiceRegistry interface {
	// Resolve looks up the RouteTarget for a named service.
	Resolve(service string) (*domain.RouteTarget, error)
	// Forward sends a request to the target service and returns the response body.
	Forward(ctx context.Context, target *domain.RouteTarget, req []byte) ([]byte, error)
	// HealthCheck returns the health status of a downstream service.
	HealthCheck(service string) (bool, error)
}

// TenantStore provides tenant configuration lookup.
type TenantStore interface {
	// GetTenant retrieves tenant context by ID.
	GetTenant(ctx context.Context, id string) (*domain.TenantContext, error)
}

// KeyStore resolves API keys to authenticated identity.
type KeyStore interface {
	// ResolveAPIKey looks up an API key hash and returns the associated AuthContext.
	ResolveAPIKey(ctx context.Context, keyHash string) (*domain.AuthContext, error)
}

// EventPublisher publishes domain events to the message bus (NATS JetStream).
type EventPublisher interface {
	// Publish sends an event to the specified subject.
	Publish(ctx context.Context, subject string, event any) error
}

// RateLimitStore provides rate limit state management (Redis backend).
type RateLimitStore interface {
	// CheckAndIncrement atomically checks the rate limit and increments the counter.
	// Returns whether the request is allowed and the remaining count.
	CheckAndIncrement(ctx context.Context, key string, limit int, windowSec int) (allowed bool, remaining int, err error)
}
