package client

import (
	"context"

	"github.com/vnp-memory/services/ov-session/internal/domain/model"
)

type LLMClient interface {
	CompressSession(ctx context.Context, messages []*model.Message, version model.CompressionVersion) (string, error)
	ExtractMemories(ctx context.Context, archive string) ([]model.CandidateMemory, error)
	FuseMemories(ctx context.Context, m1, m2 *model.CandidateMemory) (*model.CandidateMemory, error)
	EvaluateWM(ctx context.Context, currentWM *model.WorkingMemory, newMsg *model.Message) (*model.WorkingMemory, error)
}

type llmClientImpl struct{}

func NewLLMClient() LLMClient {
	return &llmClientImpl{}
}

func (c *llmClientImpl) CompressSession(ctx context.Context, messages []*model.Message, version model.CompressionVersion) (string, error) {
	return "compressed_archive_content", nil
}

func (c *llmClientImpl) ExtractMemories(ctx context.Context, archive string) ([]model.CandidateMemory, error) {
	return []model.CandidateMemory{}, nil
}

func (c *llmClientImpl) FuseMemories(ctx context.Context, m1, m2 *model.CandidateMemory) (*model.CandidateMemory, error) {
	return m1, nil
}

func (c *llmClientImpl) EvaluateWM(ctx context.Context, currentWM *model.WorkingMemory, newMsg *model.Message) (*model.WorkingMemory, error) {
	return currentWM, nil
}
