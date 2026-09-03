package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"vnp-memory/pkg/graph"
	"vnp-memory/services/graphiti-knowledge/internal/adapter/client/llm"
	"vnp-memory/services/graphiti-knowledge/internal/adapter/prompt"
)

// SagaWindow — triggers saga summarization when len(episodes) >= this
const SagaWindow = 10

// SummarizeSagaRequest — inputs for saga summarization
type SummarizeSagaRequest struct {
	Saga      graph.SagaNode
	Episodes  []graph.EpisodicNode // episodes to incorporate in this summary pass
	GroupID   string
}

// SummarizeSagaResult — updated saga after LLM summarization
type SummarizeSagaResult struct {
	UpdatedSaga graph.SagaNode
	TokenUsage  llm.TokenUsage
}

// SagaSummarizerUseCase — generates/updates saga summaries using LLM
type SagaSummarizerUseCase struct {
	llmClient llm.LLMClient
	prompts   *prompt.PromptRegistry
}

func NewSagaSummarizerUseCase(llmClient llm.LLMClient, prompts *prompt.PromptRegistry) *SagaSummarizerUseCase {
	return &SagaSummarizerUseCase{llmClient: llmClient, prompts: prompts}
}

// Execute summarizes a saga's episodes into an updated saga summary
func (uc *SagaSummarizerUseCase) Execute(ctx context.Context, req SummarizeSagaRequest) (*SummarizeSagaResult, error) {
	summaryTPL := uc.prompts.MustGet("summarize_saga")

	prevEpisodeContents := make([]string, len(req.Episodes))
	for i, ep := range req.Episodes {
		prevEpisodeContents[i] = fmt.Sprintf("[%s] %s", ep.ValidAt.Format(time.DateTime), ep.Content)
	}

	pctx := prompt.PromptContext{
		ExistingNodes: []string{req.Saga.Summary},
		PrevEpisodes:  prevEpisodeContents,
		ReferenceTime: time.Now().Format(time.RFC3339),
	}

	messages := []llm.Message{
		{Role: "system", Content: summaryTPL.SystemPrompt},
		{Role: "user", Content: summaryTPL.BuildUser(pctx)},
	}

	resp, err := uc.llmClient.GenerateResponse(ctx, messages, llm.GenerateOpts{
		PromptName:     "summarize_saga",
		ModelSize:      llm.ModelSizeMedium,
		ResponseSchema: summaryTPL.Schema,
		Temperature:    0.0,
	})
	if err != nil {
		return nil, fmt.Errorf("summarize saga LLM: %w", err)
	}

	var output struct {
		Summary string `json:"summary"`
		Title   string `json:"title"`
	}
	if err := json.Unmarshal(resp.Content, &output); err != nil {
		return nil, fmt.Errorf("parse saga summary: %w", err)
	}

	now := time.Now()
	updatedSaga := req.Saga
	updatedSaga.Summary = output.Summary
	if output.Title != "" && req.Saga.Name == "" {
		updatedSaga.Name = output.Title
	}
	updatedSaga.LastSummarizedAt = &now
	updatedSaga.UpdatedAt = now

	return &SummarizeSagaResult{
		UpdatedSaga: updatedSaga,
		TokenUsage:  resp.TokenUsage,
	}, nil
}
