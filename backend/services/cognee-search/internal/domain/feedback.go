// Package domain defines Interaction and FeedbackRecord entities for search feedback loop.
// TASK-CE-009: Feedback Loop (SOL-005 §2.1)
// These types are persisted in cognee_interactions and cognee_feedback_records tables.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Interaction — a logged search call (recorded when save_interaction=true).
// Backed by cognee_interactions table (migration 0047).
type Interaction struct {
	ID           uuid.UUID
	TenantID     string
	SessionID    *string    // optional grouping for conversation threads
	DatasetID    *uuid.UUID // optional dataset filter used in the search
	Query        string
	Strategy     string
	ResultIDs    []string
	ResultScores []float64
	NodeSets     []string
	Timestamp    time.Time
}

// FeedbackRecord — user feedback on a specific search interaction.
// Backed by cognee_feedback_records table (migration 0047).
// Score in [-1.0, 1.0]: positive triggers edge weight boost, negative triggers penalty.
type FeedbackRecord struct {
	ID            uuid.UUID
	InteractionID uuid.UUID
	TenantID      string
	Score         float64  // -1.0 (negative) to 1.0 (positive)
	Text          string   // optional text comment
	AffectedNodes []string // Neo4j node IDs whose edge weights were adjusted
	CreatedAt     time.Time
}
