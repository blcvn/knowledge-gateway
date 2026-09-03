package usecase

import (
	"context"
	"encoding/json"
	"vnp-memory/services/graphiti-knowledge/domain"
	"vnp-memory/services/graphiti-knowledge/usecase/dto"
	"vnp-memory/services/graphiti-knowledge/usecase/port"
)

type extractEdgesUseCase struct {
	llm      port.LLMClient
	registry port.PromptRegistry
	parser   interface{ ParseJSON(string) string }
}

func NewExtractEdgesUseCase(llm port.LLMClient, registry port.PromptRegistry, parser interface{ ParseJSON(string) string }) port.ExtractEdgesUseCase {
	return &extractEdgesUseCase{llm: llm, registry: registry, parser: parser}
}

func (uc *extractEdgesUseCase) Execute(ctx context.Context, req dto.ExtractEdgesRequest) ([]domain.ExtractedEdge, domain.TokenUsage, error) {
	vars := map[string]interface{}{
		"Content":          req.Content,
		"Entities":         req.Entities,
		"PreviousEpisodes": req.PreviousEpisodes,
	}
	prompt, err := uc.registry.Render("extract_edges", vars)
	if err != nil {
		return nil, domain.TokenUsage{}, err
	}
	model := uc.registry.GetModel("extract_edges")

	respMarkdown, usage, err := uc.llm.Complete(ctx, prompt, model)
	if err != nil {
		return nil, domain.TokenUsage{}, err
	}

	jsonStr := uc.parser.ParseJSON(respMarkdown)
	
	var result struct {
		Edges []domain.ExtractedEdge `json:"edges"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, usage, err
	}

	return result.Edges, usage, nil
}
