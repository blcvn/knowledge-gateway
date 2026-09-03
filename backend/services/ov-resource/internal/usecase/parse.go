package usecase

import (
	"context"

	"openviking.com/ov-resource/internal/domain/model"
	"openviking.com/ov-resource/internal/usecase/dto"
	"openviking.com/ov-resource/internal/usecase/port"
)

type parseUseCase struct {
	parserRegistry port.ParserPort
}

func NewParseUseCase(parserRegistry port.ParserPort) *parseUseCase {
	return &parseUseCase{
		parserRegistry: parserRegistry,
	}
}

func (u *parseUseCase) Execute(ctx context.Context, req dto.ParseRequest) (dto.ParseResponse, error) {
	config := model.ParserConfig{
		ChunkSizeTokens:    int(req.ChunkSize),
		ChunkOverlapTokens: int(req.ChunkOverlap),
	}
	chunks, err := u.parserRegistry.Parse(ctx, req.Content, req.Filename, config)
	if err != nil {
		return dto.ParseResponse{}, err
	}
	return dto.ParseResponse{Chunks: chunks}, nil
}
