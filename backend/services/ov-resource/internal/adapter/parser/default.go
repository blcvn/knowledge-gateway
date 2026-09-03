package parser

import (
	"context"

	"openviking.com/ov-resource/internal/domain/model"
)

type DefaultParser struct {
	config model.ParserConfig
}

func NewDefaultParser(config model.ParserConfig) *DefaultParser {
	return &DefaultParser{config: config}
}

func (p *DefaultParser) Parse(ctx context.Context, content []byte, filename string, config model.ParserConfig) ([]model.Chunk, error) {
	text := string(content)
	chunks := []model.Chunk{
		{
			ID:      "chunk-default-1",
			Content: text,
			Metadata: model.ChunkMetadata{
				StartLine:   1,
				EndLine:     1,
				TotalTokens: estimateTokens(text),
			},
		},
	}
	return chunks, nil
}

func (p *DefaultParser) Supports(filename string) bool {
	return true
}
