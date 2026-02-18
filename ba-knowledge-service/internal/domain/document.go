package v32

import "time"

// RequirementTier enum matching Proto definition
type RequirementTier int32

const (
	TierUnspecified RequirementTier = 0
	TierPRD         RequirementTier = 1
	TierURDIndex    RequirementTier = 2
	TierURDOutline  RequirementTier = 3
	TierURDFull     RequirementTier = 4
)

func (t RequirementTier) String() string {
	switch t {
	case TierPRD:
		return "PRD"
	case TierURDIndex:
		return "URD_INDEX"
	case TierURDOutline:
		return "URD_OUTLINE"
	case TierURDFull:
		return "URD_FULL"
	default:
		return "UNKNOWN"
	}
}

// DocumentStatus enum
type DocumentStatus string

const (
	DocumentStatusDraft      DocumentStatus = "draft"
	DocumentStatusGenerating DocumentStatus = "generating"
	DocumentStatusReviewing  DocumentStatus = "reviewing"
	DocumentStatusApproved   DocumentStatus = "approved"
	DocumentStatusFailed     DocumentStatus = "failed"
)

// Document represents a generic requirement document in the system
type Document struct {
	ID               string          `json:"id" bson:"_id"`
	ProjectID        string          `json:"project_id" bson:"project_id"`
	ParentDocumentID string          `json:"parent_document_id" bson:"parent_document_id"` // E.g., Index ID for Outline
	RootDocumentID   string          `json:"root_document_id" bson:"root_document_id"`     // E.g., BRD ID for URD
	Tier             RequirementTier `json:"tier" bson:"tier"`
	Status           DocumentStatus  `json:"status" bson:"status"`
	ModuleName       string          `json:"module_name" bson:"module_name"`
	Version          int             `json:"version" bson:"version"`
	Content          string          `json:"content" bson:"content"` // Markdown content
	CreatedAt        time.Time       `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" bson:"updated_at"`
	ApprovedAt       *time.Time      `json:"approved_at,omitempty" bson:"approved_at,omitempty"`
}
