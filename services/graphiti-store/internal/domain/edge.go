package domain

import (
	"fmt"
	"time"
)

// EntityEdge represents a bi-temporal relationship (fact triple) between two entities.
//
// Bi-temporal model:
//   - valid_at:   when the fact became true in the real world (required)
//   - invalid_at: when the fact stopped being true (nil = still valid)
//   - expired_at: when the edge was superseded by a newer extraction
//   - created_at: system timestamp when the edge was ingested
type EntityEdge struct {
	UUID          string            `json:"uuid"`
	SourceNodeID  string            `json:"source_node_id"`
	TargetNodeID  string            `json:"target_node_id"`
	Name          string            `json:"name"`
	GroupID       string            `json:"group_id"`
	Fact          string            `json:"fact"`
	FactEmbedding []float32         `json:"fact_embedding,omitempty"`
	ValidAt       time.Time         `json:"valid_at"`
	InvalidAt     *time.Time        `json:"invalid_at,omitempty"`
	ExpiredAt     *time.Time        `json:"expired_at,omitempty"`
	Attributes    map[string]string `json:"attributes,omitempty"`
	EpisodeID     string            `json:"episode_id"`
	CreatedAt     time.Time         `json:"created_at"`
}

// Validate checks bi-temporal constraints and required fields.
func (e *EntityEdge) Validate() error {
	if e.UUID == "" {
		return ErrMissingUUID
	}
	if e.SourceNodeID == "" || e.TargetNodeID == "" {
		return ErrMissingNodeID
	}
	if e.GroupID == "" {
		return ErrMissingGroupID
	}
	if e.Fact == "" {
		return ErrEmptyFact
	}
	if e.ValidAt.IsZero() {
		return ErrMissingValidAt
	}
	if e.InvalidAt != nil && e.InvalidAt.Before(e.ValidAt) {
		return &ErrInvalidTemporalRange{
			Field:    "invalid_at",
			Value:    *e.InvalidAt,
			ValidAt:  e.ValidAt,
			Reason:   "invalid_at must be after valid_at",
		}
	}
	if e.ExpiredAt != nil && !e.CreatedAt.IsZero() && e.ExpiredAt.Before(e.CreatedAt) {
		return &ErrInvalidTemporalRange{
			Field:    "expired_at",
			Value:    *e.ExpiredAt,
			ValidAt:  e.CreatedAt,
			Reason:   "expired_at must be after created_at",
		}
	}
	return nil
}

// IsCurrentlyValid returns true if the edge has no invalid_at or invalid_at is in the future.
func (e *EntityEdge) IsCurrentlyValid(now time.Time) bool {
	return e.InvalidAt == nil || e.InvalidAt.After(now)
}

// IsExpired returns true if the edge has been superseded by a newer extraction.
func (e *EntityEdge) IsExpired() bool {
	return e.ExpiredAt != nil
}

// OverlapsWindow returns true if the edge's validity window intersects [from, to].
func (e *EntityEdge) OverlapsWindow(from, to time.Time) bool {
	// Edge valid from valid_at to invalid_at (or forever if nil)
	if e.ValidAt.After(to) {
		return false
	}
	if e.InvalidAt != nil && e.InvalidAt.Before(from) {
		return false
	}
	return true
}

// String returns a human-readable representation.
func (e *EntityEdge) String() string {
	return fmt.Sprintf("Edge{uuid=%s, %s -[%s]-> %s, valid=%s}",
		e.UUID, e.SourceNodeID, e.Name, e.TargetNodeID, e.ValidAt.Format(time.RFC3339))
}

// EpisodicEdge represents a MENTIONS relationship between an episode and an entity.
type EpisodicEdge struct {
	UUID       string    `json:"uuid"`
	EpisodeID  string    `json:"episode_id"`  // Source: EpisodicNode
	EntityID   string    `json:"entity_id"`   // Target: EntityNode
	GroupID    string    `json:"group_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// Validate checks required fields.
func (e *EpisodicEdge) Validate() error {
	if e.UUID == "" {
		return ErrMissingUUID
	}
	if e.EpisodeID == "" || e.EntityID == "" {
		return ErrMissingNodeID
	}
	if e.GroupID == "" {
		return ErrMissingGroupID
	}
	return nil
}
