package dto

import "vnp-memory/services/memobase-engine/internal/domain/model"

// ProcessBufferResponse is the response DTO for the main pipeline.
type ProcessBufferResponse struct {
	Result model.PipelineResult
}

// ExtractProfileResponse is the response DTO for extraction.
type ExtractProfileResponse struct {
	ExtractedFacts []model.Profile
}

// MergeProfileResponse is the response DTO for YOLO merge.
type MergeProfileResponse struct {
	Decision model.MergeDecision
}
