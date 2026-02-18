package v32

import (
	"context"
	"time"
)

// Approval represents a document approval action
type Approval struct {
	ID         string          `json:"id" gorm:"primaryKey"`
	DocumentID string          `json:"document_id" gorm:"index"`
	Tier       RequirementTier `json:"tier"`
	Comment    string          `json:"comment"`
	ApprovedBy string          `json:"approved_by"`
	ApprovedAt time.Time       `json:"approved_at"`
}

// ApprovalRepository defines the interface for approval persistence
type ApprovalRepository interface {
	Create(ctx context.Context, approval *Approval) error
	GetByID(ctx context.Context, id string) (*Approval, error)
	GetByDocumentID(ctx context.Context, documentID string) ([]*Approval, error)
}
