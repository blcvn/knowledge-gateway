// Package port — Console-specific output port interfaces (T08-T09).
package port

import (
	"context"

	"github.com/vnp-community/vnp-memory/gateway/domain"
)

// AuditStore provides audit log persistence (T08).
type AuditStore interface {
	// Insert records a new audit entry.
	Insert(ctx context.Context, entry *domain.AuditEntry) error
	// Search queries audit logs with filters. Returns entries and total count.
	Search(ctx context.Context, filter *domain.AuditFilter) ([]*domain.AuditEntry, int, error)
}

// PolicyStore provides governance policy persistence (T09).
type PolicyStore interface {
	// List returns all policies for a tenant.
	List(ctx context.Context, tenantID string) ([]*domain.Policy, error)
	// Get returns a single policy by ID.
	Get(ctx context.Context, id string) (*domain.Policy, error)
	// Create inserts a new policy.
	Create(ctx context.Context, p *domain.Policy) error
	// Update modifies an existing policy.
	Update(ctx context.Context, p *domain.Policy) error
}
