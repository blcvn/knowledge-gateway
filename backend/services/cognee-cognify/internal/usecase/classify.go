package usecase

import (
	"context"
	"log/slog"

	"vnp-memory/services/cognee-cognify/internal/domain"
	"vnp-memory/services/cognee-cognify/internal/usecase/port"
)

// Stage is the interface for each pipeline stage (old-style, with job context).
type Stage interface {
	Name() domain.StageType
	Execute(ctx context.Context, job *domain.CognifyJob, state *CognifyPipelineState) error
}

// CognifyPipelineState holds intermediate results for the old-style stage pipeline.
// The new-style chain uses PipelineState from start_cognify.go.
type CognifyPipelineState struct {
	TextItems      []port.TextItem
	Classification *domain.ClassificationResult
	Chunks         []*domain.Chunk
	Entities       []*domain.Entity
	Relationships  []*domain.Relationship
	Communities    []*domain.Community
}

// ClassifyStage determines content type and chunking strategy via LLM.
type ClassifyStage struct {
	llm    port.LLMClient
	logger *slog.Logger
}

func NewClassifyStage(llm port.LLMClient, logger *slog.Logger) *ClassifyStage {
	return &ClassifyStage{llm: llm, logger: logger.With("stage", "classify")}
}

func (s *ClassifyStage) Name() domain.StageType { return domain.StageClassify }

func (s *ClassifyStage) Execute(ctx context.Context, job *domain.CognifyJob, state *CognifyPipelineState) error {
	if len(state.TextItems) == 0 {
		state.Classification = &domain.ClassificationResult{
			Strategy: domain.StrategyRecursive,
		}
		return nil
	}

	// Take sample from first text item (up to 2000 chars)
	sample := state.TextItems[0].Text
	if len(sample) > 2000 {
		sample = sample[:2000]
	}

	result := &domain.ClassificationResult{}
	err := s.llm.CompleteStructured(ctx,
		classifySystemPrompt,
		"Classify this content:\n\n"+sample,
		result,
	)
	if err != nil {
		s.logger.Warn("LLM classification failed, using defaults", "error", err)
		result = &domain.ClassificationResult{
			ContentType: "general",
			Language:    "en",
			Strategy:    domain.StrategyRecursive,
		}
	}

	state.Classification = result
	job.Metrics.LLMCallsTotal++
	s.logger.Info("classification complete", "content_type", result.ContentType, "strategy", result.Strategy)
	return nil
}

const classifySystemPrompt = `You are a content classifier. Analyze the given text and return a JSON object:
{
  "content_type": "<technical|narrative|tabular|conversational|code>",
  "language": "<ISO 639-1 code>",
  "topics": ["<topic1>", "<topic2>"],
  "strategy": "<recursive|semantic|sentence|paragraph>"
}
Choose "semantic" for technical/complex text, "recursive" for general, "sentence" for conversational, "paragraph" for narrative.`
