package parser

import (
	"context"
	"path/filepath"
	"strings"

	"openviking.com/ov-resource/internal/domain/model"
)

type DocumentParser struct {
	config model.ParserConfig
}

func NewDocumentParser(config model.ParserConfig) *DocumentParser {
	return &DocumentParser{config: config}
}

func (p *DocumentParser) Parse(ctx context.Context, content []byte, filename string, config model.ParserConfig) ([]model.Chunk, error) {
	// Stub implementation for Document parsing
	text := "extracted text from doc"
	chunks := []model.Chunk{
		{
			ID:      "chunk-doc-1",
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

func (p *DocumentParser) Supports(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".pdf" || ext == ".docx"
}
