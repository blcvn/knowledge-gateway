package usecase

import (
	"context"
	"encoding/json"
	"vnp-memory/services/graphiti-knowledge/internal/domain"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/dto"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/port"
)

type extractEntitiesUseCase struct {
	llm      port.LLMClient
	registry port.PromptRegistry
	parser   interface {
		ParseJSON(string) string
	}
}

func NewExtractEntitiesUseCase(llm port.LLMClient, registry port.PromptRegistry, parser interface{ ParseJSON(string) string }) port.ExtractEntitiesUseCase {
	return &extractEntitiesUseCase{llm: llm, registry: registry, parser: parser}
}

func (uc *extractEntitiesUseCase) Execute(ctx context.Context, req dto.ExtractEntitiesRequest) ([]domain.ExtractedEntity, domain.TokenUsage, error) {
	// Build prompt
	vars := map[string]interface{}{
		"Content":          req.Content,
		"PreviousEpisodes": req.PreviousEpisodes,
		"EntityTypes":      req.EntityTypes,
	}
	prompt, err := uc.registry.Render("extract_entities", vars)
	if err != nil {
		return nil, domain.TokenUsage{}, err
	}
	model := uc.registry.GetModel("extract_entities")

	// Call LLM
	respMarkdown, usage, err := uc.llm.Complete(ctx, prompt, model)
	if err != nil {
		return nil, domain.TokenUsage{}, err
	}

	// Parse JSON
	jsonStr := uc.parser.ParseJSON(respMarkdown)
	
	var result struct {
		Entities []domain.ExtractedEntity `json:"entities"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, usage, err
	}

	return result.Entities, usage, nil
}
