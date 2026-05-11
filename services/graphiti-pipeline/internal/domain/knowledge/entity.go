package knowledge

import "time"

type ExtractedEntity struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Summary string `json:"summary"`
}

type ExtractedEdge struct {
	Source    string     `json:"source"`
	Target    string     `json:"target"`
	Relation  string     `json:"relation"`
	ValidAt   time.Time  `json:"valid_at"`
	InvalidAt *time.Time `json:"invalid_at,omitempty"`
	ExpiredAt *time.Time `json:"expired_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func (e *ExtractedEdge) Validate() error {
	if e.ValidAt.IsZero() {
		return ErrInvalidEdgeValidAt
	}
	if e.InvalidAt != nil && e.ValidAt.After(*e.InvalidAt) {
		return ErrInvalidEdgeTimeWindow
	}
	return nil
}

type DuplicateDecision string

const (
	DecisionMerge  DuplicateDecision = "merge"
	DecisionCreate DuplicateDecision = "create"
	DecisionSkip   DuplicateDecision = "skip"
)

type Resolution struct {
	ExistingEntityID string            `json:"existing_entity_id,omitempty"`
	ExtractedEntity  ExtractedEntity   `json:"extracted_entity"`
	Decision         DuplicateDecision `json:"decision"`
	Confidence       float64           `json:"confidence"`
}
