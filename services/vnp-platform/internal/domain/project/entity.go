// Package project defines domain entities for project/space management.
//
// Absorbed from: sm-project
package project

import (
	"time"

	"github.com/google/uuid"
)

// Space represents a document organization container (from sm-project).
type Space struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	Name      string         `json:"name"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// ContainerTag represents a label for organizing resources.
type ContainerTag struct {
	ID       uuid.UUID `json:"id"`
	SpaceID  uuid.UUID `json:"space_id"`
	Name     string    `json:"name"`
	Color    string    `json:"color"`
}

// Membership tracks user access to spaces.
type Membership struct {
	ID       uuid.UUID `json:"id"`
	SpaceID  uuid.UUID `json:"space_id"`
	UserID   uuid.UUID `json:"user_id"`
	Role     string    `json:"role"` // owner, editor, viewer
	JoinedAt time.Time `json:"joined_at"`
}
