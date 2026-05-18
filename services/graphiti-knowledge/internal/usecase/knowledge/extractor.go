package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	pb "vnp-memory/services/graphiti-knowledge/internal/adapter/grpc/pb"
	"vnp-memory/services/graphiti-knowledge/internal/domain/repository"
)

// LLMClient defines the interface for interacting with the foundational models.
type LLMClient interface {
	GenerateJSON(ctx context.Context, systemPrompt, userPrompt string) ([]byte, error)
	EvaluateSimilarity(ctx context.Context, newEntity *pb.ExtractedEntity, candidates []*pb.ExtractedEntity) (*pb.Resolution, error)
}

// KnowledgeUsecase orchestrates the core knowledge extraction and resolution logic.
type KnowledgeUsecase struct {
	llm  LLMClient
	repo repository.GraphRepository
}

// NewKnowledgeUsecase creates a new knowledge usecase.
func NewKnowledgeUsecase(llm LLMClient, repo repository.GraphRepository) *KnowledgeUsecase {
	return &KnowledgeUsecase{
		llm:  llm,
		repo: repo,
	}
}

// ExtractEntities uses an LLM to identify structured entities from raw text.
func (uc *KnowledgeUsecase) ExtractEntities(ctx context.Context, content string, previousEpisodes []string, entityTypes []string) ([]*pb.ExtractedEntity, error) {
	systemPrompt := `You are an expert NLP system. Extract entities from the provided text.
Return ONLY a JSON array of objects with 'name', 'label', and 'summary' fields.
Focus specifically on these types: ` + fmt.Sprintf("%v", entityTypes)

	userPrompt := fmt.Sprintf("Context (Previous Episodes): %v\n\nContent to analyze: %s", previousEpisodes, content)

	// Call the LLM (e.g., GPT-4 or Claude 3.5 Sonnet)
	responseBytes, err := uc.llm.GenerateJSON(ctx, systemPrompt, userPrompt)
	if err != nil {
		log.Printf("LLM extraction failed: %v", err)
		return nil, fmt.Errorf("llm generation error: %w", err)
	}

	// Parse the JSON response
	var extracted []struct {
		Name    string `json:"name"`
		Label   string `json:"label"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(responseBytes, &extracted); err != nil {
		log.Printf("Failed to parse LLM JSON: %v. Raw Output: %s", err, string(responseBytes))
		return nil, fmt.Errorf("failed to parse extracted entities: %w", err)
	}

	// Map to Protobuf Domain Model
	var pbEntities []*pb.ExtractedEntity
	for _, e := range extracted {
		pbEntities = append(pbEntities, &pb.ExtractedEntity{
			Name:    e.Name,
			Label:   e.Label,
			Summary: e.Summary,
		})
	}

	return pbEntities, nil
}

// ResolveEntities processes a list of newly extracted entities and determines whether they are new
// or should be merged with existing entities in the graph (Deduplication).
func (uc *KnowledgeUsecase) ResolveEntities(ctx context.Context, extractedEntities []*pb.ExtractedEntity, groupID string) ([]*pb.Resolution, error) {
	if groupID == "" {
		return nil, errors.New("groupID is required for resolution context")
	}

	var finalResolutions []*pb.Resolution

	// 1. Loop through each newly extracted entity
	for _, newEntity := range extractedEntities {
		
		// 2. Query the GraphRepository (Vector Search) for semantic candidates
		candidates, err := uc.repo.SearchSimilarEntities(ctx, groupID, newEntity.Name, 5)
		if err != nil {
			// Log error but continue with other entities
			log.Printf("Error searching candidates for %s: %v\n", newEntity.Name, err)
			continue
		}

		// 3. If no candidates found, it's definitely a new entity
		if len(candidates) == 0 {
			finalResolutions = append(finalResolutions, &pb.Resolution{
				ExtractedEntity: newEntity,
				Decision:        "CREATE_NEW",
				Confidence:      1.0,
			})
			continue
		}

		// 4. If candidates exist, ask the LLMResolver to evaluate and decide
		// The LLM will check descriptions, labels, etc. to decide between CREATE_NEW or MERGE
		resolution, err := uc.llm.EvaluateSimilarity(ctx, newEntity, candidates)
		if err != nil {
			log.Printf("LLM failed to resolve %s: %v\n", newEntity.Name, err)
			// Fallback to CREATE_NEW if LLM fails
			finalResolutions = append(finalResolutions, &pb.Resolution{
				ExtractedEntity: newEntity,
				Decision:        "CREATE_NEW",
				Confidence:      0.5, // Low confidence due to fallback
			})
			continue
		}

		finalResolutions = append(finalResolutions, resolution)
	}

	return finalResolutions, nil
}
