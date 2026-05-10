// Package knowledge defines domain entities for the Graphiti knowledge extraction.
package knowledge

import (
	"time"

	"github.com/google/uuid"
)

// Entity represents a node in the temporal knowledge graph.
type Entity struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description,omitempty"`
	ValidAt     time.Time `json:"valid_at"`
	InvalidAt   *time.Time `json:"invalid_at,omitempty"` // Bi-temporal
	CreatedAt   time.Time `json:"created_at"`
}

// Edge represents a relationship in the temporal knowledge graph.
type Edge struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	SourceID    uuid.UUID `json:"source_id"`
	TargetID    uuid.UUID `json:"target_id"`
	Relation    string    `json:"relation"`
	Weight      float64   `json:"weight"`
	ValidAt     time.Time `json:"valid_at"`
	InvalidAt   *time.Time `json:"invalid_at,omitempty"` // Bi-temporal
	CreatedAt   time.Time `json:"created_at"`
}

// Community represents a detected graph community.
type Community struct {
	ID       uuid.UUID   `json:"id"`
	TenantID uuid.UUID   `json:"tenant_id"`
	Name     string      `json:"name"`
	Summary  string      `json:"summary"`
	Members  []uuid.UUID `json:"members"` // Entity IDs
}
