package usecase

import (
	"context"

	"vnp-memory/services/memobase-engine/internal/usecase/dto"
	"vnp-memory/services/memobase-engine/internal/usecase/port"
)

type mergeProfileUseCase struct {
	llmClient port.LLMClient
}

// NewMergeProfileUseCase creates a new instance of the YOLO merge profile use case.
func NewMergeProfileUseCase(llmClient port.LLMClient) port.ProfileMerger {
	return &mergeProfileUseCase{
		llmClient: llmClient,
	}
}

func (u *mergeProfileUseCase) MergeProfile(ctx context.Context, req dto.MergeProfileRequest) (dto.MergeProfileResponse, error) {
	// 1. Construct prompt using existing profiles (indexed) and new extracted facts.
	// 2. Call LLM #3 (YOLO merge).
	// 3. Parse JSON response into model.MergeDecision (add, update, delete).
	
	// Stub implementation
	return dto.MergeProfileResponse{}, nil
}
