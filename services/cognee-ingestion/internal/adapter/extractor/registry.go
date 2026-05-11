// Package extractor implements the TextExtractor port with format-specific handlers.
package extractor

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/domain"
)

// FormatExtractor extracts text from a single MIME type.
type FormatExtractor interface {
	Extract(ctx context.Context, reader io.Reader) (string, error)
	MimeType() domain.MimeType
}

// Registry routes MimeType to the appropriate FormatExtractor.
// It implements port.TextExtractor.
type Registry struct {
	mu         sync.RWMutex
	extractors map[domain.MimeType]FormatExtractor
}

// NewRegistry creates a new extractor registry with all default format handlers.
func NewRegistry() *Registry {
	r := &Registry{
		extractors: make(map[domain.MimeType]FormatExtractor),
	}

	// Register built-in extractors
	r.Register(&PlainTextExtractor{})
	r.Register(&MarkdownExtractor{})
	r.Register(&HTMLExtractor{})
	r.Register(&CSVExtractor{})
	r.Register(&JSONExtractor{})
	// PDF, DOCX, PPTX require external libraries — registered separately
	// r.Register(NewPDFExtractor())
	// r.Register(NewDOCXExtractor())

	return r
}

// Register adds a format extractor to the registry.
func (r *Registry) Register(ext FormatExtractor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.extractors[ext.MimeType()] = ext
}

// Extract dispatches extraction to the appropriate format handler.
func (r *Registry) Extract(ctx context.Context, reader io.Reader, mimeType domain.MimeType) (string, error) {
	r.mu.RLock()
	ext, ok := r.extractors[mimeType]
	r.mu.RUnlock()

	if !ok {
		return "", &domain.ErrUnsupportedFormat{MimeType: string(mimeType)}
	}

	text, err := ext.Extract(ctx, reader)
	if err != nil {
		return "", &domain.ErrExtractionFailed{Filename: string(mimeType), Cause: err}
	}
	return text, nil
}

// Supported returns all registered MIME types.
func (r *Registry) Supported() []domain.MimeType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]domain.MimeType, 0, len(r.extractors))
	for mt := range r.extractors {
		types = append(types, mt)
	}
	return types
}

// PlainTextExtractor handles text/plain — passthrough.
type PlainTextExtractor struct{}

func (e *PlainTextExtractor) Extract(_ context.Context, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read text: %w", err)
	}
	return string(data), nil
}

func (e *PlainTextExtractor) MimeType() domain.MimeType { return domain.MimePlainText }

// MarkdownExtractor handles text/markdown — passthrough (markdown is text).
type MarkdownExtractor struct{}

func (e *MarkdownExtractor) Extract(_ context.Context, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read markdown: %w", err)
	}
	return string(data), nil
}

func (e *MarkdownExtractor) MimeType() domain.MimeType { return domain.MimeMarkdown }

// HTMLExtractor handles text/html — strips HTML tags, returns text content.
type HTMLExtractor struct{}

func (e *HTMLExtractor) Extract(_ context.Context, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read html: %w", err)
	}
	// Simple tag stripping — in production, use x/net/html tokenizer
	return stripHTMLTags(string(data)), nil
}

func (e *HTMLExtractor) MimeType() domain.MimeType { return domain.MimeHTML }

// stripHTMLTags removes HTML tags from content (simple implementation).
func stripHTMLTags(s string) string {
	var result []byte
	inTag := false
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '<':
			inTag = true
		case s[i] == '>':
			inTag = false
			result = append(result, ' ')
		case !inTag:
			result = append(result, s[i])
		}
	}
	return string(result)
}

// CSVExtractor handles text/csv — returns rows as text.
type CSVExtractor struct{}

func (e *CSVExtractor) Extract(_ context.Context, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read csv: %w", err)
	}
	return string(data), nil
}

func (e *CSVExtractor) MimeType() domain.MimeType { return domain.MimeCSV }

// JSONExtractor handles application/json — returns JSON as text.
type JSONExtractor struct{}

func (e *JSONExtractor) Extract(_ context.Context, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read json: %w", err)
	}
	return string(data), nil
}

func (e *JSONExtractor) MimeType() domain.MimeType { return domain.MimeJSON }
