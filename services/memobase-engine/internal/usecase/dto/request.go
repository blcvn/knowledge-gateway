package dto

import (
	"github.com/google/uuid"
	"vnp-memory/services/memobase-engine/internal/domain/model"
)

// ProcessBufferRequest is the DTO for the main pipeline.
type ProcessBufferRequest struct {
	UserID    string
	ProjectID string
	BufferIDs []uuid.UUID
}

// ExtractProfileRequest is the DTO for the extraction phase.
type ExtractProfileRequest struct {
	UserMemoStr   string
	ProfileSchema model.ProfileConfig
}

// MergeProfileRequest is the DTO for the YOLO merge phase.
type MergeProfileRequest struct {
	ExtractedFacts   []model.Profile
	ExistingProfiles []model.Profile
}

// ProcessEventRequest is the DTO for event processing.
type ProcessEventRequest struct {
	UserID    string
	ProjectID string
	MemoStr   string
	Config    model.ProfileConfig
}
