package usecase

import (
	"context"

	"github.com/vnp-memory/services/ov-session/adapter/client"
	"github.com/vnp-memory/services/ov-session/domain/model"
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
