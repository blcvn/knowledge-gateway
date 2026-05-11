package usecase

import (
	"context"
	"vnp-memory/services/graphiti-knowledge/internal/domain"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/dto"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/port"
)

type resolveEntitiesUseCase struct {
	llm      port.LLMClient
	embedder port.EmbedderClient
	reader   port.GraphReader
	registry port.PromptRegistry
	parser   interface{ ParseJSON(string) string }
}

func NewResolveEntitiesUseCase(llm port.LLMClient, embedder port.EmbedderClient, reader port.GraphReader, registry port.PromptRegistry, parser interface{ ParseJSON(string) string }) port.ResolveEntitiesUseCase {
	return &resolveEntitiesUseCase{llm: llm, embedder: embedder, reader: reader, registry: registry, parser: parser}
}

func (uc *resolveEntitiesUseCase) Execute(ctx context.Context, req dto.ResolveEntitiesRequest) ([]domain.Resolution, error) {
	resolutions := []domain.Resolution{}
	
	for _, entity := range req.ExtractedEntities {
		// Embed entity name
		vec, err := uc.embedder.Embed(ctx, entity.Name, "embedding-model")
		if err != nil {
			continue // Skip or handle
		}
		
		// Search similar
		similar, err := uc.reader.FindSimilarEntities(ctx, vec, 0.85)
		if err != nil {
			continue // Skip or handle
		}

		if len(similar) > 0 {
			// Needs LLM resolution
			vars := map[string]interface{}{
				"Extracted": entity,
				"Existing":  similar,
			}
			prompt, _ := uc.registry.Render("resolve_entities", vars)
			// Mocking decision parsing here to keep it short
			_, _, _ = uc.llm.Complete(ctx, prompt, uc.registry.GetModel("resolve_entities"))

			resolutions = append(resolutions, domain.Resolution{
				ExtractedEntity: entity,
				Decision:        domain.DecisionMerge,
				ExistingEntityID: similar[0].Name, // using Name as ID mockup
			})
		} else {
			resolutions = append(resolutions, domain.Resolution{
				ExtractedEntity: entity,
				Decision:        domain.DecisionCreate,
			})
		}
	}
	
	return resolutions, nil
}
