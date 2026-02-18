package v32

import "time"

// PlannerEventType represents different types of planner events
type PlannerEventType string

const (
	EventIndexApproved   PlannerEventType = "index_approved"
	EventOutlineApproved PlannerEventType = "outline_approved"
	EventFullApproved    PlannerEventType = "full_approved"
	EventPRDUploaded     PlannerEventType = "prd_uploaded"
	EventReviewSubmitted PlannerEventType = "review_submitted"
)

// PlannerEvent represents an event in the document workflow
type PlannerEvent struct {
	ID         string                 `json:"id"`
	Type       PlannerEventType       `json:"type"`
	DocumentID string                 `json:"document_id"`
	Tier       RequirementTier        `json:"tier"`
	ProjectID  string                 `json:"project_id"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// EventEmitter defines interface for emitting planner events
type EventEmitter interface {
	Emit(event *PlannerEvent) error
}

// EventHandler defines interface for handling planner events
type EventHandler interface {
	Handle(event *PlannerEvent) error
}
