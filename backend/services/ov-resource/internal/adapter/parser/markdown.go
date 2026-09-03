package parser

import (
	"context"
	"path/filepath"
	"strings"

	"openviking.com/ov-resource/internal/domain/model"
)

type MarkdownParser struct {
	config model.ParserConfig
}

func NewMarkdownParser(config model.ParserConfig) *MarkdownParser {
	return &MarkdownParser{config: config}
}

func (p *MarkdownParser) Parse(ctx context.Context, content []byte, filename string, config model.ParserConfig) ([]model.Chunk, error) {
	// Stub implementation for Markdown parsing
	text := string(content)
	chunks := []model.Chunk{
		{
			ID:      "chunk-md-1",
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

func (p *MarkdownParser) Supports(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".md" || ext == ".mdx"
}
