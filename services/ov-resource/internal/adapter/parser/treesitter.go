package parser

import (
	"context"
	"path/filepath"
	"strings"

	"openviking.com/ov-resource/internal/domain/model"
)

type TreesitterParser struct {
	config model.ParserConfig
}

func NewTreesitterParser(config model.ParserConfig) *TreesitterParser {
	return &TreesitterParser{config: config}
}

func (p *TreesitterParser) Parse(ctx context.Context, content []byte, filename string, config model.ParserConfig) ([]model.Chunk, error) {
	// Stub implementation for Treesitter parsing
	// In reality, this would use CGo tree-sitter bindings
	text := string(content)
	chunks := []model.Chunk{
		{
			ID:      "chunk-1",
			Content: text,
			Metadata: model.ChunkMetadata{
				StartLine:   1,
				EndLine:     1,
				TotalTokens: estimateTokens(text),
				ASTNodeType: "module",
				ASTNodePath: "/",
			},
		},
	}
	return chunks, nil
}

func (p *TreesitterParser) Supports(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go", ".py", ".js", ".ts", ".rs", ".java":
		return true
	}
	return false
}
