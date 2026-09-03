package user

import (
	"time"

	"github.com/google/uuid"
)

// User represents a top-level entity possessing sessions and threads.
type User struct {
	UUID        uuid.UUID
	UserID      string    // External identifier
	ProjectUUID uuid.UUID // Tenant Isolation
	Metadata    map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// Repository handles user persistence.
type Repository interface {
	Upsert(ctx context.Context, user *User) error
	GetByID(ctx context.Context, projectUUID uuid.UUID, userID string) (*User, error)
	SoftDelete(ctx context.Context, uuid uuid.UUID) error
}
