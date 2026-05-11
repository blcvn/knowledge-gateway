package usecase

import (
	"context"
	"encoding/json"
	"vnp-memory/services/graphiti-knowledge/internal/domain"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/dto"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/port"
)

type updateCommunityUseCase struct {
	llm      port.LLMClient
	registry port.PromptRegistry
	parser   interface{ ParseJSON(string) string }
}

func NewUpdateCommunityUseCase(llm port.LLMClient, registry port.PromptRegistry, parser interface{ ParseJSON(string) string }) port.UpdateCommunityUseCase {
	return &updateCommunityUseCase{llm: llm, registry: registry, parser: parser}
}

func (uc *updateCommunityUseCase) Execute(ctx context.Context, req dto.UpdateCommunityRequest) (dto.UpdateCommunityResponse, error) {
	vars := map[string]interface{}{
		"Members": req.EntityIDs,
		"Edges":   "[]", // Mock fetching edges
	}
	prompt, err := uc.registry.Render("summarize_community", vars)
	if err != nil {
		return dto.UpdateCommunityResponse{}, err
	}
	model := uc.registry.GetModel("summarize_community")

	respMarkdown, _, err := uc.llm.Complete(ctx, prompt, model)
	if err != nil {
		return dto.UpdateCommunityResponse{}, err
	}

	jsonStr := uc.parser.ParseJSON(respMarkdown)
	var summaryData struct {
		Summary string `json:"summary"`
	}
	json.Unmarshal([]byte(jsonStr), &summaryData)

	return dto.UpdateCommunityResponse{
		UpdatedCommunities: []domain.CommunityNode{
			{Name: "Community", Summary: summaryData.Summary},
		},
	}, nil
}
