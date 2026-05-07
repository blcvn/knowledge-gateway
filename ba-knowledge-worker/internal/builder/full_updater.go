package kg_builder

import (
	"context"
	"fmt"
	"log"
	"strings"

	v32 "github.com/blcvn/ba-shared-libs/pkg/domain/v3.2"
)

// UpdateFromFull updates the KG based on the Full URD content
func (b *KGBuilder) UpdateFromFull(ctx context.Context, graph *v32.RequirementGraph, fullURD *v32.URDFull) error {
	log.Printf("[KGBuilder] Updating KG from Full URD with %d UseCases, %d APIs",
		len(fullURD.UseCases), len(fullURD.APISpecifications))

	docID := graph.DocumentID
	if docID == "" {
		if val, ok := graph.Metadata["document_id"].(string); ok {
			docID = val
		} else {
			return fmt.Errorf("document_id missing in graph metadata")
		}
	}

	// 1. Update Use Cases (enrich existing nodes)
	for _, uc := range fullURD.UseCases {
		originalID := uc.ID
		uniqueID := fmt.Sprintf("%s_%s", docID, originalID)

		// GetNode
		node, exists := graph.GetNode(uniqueID)
		if !exists {
			log.Printf("[KGBuilder] Warning: UseCase %s not found in KG, adding new node", uniqueID)
			// Ensure we add it if it doesn't exist (though it should from Outline)
			newNode := v32.RequirementNode{
				ID:          uniqueID,
				DocumentID:  docID,
				ReferenceID: originalID,
				Type:        v32.ReqTypeUseCase,
				Summary:     uc.Name,
			}
			graph.AddNode(newNode)
		} else {
			// Update description/metadata if available
			if uc.Description != "" {
				node.Description = uc.Description
				node.ReferenceID = originalID // Ensure ReferenceID is set
			}
			// Update other metadata from FullUseCase if needed
		}
	}

	// 2. Create API Nodes and Edges
	for _, apiSpec := range fullURD.APISpecifications {
		for _, endpoint := range apiSpec.Endpoints {
			// ID: API-METHOD-PATH or use embedded ID if robust
			originalAPIID := fmt.Sprintf("API-%s-%s", endpoint.Method, sanitizePath(endpoint.Path))
			if endpoint.ID != "" {
				originalAPIID = endpoint.ID
			}
			uniqueAPIID := fmt.Sprintf("%s_%s", docID, originalAPIID)

			apiNode := v32.RequirementNode{
				ID:          uniqueAPIID,
				DocumentID:  docID,
				ReferenceID: originalAPIID,
				Type:        v32.ReqTypeAPI,
				Summary:     fmt.Sprintf("%s %s", endpoint.Method, endpoint.Path),
				Description: endpoint.Description,
				Metadata: map[string]interface{}{
					"request":  endpoint.Request,
					"response": endpoint.Response,
				},
			}
			graph.AddNode(apiNode)

			// Link API to UseCases
			if endpoint.UseCaseID != "" {
				originalUCID := endpoint.UseCaseID
				uniqueUCID := fmt.Sprintf("%s_%s", docID, originalUCID)

				// Edge
				edgeID := fmt.Sprintf("edge_%s_%s_%s", docID, originalUCID, originalAPIID)

				edge := v32.DependencyEdge{
					ID:         edgeID,
					DocumentID: docID,
					SourceID:   uniqueUCID,
					TargetID:   uniqueAPIID,
					Type:       v32.DepTypeRelatesTo,
				}
				graph.AddEdge(edge)
			}
		}
	}

	// 3. Create Data Model Nodes (from Entities in Full URD if available)
	for _, entity := range fullURD.Entities {
		originalID := entity.EntityID
		uniqueID := fmt.Sprintf("%s_%s", docID, originalID)

		modelNode := v32.RequirementNode{
			ID:          uniqueID,
			DocumentID:  docID,
			ReferenceID: originalID,
			Type:        v32.ReqTypeEntity,
			Summary:     entity.Name,
			// Description: entity.Description, // OutlineEntity struct in urd_full.go (reference) had Description?
			// My v32/urd_outline.go OutlineEntity has Description?
			// Let's check v32.OutlineEntity.
			// It has EntityID, Name, Description, KeyAttributes.
		}
		if entity.Description != "" {
			modelNode.Description = entity.Description
		}

		graph.AddNode(modelNode)
	}

	return nil
}

func sanitizePath(path string) string {
	return strings.ReplaceAll(strings.ReplaceAll(path, "/", "_"), "{", "")
}
