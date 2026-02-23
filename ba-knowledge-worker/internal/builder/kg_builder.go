package kg_builder

import (
	"context"
	"fmt"
	"log"
	"time"

	v32 "github.com/blcvn/backend/services/pkg/domain/v3.2"
)

// KGBuilder builds and updates Knowledge Graphs from document artifacts
type KGBuilder struct {
	graphRepo v32.GraphRepository
}

// NewKGBuilder creates a new KG builder
func NewKGBuilder(graphRepo v32.GraphRepository) *KGBuilder {
	return &KGBuilder{
		graphRepo: graphRepo,
	}
}

// BuildFromPRD builds initial Knowledge Graph from parsed PRD
func (b *KGBuilder) BuildFromPRD(ctx context.Context, prd *v32.StructuredPRD, docID string) (*v32.RequirementGraph, error) {
	log.Printf("[KGBuilder] Building KG from PRD for doc: %s", docID)

	graph := v32.NewRequirementGraph()
	// v32.RequirementGraph has ID field.
	graph.ID = fmt.Sprintf("kg-%s-%d", docID, time.Now().UnixNano())
	graph.DocumentID = docID

	// Build nodes from User Stories
	for _, story := range prd.UserStories {
		originalID := story.ID
		uniqueID := fmt.Sprintf("%s_%s", docID, originalID)

		node := v32.RequirementNode{
			ID:          uniqueID,
			DocumentID:  docID,
			ReferenceID: originalID,
			Type:        v32.ReqTypeUserStory,
			Summary:     story.IWant,
			Description: fmt.Sprintf("As a %s, I want %s, so that %s", story.AsA, story.IWant, story.SoThat),
			SourceID:    story.ID, // Keep original source reference
			Metadata: map[string]any{
				"feature_id": story.FeatureID,
				"priority":   story.Priority,
			},
		}
		graph.AddNode(node)
	}

	// Build nodes from Features
	for _, feature := range prd.Features {
		originalID := feature.ID
		uniqueID := fmt.Sprintf("%s_%s", docID, originalID)

		node := v32.RequirementNode{
			ID:          uniqueID,
			DocumentID:  docID,
			ReferenceID: originalID,
			Type:        v32.ReqTypeFunctional,
			Summary:     feature.Name,
			Description: feature.Description,
			SourceID:    feature.ID,
			Metadata: map[string]any{
				"priority": feature.Priority,
			},
		}
		graph.AddNode(node)
	}

	// Build edges: Feature -> UserStory
	for _, story := range prd.UserStories {
		if story.FeatureID != "" {
			uniqueSourceID := fmt.Sprintf("%s_%s", docID, story.FeatureID)
			uniqueTargetID := fmt.Sprintf("%s_%s", docID, story.ID)
			edgeID := fmt.Sprintf("edge_%s_%s_%s", docID, story.FeatureID, story.ID)

			edge := v32.DependencyEdge{
				ID:         edgeID,
				DocumentID: docID,
				SourceID:   uniqueSourceID,
				TargetID:   uniqueTargetID,
				Type:       v32.DepTypeRefines,
				Reason:     "Feature contains User Story",
			}
			graph.AddEdge(edge)
		}
	}

	// Build nodes from Entities
	for _, entity := range prd.Entities {
		originalID := entity.ID
		uniqueID := fmt.Sprintf("%s_%s", docID, originalID)

		node := v32.RequirementNode{
			ID:          uniqueID,
			DocumentID:  docID,
			ReferenceID: originalID,
			Type:        v32.ReqTypeEntity,
			Summary:     entity.Name,
			Description: entity.Description,
			SourceID:    entity.ID,
		}
		graph.AddNode(node)
	}

	// NEW: Build nodes from Personas
	for _, persona := range prd.Personas {
		originalID := persona.ID
		uniqueID := fmt.Sprintf("%s_%s", docID, originalID)

		node := v32.RequirementNode{
			ID:          uniqueID,
			DocumentID:  docID,
			ReferenceID: originalID,
			Type:        v32.ReqTypePersona,
			Summary:     persona.Name,
			Description: persona.Role,
			SourceID:    persona.ID,
			Metadata: map[string]any{
				"role":            persona.Role,
				"goals":           persona.Goals,
				"pain_points":     persona.PainPoints,
				"technical_level": persona.TechnicalLevel,
			},
		}
		graph.AddNode(node)
	}

	// NEW: Build nodes from Integrations
	for _, integration := range prd.Integrations {
		originalID := integration.ID
		uniqueID := fmt.Sprintf("%s_%s", docID, originalID)

		node := v32.RequirementNode{
			ID:          uniqueID,
			DocumentID:  docID,
			ReferenceID: originalID,
			Type:        v32.ReqTypeAPI,
			Summary:     integration.SystemName,
			Description: integration.Purpose,
			SourceID:    integration.ID,
			Metadata: map[string]any{
				"type":      integration.Type,
				"direction": integration.Direction,
				"status":    integration.Status,
			},
		}
		graph.AddNode(node)
	}

	// NEW: Build nodes from Business Rules
	for _, rule := range prd.BusinessRules {
		originalID := rule.ID
		uniqueID := fmt.Sprintf("%s_%s", docID, originalID)

		node := v32.RequirementNode{
			ID:          uniqueID,
			DocumentID:  docID,
			ReferenceID: originalID,
			Type:        v32.ReqTypeBusinessRule,
			Summary:     rule.Name,
			Description: rule.Description,
			SourceID:    rule.ID,
			Metadata: map[string]any{
				"severity":   rule.Severity,
				"applies_to": rule.AppliesTo,
			},
		}
		graph.AddNode(node)
	}

	// Add Metadata to Graph Metadata
	if graph.Metadata == nil {
		graph.Metadata = make(map[string]any)
	}
	graph.Metadata["product_name"] = prd.Metadata.ProductName
	graph.Metadata["version"] = prd.Metadata.Version
	graph.Metadata["status"] = prd.Metadata.Status
	graph.Metadata["vision"] = prd.ProductOverview.Vision

	log.Printf("[KGBuilder] Built KG with %d nodes and %d edges", len(graph.Nodes), len(graph.Edges))
	return graph, nil
}

// UpdateFromIndex updates KG with Use Case nodes and metadata from URD Index
func (b *KGBuilder) UpdateFromIndex(ctx context.Context, graph *v32.RequirementGraph, index *v32.URDIndex) error {
	log.Printf("[KGBuilder] Updating KG from Index")

	docID := graph.DocumentID
	if docID == "" {
		// Fallback to metadata if empty (legacy support during migration)
		if val, ok := graph.Metadata["document_id"].(string); ok {
			docID = val
		} else {
			return fmt.Errorf("document_id missing in graph")
		}
	}

	// 1. Add/Update Human Actors
	for _, actor := range index.HumanActors {
		originalID := actor.ID
		uniqueID := fmt.Sprintf("%s_%s", docID, originalID)

		if _, exists := graph.GetNode(uniqueID); !exists {
			node := v32.RequirementNode{
				ID:          uniqueID,
				DocumentID:  docID,
				ReferenceID: originalID,
				Type:        v32.ReqTypePersona,
				Summary:     actor.Name,
				Description: actor.Role,
				Metadata: map[string]any{
					"responsibilities": actor.Description,
				},
			}
			graph.AddNode(node)
		}
	}

	// 2. Add/Update System Actors
	for _, actor := range index.SystemActors {
		originalID := actor.ID
		uniqueID := fmt.Sprintf("%s_%s", docID, originalID)

		if _, exists := graph.GetNode(uniqueID); !exists {
			node := v32.RequirementNode{
				ID:          uniqueID,
				DocumentID:  docID,
				ReferenceID: originalID,
				Type:        v32.ReqTypeAPI,
				Summary:     actor.Name,
				Description: actor.Description,
				Metadata: map[string]any{
					"system_type": actor.Type,
				},
			}
			graph.AddNode(node)
		}
	}

	// 3. Add UC nodes and US->UC mapping
	for _, mapping := range index.USToUCMapping {
		for _, ucID := range mapping.UseCaseIDs {
			// ucID is original ID (e.g., UC-001)
			originalUCID := ucID
			uniqueUCID := fmt.Sprintf("%s_%s", docID, originalUCID)

			if _, exists := graph.GetNode(uniqueUCID); !exists {
				node := v32.RequirementNode{
					ID:          uniqueUCID,
					DocumentID:  docID,
					ReferenceID: originalUCID,
					Type:        v32.ReqTypeUseCase,
					Summary:     originalUCID, // Initial summary is ID
					Description: mapping.MappingNote,
					SourceID:    mapping.UserStoryID,
				}
				graph.AddNode(node)
			}

			// Add edge: US -> UC
			// US node should exist with unique ID
			originalUSID := mapping.UserStoryID
			uniqueUSID := fmt.Sprintf("%s_%s", docID, originalUSID)

			edgeID := fmt.Sprintf("edge_%s_%s_%s", docID, originalUSID, originalUCID)

			edgeExists := false
			for _, e := range graph.Edges {
				if e.ID == edgeID {
					edgeExists = true
					break
				}
			}

			if !edgeExists {
				edge := v32.DependencyEdge{
					ID:         edgeID,
					DocumentID: docID,
					SourceID:   uniqueUSID,
					TargetID:   uniqueUCID,
					Type:       v32.DepTypeRefines,
					Reason:     mapping.MappingNote,
				}
				graph.AddEdge(edge)
			}
		}
	}

	// 4. Update UC nodes with Summary Table info
	for _, summary := range index.UseCaseSummaryTable {
		originalID := summary.ID
		uniqueID := fmt.Sprintf("%s_%s", docID, originalID)

		// Find by Unique ID
		if node, exists := graph.GetNode(uniqueID); exists {
			node.Summary = summary.Name
			if node.Metadata == nil {
				node.Metadata = make(map[string]any)
			}
			node.Metadata["trigger"] = summary.Trigger
			node.Metadata["priority"] = summary.Priority
			node.Metadata["primary_actor"] = summary.PrimaryActor
			node.ReferenceID = originalID

			// Add edge: Actor -> UC
			if summary.PrimaryActor != "" {
				originalActorID := summary.PrimaryActor
				uniqueActorID := fmt.Sprintf("%s_%s", docID, originalActorID)
				edgeID := fmt.Sprintf("edge_%s_%s_%s", docID, originalActorID, originalID)

				edgeExists := false
				for _, e := range graph.Edges {
					if e.ID == edgeID {
						edgeExists = true
						break
					}
				}

				if !edgeExists {
					edge := v32.DependencyEdge{
						ID:         edgeID,
						DocumentID: docID,
						SourceID:   uniqueActorID,
						TargetID:   uniqueID,
						Type:       v32.DepTypeRelatesTo,
						Reason:     "Actor performs Use Case",
					}
					graph.AddEdge(edge)
				}
			}
		}
	}

	// 5. Add/Update Integrations
	for _, integ := range index.IntegrationTouchpoints {
		originalID := integ.ID
		uniqueID := fmt.Sprintf("%s_%s", docID, originalID)

		if _, exists := graph.GetNode(uniqueID); !exists {
			node := v32.RequirementNode{
				ID:          uniqueID,
				DocumentID:  docID,
				ReferenceID: originalID,
				Type:        v32.ReqTypeAPI,
				Summary:     integ.Name,
				Description: integ.Description,
				Metadata: map[string]any{
					"type":      integ.Type,
					"direction": integ.Direction,
				},
			}
			graph.AddNode(node)
		}
	}

	// 6. Add/Update Data Entities
	for _, entity := range index.DataEntities {
		originalID := entity.ID
		uniqueID := fmt.Sprintf("%s_%s", docID, originalID)

		if _, exists := graph.GetNode(uniqueID); !exists {
			node := v32.RequirementNode{
				ID:          uniqueID,
				DocumentID:  docID,
				ReferenceID: originalID,
				Type:        v32.ReqTypeEntity,
				Summary:     entity.Name,
				Description: entity.Description,
				Metadata: map[string]any{
					"attributes": entity.KeyAttributes,
				},
			}
			graph.AddNode(node)
		}
	}

	log.Printf("[KGBuilder] Updated KG: %d nodes, %d edges", len(graph.Nodes), len(graph.Edges))
	return nil
}

// UpdateFromOutline updates KG with detailed UC information from Outline
func (b *KGBuilder) UpdateFromOutline(ctx context.Context, graph *v32.RequirementGraph, outline *v32.URDOutline) error {
	log.Printf("[KGBuilder] Updating KG from Outline")

	docID := graph.DocumentID
	if docID == "" {
		if val, ok := graph.Metadata["document_id"].(string); ok {
			docID = val
		} else {
			return fmt.Errorf("document_id missing in graph metadata")
		}
	}

	// Update existing UC nodes with detailed info from Outline
	for _, uc := range outline.UseCases {
		originalID := uc.ID
		uniqueID := fmt.Sprintf("%s_%s", docID, originalID)

		// Find and update existing node
		if node, exists := graph.GetNode(uniqueID); exists {
			node.Summary = uc.Name
			node.Description = uc.Description
			if node.Metadata == nil {
				node.Metadata = make(map[string]any)
			}
			node.Metadata["preconditions"] = uc.Preconditions
			node.Metadata["postconditions"] = uc.Postconditions
			node.Metadata["main_flow"] = uc.MainFlow
			node.Metadata["priority"] = uc.Priority
			node.Metadata["primary_actor"] = uc.PrimaryActor

			// Ensure edge from Actor to UC
			if uc.PrimaryActor != "" {
				originalActorID := uc.PrimaryActor
				uniqueActorID := fmt.Sprintf("%s_%s", docID, originalActorID)
				edgeID := fmt.Sprintf("edge_%s_%s_%s", docID, originalActorID, originalID)

				edgeExists := false
				for _, e := range graph.Edges {
					if e.ID == edgeID {
						edgeExists = true
						break
					}
				}

				if !edgeExists {
					edge := v32.DependencyEdge{
						ID:         edgeID,
						DocumentID: docID,
						SourceID:   uniqueActorID,
						TargetID:   uniqueID,
						Type:       v32.DepTypeRelatesTo,
						Reason:     "Primary actor for Use Case",
					}
					graph.AddEdge(edge)
				}
			}
		} else {
			// Create it if it doesn't exist
			node := v32.RequirementNode{
				ID:          uniqueID,
				DocumentID:  docID,
				ReferenceID: originalID,
				Type:        v32.ReqTypeUseCase,
				Summary:     uc.Name,
				Description: uc.Description,
				Metadata: map[string]any{
					"preconditions":  uc.Preconditions,
					"postconditions": uc.Postconditions,
					"main_flow":      uc.MainFlow,
					"priority":       uc.Priority,
					"primary_actor":  uc.PrimaryActor,
				},
			}
			graph.AddNode(node)
		}
	}

	log.Printf("[KGBuilder] Updated KG from Outline: %d nodes, %d edges", len(graph.Nodes), len(graph.Edges))
	return nil
}
