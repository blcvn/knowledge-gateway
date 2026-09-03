package thread

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Thread represents a continuous conversation session.
type Thread struct {
	UUID        uuid.UUID
	SessionID   string
	UserUUID    *uuid.UUID // Optional association
	ProjectUUID uuid.UUID
	Metadata    map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   time.Time
	EndedAt     *time.Time
	DeletedAt   *time.Time
}

// Repository handles thread persistence.
type Repository interface {
	Upsert(ctx context.Context, thread *Thread) error
	GetBySessionID(ctx context.Context, projectUUID uuid.UUID, sessionID string) (*Thread, error)
	SoftDelete(ctx context.Context, uuid uuid.UUID) error
}
