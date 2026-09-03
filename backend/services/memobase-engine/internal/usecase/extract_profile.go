package usecase

import (
	"context"

	"vnp-memory/services/memobase-engine/internal/usecase/dto"
	"vnp-memory/services/memobase-engine/internal/usecase/port"
)

type extractProfileUseCase struct {
	llmClient port.LLMClient
}

// NewExtractProfileUseCase creates a new instance of the extract profile use case.
func NewExtractProfileUseCase(llmClient port.LLMClient) port.ProfileExtractor {
	return &extractProfileUseCase{
		llmClient: llmClient,
	}
}

func (u *extractProfileUseCase) ExtractProfile(ctx context.Context, req dto.ExtractProfileRequest) (dto.ExtractProfileResponse, error) {
	// 1. Construct prompt using req.UserMemoStr and req.ProfileSchema
	// 2. Call LLM #2
	// 3. Parse JSON response into model.Profile list
	
	// Stub implementation
	return dto.ExtractProfileResponse{}, nil
}
