package domain

import "errors"

type ExtractedEntity struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Summary string `json:"summary"`
}

type ExtractedEdge struct {
	Source   string   `json:"source"`
	Target   string   `json:"target"`
	Relation string   `json:"relation"`
	Fact     string   `json:"fact"`
	Temporal []string `json:"temporal,omitempty"`
}

type Resolution struct {
	ExistingEntityID string            `json:"existing_entity_id"`
	ExtractedEntity  ExtractedEntity   `json:"extracted_entity"`
	Decision         DuplicateDecision `json:"decision"`
	Confidence       float64           `json:"confidence"`
}

type DuplicateDecision string

const (
	DecisionMerge  DuplicateDecision = "merge"
	DecisionCreate DuplicateDecision = "create"
	DecisionSkip   DuplicateDecision = "skip"
)

func (d DuplicateDecision) Validate() error {
	switch d {
	case DecisionMerge, DecisionCreate, DecisionSkip:
		return nil
	default:
		return errors.New("invalid duplicate decision")
	}
}
