package port

import (
	"context"

	"vnp-memory/services/ov-crypto/internal/domain"
)

// EventPublisherPort defines the outbound interface for publishing domain events.
type EventPublisherPort interface {
	PublishKeyRotated(ctx context.Context, event domain.KeyRotated) error
}
