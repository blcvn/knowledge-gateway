package usecase

import (
	"context"

	"github.com/vnp-memory/services/ov-session/internal/adapter/client"
	"github.com/vnp-memory/services/ov-session/internal/domain/model"
)

type MemoryExtractorUseCase interface {
	Extract(ctx context.Context, archive string) ([]model.CandidateMemory, error)
}

type memoryExtractorUseCaseImpl struct {
	llmClient client.LLMClient
}

func NewMemoryExtractorUseCase(llmClient client.LLMClient) MemoryExtractorUseCase {
	return &memoryExtractorUseCaseImpl{
		llmClient: llmClient,
	}
}

func (uc *memoryExtractorUseCaseImpl) Extract(ctx context.Context, archive string) ([]model.CandidateMemory, error) {
	return uc.llmClient.ExtractMemories(ctx, archive)
}
