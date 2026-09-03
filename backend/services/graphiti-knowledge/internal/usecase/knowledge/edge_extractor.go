package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	pb "vnp-memory/services/graphiti-knowledge/adapter/grpc/pb"
)

// ExtractEdges uses the LLM to find relationships (edges) between extracted entities.
func (uc *KnowledgeUsecase) ExtractEdges(ctx context.Context, content string, entities []*pb.ExtractedEntity) ([]*pb.ExtractedEdge, error) {
	// Construct the list of known entities to guide the LLM
	var entityNames []string
	for _, e := range entities {
		entityNames = append(entityNames, e.Name)
	}

	systemPrompt := `You are an expert NLP system specialized in Knowledge Graphs.
Extract relationships (edges) between entities from the provided text.
Return ONLY a JSON array of objects with 'source', 'target', 'relation', 'fact', and an array of 'temporal' tags.
IMPORTANT: Only extract relationships where BOTH the source and target are in the provided list of known entities.`

	userPrompt := fmt.Sprintf("Known Entities: %v\n\nContent to analyze: %s", entityNames, content)

	// Generate JSON via LLM
	responseBytes, err := uc.llm.GenerateJSON(ctx, systemPrompt, userPrompt)
	if err != nil {
		log.Printf("LLM edge extraction failed: %v", err)
		return nil, fmt.Errorf("llm generation error: %w", err)
	}

	var extracted []struct {
		Source   string   `json:"source"`
		Target   string   `json:"target"`
		Relation string   `json:"relation"`
		Fact     string   `json:"fact"`
		Temporal []string `json:"temporal"`
	}
	if err := json.Unmarshal(responseBytes, &extracted); err != nil {
		log.Printf("Failed to parse LLM Edge JSON: %v. Raw Output: %s", err, string(responseBytes))
		return nil, fmt.Errorf("failed to parse extracted edges: %w", err)
	}

	// Map to Protobuf Domain Model
	var pbEdges []*pb.ExtractedEdge
	for _, e := range extracted {
		pbEdges = append(pbEdges, &pb.ExtractedEdge{
			Source:   e.Source,
			Target:   e.Target,
			Relation: e.Relation,
			Fact:     e.Fact,
			Temporal: e.Temporal,
		})
	}

	return pbEdges, nil
}
