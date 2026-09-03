// Package steps implements the CLASSIFY pipeline step.
// Detects content type and language from raw content.
package steps

import (
	"context"
	"strings"

	"vnp-memory/services/cognee-cognify/internal/domain"
	"vnp-memory/services/cognee-cognify/internal/usecase"
)

// ClassifyStep detects content type and language.
type ClassifyStep struct{}

func NewClassifyStep() *ClassifyStep { return &ClassifyStep{} }

func (s *ClassifyStep) Name() domain.PipelineStep { return domain.StepClassify }

func (s *ClassifyStep) Execute(ctx context.Context, state *usecase.PipelineState) (*usecase.PipelineState, error) {
	if len(state.RawContent) == 0 {
		// Nothing to classify — passthrough
		return state, nil
	}
	// Heuristic classification based on content characteristics
	sample := ""
	if len(state.RawContent) > 0 {
		sample = state.RawContent[0]
	}
	state.ContentType = detectContentType(sample)
	return state, nil
}

func detectContentType(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "http://") || strings.HasPrefix(content, "https://") {
		return "URL"
	}
	if strings.HasPrefix(content, "{") || strings.HasPrefix(content, "[") {
		return "JSON"
	}
	if strings.Contains(content, "<!DOCTYPE") || strings.Contains(content, "<html") {
		return "HTML"
	}
	return "TEXT"
}
