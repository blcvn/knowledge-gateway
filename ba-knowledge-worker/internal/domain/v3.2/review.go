package v32

import (
	"context"
	"time"
)

// ReviewActionType represents the type of action required from a review
type ReviewActionType string

const (
	ActionPartialRegen  ReviewActionType = "partial_regeneration"
	ActionClarification ReviewActionType = "clarification"
	ActionComment       ReviewActionType = "comment"
	ActionFeedback      ReviewActionType = "feedback"
	ActionPending       ReviewActionType = "pending"
)

// Modification represents a proposed change to the document
type Modification struct {
	ModificationID string `json:"modification_id"`
	SectionType    string `json:"section_type"`
	SectionID      string `json:"section_id"`
	ActionType     string `json:"action_type"` // add, modify, delete, no_change
	OldContent     string `json:"old_content,omitempty"`
	NewContent     string `json:"new_content,omitempty"`
	StartIndex     int    `json:"start_index"`
	EndIndex       int    `json:"end_index"`
	Reasoning      string `json:"reasoning"`
	ImpactAnalysis string `json:"impact_analysis"`
}

// Review represents a review comment on a document
type Review struct {
	ID               string                   `json:"id" gorm:"primaryKey"`
	DocumentID       string                   `json:"document_id" gorm:"index"`
	NewDocumentID    string                   `json:"new_document_id" gorm:"index"`
	Tier             RequirementTier          `json:"tier"`
	Comment          string                   `json:"comment"`
	ActionType       ReviewActionType         `json:"action_type"`
	AffectedSections []string                 `json:"affected_sections" gorm:"serializer:json"`
	CreatedAt        time.Time                `json:"created_at"`
	DiffMarkdown     string                   `json:"diff_markdown"`
	ModTracings      []*Modification          `json:"mod_tracings" gorm:"serializer:json"`
}

// ReviewRepository defines the interface for review persistence
type ReviewRepository interface {
	Create(ctx context.Context, review *Review) error
	GetByDocumentID(ctx context.Context, documentID string) ([]*Review, error)
	GetByID(ctx context.Context, id string) (*Review, error)
	Update(ctx context.Context, review *Review) error
}
