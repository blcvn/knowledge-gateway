package usecase

import (
	"context"

	"github.com/vnp-memory/services/ov-session/internal/adapter/client"
	"github.com/vnp-memory/services/ov-session/internal/domain/model"
)

type CompressorUseCase interface {
	Compress(ctx context.Context, messages []*model.Message, version model.CompressionVersion) (string, error)
}

type compressorUseCaseImpl struct {
	llmClient client.LLMClient
}

func NewCompressorUseCase(llmClient client.LLMClient) CompressorUseCase {
	return &compressorUseCaseImpl{
		llmClient: llmClient,
	}
}

func (uc *compressorUseCaseImpl) Compress(ctx context.Context, messages []*model.Message, version model.CompressionVersion) (string, error) {
	return uc.llmClient.CompressSession(ctx, messages, version)
}
