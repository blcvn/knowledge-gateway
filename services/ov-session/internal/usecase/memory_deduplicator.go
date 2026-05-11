package usecase

import (
	"context"
	"time"

	"github.com/vnp-memory/services/ov-session/internal/adapter/client"
	"github.com/vnp-memory/services/ov-session/internal/domain/model"
)

type MemoryDeduplicatorUseCase interface {
	Deduplicate(ctx context.Context, candidates []model.CandidateMemory) ([]model.CandidateMemory, error)
}

type memoryDeduplicatorUseCaseImpl struct {
	llmClient client.LLMClient
}

func NewMemoryDeduplicatorUseCase(llmClient client.LLMClient) MemoryDeduplicatorUseCase {
	return &memoryDeduplicatorUseCaseImpl{
		llmClient: llmClient,
	}
}

func (uc *memoryDeduplicatorUseCaseImpl) Deduplicate(ctx context.Context, candidates []model.CandidateMemory) ([]model.CandidateMemory, error) {
	var finalMemories []model.CandidateMemory
	for _, cand := range candidates {
		// Mock deduplication logic based on TDD
		// In reality, this would do similarity search against existing memories using vector embeddings.
		
		sim := 0.5 // Mock similarity
		
		if sim == 1.0 {
			cand.DedupAction = model.DedupActionSkip
		} else if sim > 0.85 {
			cand.DedupAction = model.DedupActionMerge
			fused, _ := uc.llmClient.FuseMemories(ctx, &cand, nil)
			cand = *fused
		} else if sim > 0.60 {
			cand.DedupAction = model.DedupActionCreate
		} else {
			cand.DedupAction = model.DedupActionCreate
		}
		
		if cand.DedupAction != model.DedupActionSkip {
			cand.CreatedAt = time.Now()
			finalMemories = append(finalMemories, cand)
		}
	}
	return finalMemories, nil
}
