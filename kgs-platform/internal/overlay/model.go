package overlay

import "time"

type Status string

const (
	StatusCreated   Status = "CREATED"
	StatusActive    Status = "ACTIVE"
	StatusCommitted Status = "COMMITTED"
	StatusPartial   Status = "PARTIAL"
	StatusDiscarded Status = "DISCARDED"
)

type EntityDelta struct {
	ID         string         `json:"id"`
	Label      string         `json:"label"`
	Properties map[string]any `json:"properties"`
}

type EdgeDelta struct {
	ID         string         `json:"id"`
	SourceID   string         `json:"source_id"`
	TargetID   string         `json:"target_id"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

type OverlayGraph struct {
	OverlayID     string        `json:"overlay_id"`
	Namespace     string        `json:"namespace"`
	SessionID     string        `json:"session_id"`
	BaseVersionID string        `json:"base_version_id"`
	Status        Status        `json:"status"`
	EntitiesDelta []EntityDelta `json:"entities_delta"`
	EdgesDelta    []EdgeDelta   `json:"edges_delta"`
	CreatedAt     time.Time     `json:"created_at"`
	ExpiresAt     time.Time     `json:"expires_at"`
	CommittedAt   *time.Time    `json:"committed_at,omitempty"`
}

type CommitResult struct {
	NewVersionID      string `json:"new_version_id"`
	EntitiesCommitted int    `json:"entities_committed"`
	EdgesCommitted    int    `json:"edges_committed"`
	ConflictsResolved int    `json:"conflicts_resolved"`
}
