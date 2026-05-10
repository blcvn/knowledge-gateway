// Package user defines domain entities for the Zep user sub-domain.
package user

import (
	"time"

	"github.com/google/uuid"
)

// User represents a Zep user with JSONB metadata.
type User struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	UserID    string         `json:"user_id"` // External user identifier
	Email     string         `json:"email,omitempty"`
	FirstName string         `json:"first_name,omitempty"`
	LastName  string         `json:"last_name,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"` // JSONB merge-patch
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
