package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	pb "vnp-memory/services/graphiti-knowledge/internal/adapter/grpc/pb"
)

// ResolveEntities performs entity deduplication against the existing Graph.
// It compares new extracted entities with previously stored entities in the same group,
// using the LLM to decide whether to MERGE, CREATE_NEW, or DISCARD.
func (uc *ExtractorUseCase) ResolveEntities(ctx context.Context, entities []*pb.ExtractedEntity, groupID string) ([]*pb.Resolution, error) {
	// 1. In a real system, we'd query Neo4j for existing entities in this group:
	// existingEntities := fetchFromNeo4j(groupID)
	
	// For demonstration of the AI logic, we assume we fetched a list of known entities.
	// We'll mock two existing entities.
	knownEntitiesJSON := `[
		{"id": "ent_123", "name": "Apple", "label": "Organization", "summary": "A tech company."},
		{"id": "ent_124", "name": "Tim Cook", "label": "Person", "summary": "CEO of Apple."}
	]`

	// 2. Prepare the batch to send to the LLM
	var newEntitiesJSON string
	for i, e := range entities {
		newEntitiesJSON += fmt.Sprintf(`{"index": %d, "name": "%s", "label": "%s", "summary": "%s"}`+"\n", i, e.Name, e.Label, e.Summary)
	}

	systemPrompt := `You are an Entity Resolution Expert for Knowledge Graphs.
You are given a list of EXISTING entities and a list of NEW extracted entities.
For each NEW entity, determine if it refers to the EXACT SAME real-world concept as an EXISTING entity.
Output a JSON array of resolution objects. Each object must contain:
- 'index': the index of the new entity
- 'decision': strictly one of ["MERGE", "CREATE_NEW", "DISCARD"]
- 'existing_entity_id': the ID of the matched existing entity (only if decision is MERGE)
- 'confidence': float between 0.0 and 1.0`

	userPrompt := fmt.Sprintf("EXISTING ENTITIES:\n%s\n\nNEW ENTITIES:\n%s", knownEntitiesJSON, newEntitiesJSON)

	// 3. Invoke LLM for resolution decisions
	responseBytes, err := uc.llm.GenerateJSON(ctx, systemPrompt, userPrompt)
	if err != nil {
		log.Printf("LLM resolution failed: %v", err)
		return nil, fmt.Errorf("llm generation error during resolution: %w", err)
	}

	var aiDecisions []struct {
		Index            int     `json:"index"`
		Decision         string  `json:"decision"`
		ExistingEntityID string  `json:"existing_entity_id"`
		Confidence       float64 `json:"confidence"`
	}
	if err := json.Unmarshal(responseBytes, &aiDecisions); err != nil {
		return nil, fmt.Errorf("failed to parse AI resolution output: %w", err)
	}

	// 4. Map the AI decisions back to our Protobuf response format
	resolutions := make([]*pb.Resolution, len(entities))
	for _, decision := range aiDecisions {
		if decision.Index < 0 || decision.Index >= len(entities) {
			continue // Safety check
		}
		
		res := &pb.Resolution{
			ExtractedEntity: entities[decision.Index],
			Decision:        decision.Decision,
			Confidence:      decision.Confidence,
		}
		
		if decision.Decision == "MERGE" {
			res.ExistingEntityId = decision.ExistingEntityID
		}
		
		resolutions[decision.Index] = res
	}

	// Fallback for any unmapped entities (failsafe against LLM hallucinating indices)
	for i, r := range resolutions {
		if r == nil {
			resolutions[i] = &pb.Resolution{
				ExtractedEntity: entities[i],
				Decision:        "CREATE_NEW",
				Confidence:      1.0,
			}
		}
	}

	return resolutions, nil
}
