package usecase

import (
	"context"
	"encoding/json"
	"vnp-memory/services/graphiti-knowledge/internal/domain"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/dto"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/port"
)

type resolveEdgesUseCase struct {
	llm      port.LLMClient
	registry port.PromptRegistry
	parser   interface{ ParseJSON(string) string }
}

func NewResolveEdgesUseCase(llm port.LLMClient, registry port.PromptRegistry, parser interface{ ParseJSON(string) string }) port.ResolveEdgesUseCase {
	return &resolveEdgesUseCase{llm: llm, registry: registry, parser: parser}
}

func (uc *resolveEdgesUseCase) Execute(ctx context.Context, req dto.ResolveEdgesRequest) (dto.ResolveEdgesResponse, error) {
	// For each extracted edge, we might query existing edges and resolve.
	// Simplified to bulk processing for brevity
	vars := map[string]interface{}{
		"NewEdge":       req.ExtractedEdges,
		"ExistingEdges": "[]", // Should be fetched from graph reader
	}
	prompt, err := uc.registry.Render("resolve_edges", vars)
	if err != nil {
		return dto.ResolveEdgesResponse{}, err
	}
	model := uc.registry.GetModel("resolve_edges")

	respMarkdown, _, err := uc.llm.Complete(ctx, prompt, model)
	if err != nil {
		return dto.ResolveEdgesResponse{}, err
	}

	jsonStr := uc.parser.ParseJSON(respMarkdown)
	
	var result struct {
		ResolvedEdges []domain.ExtractedEdge `json:"resolved_edges"`
		Invalidated   []string               `json:"invalidated_edges"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return dto.ResolveEdgesResponse{}, err
	}

	return dto.ResolveEdgesResponse{
		ResolvedEdges: result.ResolvedEdges,
		Invalidated:   result.Invalidated,
	}, nil
}
