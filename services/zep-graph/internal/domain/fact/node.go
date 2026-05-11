package fact

import (
	"time"
	"github.com/google/uuid"
)

// NodePriority represents the strict 9-level Graphiti entity priority.
type NodePriority int

const (
	PriorityUser NodePriority = iota + 1
	PriorityAssistant
	PriorityPreference
	PriorityOrganization
	PriorityEvent
	PriorityLocation
	PriorityDocument
	PriorityTopic
	PriorityObject
)

// Node represents an extracted entity in the temporal KG.
type Node struct {
	UUID     uuid.UUID
	Name     string
	Label    string
	Priority NodePriority
	Metadata map[string]interface{}
}

// Edge represents a relationship with temporal validity.
type Edge struct {
	UUID      uuid.UUID
	SourceID  uuid.UUID
	TargetID  uuid.UUID
	Relation  string // e.g., "LOCATED_AT", "OCCURRED_AT"
	Fact      string // Human-readable statement
	ValidAt   *time.Time
	InvalidAt *time.Time
	CreatedAt time.Time
}
