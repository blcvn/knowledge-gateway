package project

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Project represents an isolated tenant environment within Zep.
type Project struct {
	UUID        uuid.UUID
	Name        string
	Description string
	APIKeyHash  string // Argon2id hash for security
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// Repository handles persistence logic for projects.
type Repository interface {
	Create(ctx context.Context, proj *Project) error
	GetByUUID(ctx context.Context, id uuid.UUID) (*Project, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// EventPublisher handles NATS broadcasting of lifecycle events.
type EventPublisher interface {
	PublishProjectCreated(ctx context.Context, proj *Project) error
	PublishProjectDeleted(ctx context.Context, id uuid.UUID) error
}
