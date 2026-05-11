package parser

import (
	"context"

	"openviking.com/ov-resource/internal/domain/model"
	"openviking.com/ov-resource/internal/usecase/port"
)

type Registry struct {
	parsers []port.ParserPort
}

func NewRegistry(config model.ParserConfig) *Registry {
	parsers := []port.ParserPort{
		NewMarkdownParser(config),
		NewDocumentParser(config),
	}
	if config.TreesitterEnabled {
		parsers = append(parsers, NewTreesitterParser(config))
	}
	// Default parser goes last
	parsers = append(parsers, NewDefaultParser(config))

	return &Registry{
		parsers: parsers,
	}
}

func (r *Registry) GetParser(filename string, forceParser string) port.ParserPort {
	if forceParser != "" {
		for _, p := range r.parsers {
			if getParserName(p) == forceParser {
				return p
			}
		}
	}

	for _, p := range r.parsers {
		if p.Supports(filename) {
			return p
		}
	}
	return r.parsers[len(r.parsers)-1] // fallback to default
}

func (r *Registry) Parse(ctx context.Context, content []byte, filename string, config model.ParserConfig) ([]model.Chunk, error) {
	parser := r.GetParser(filename, "")
	return parser.Parse(ctx, content, filename, config)
}

func (r *Registry) Supports(filename string) bool {
	return true
}

func getParserName(p port.ParserPort) string {
	switch p.(type) {
	case *TreesitterParser:
		return string(model.ParserTypeTreeSitter)
	case *MarkdownParser:
		return string(model.ParserTypeMarkdown)
	case *DocumentParser:
		return string(model.ParserTypeDocument)
	default:
		return string(model.ParserTypeDefault)
	}
}
