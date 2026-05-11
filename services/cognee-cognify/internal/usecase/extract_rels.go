package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/domain"
	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/usecase/port"
)

// llmRelationshipResponse matches the JSON schema for relationship extraction.
type llmRelationshipResponse struct {
	Relationships []struct {
		Source   string  `json:"source"`
		Target   string  `json:"target"`
		Relation string  `json:"relation"`
		Weight   float64 `json:"weight"`
	} `json:"relationships"`
}

// ExtractRelationshipsStage extracts entity relationships from chunks via LLM.
type ExtractRelationshipsStage struct {
	llm    port.LLMClient
	logger *slog.Logger
}

func NewExtractRelationshipsStage(llm port.LLMClient, logger *slog.Logger) *ExtractRelationshipsStage {
	return &ExtractRelationshipsStage{llm: llm, logger: logger.With("stage", "extract_rels")}
}

func (s *ExtractRelationshipsStage) Name() domain.StageType { return domain.StageExtractRelationships }

func (s *ExtractRelationshipsStage) Execute(ctx context.Context, job *domain.CognifyJob, state *PipelineState) error {
	// Build entity name → ID index for fast lookup
	entityIndex := make(map[string]*domain.Entity, len(state.Entities))
	for _, e := range state.Entities {
		entityIndex[strings.ToLower(e.Name)] = e
	}

	var allRels []*domain.Relationship

	for _, chunk := range state.Chunks {
		// Build entity context for the prompt
		var entityNames []string
		for _, e := range state.Entities {
			if e.SourceChunk == chunk.ID {
				entityNames = append(entityNames, e.Name)
			}
		}
		if len(entityNames) == 0 {
			continue
		}

		userPrompt := fmt.Sprintf("Entities in context: %s\n\nText:\n%s",
			strings.Join(entityNames, ", "), chunk.Text)

		resp := &llmRelationshipResponse{}
		err := s.llm.CompleteStructured(ctx, relExtractionPrompt, userPrompt, resp)
		if err != nil {
			s.logger.Warn("relationship extraction failed", "chunk_id", chunk.ID, "error", err)
			job.Metrics.LLMCallsTotal++
			continue
		}
		job.Metrics.LLMCallsTotal++

		for _, r := range resp.Relationships {
			sourceEntity := entityIndex[strings.ToLower(r.Source)]
			targetEntity := entityIndex[strings.ToLower(r.Target)]
			if sourceEntity == nil || targetEntity == nil {
				continue
			}

			weight := r.Weight
			if weight <= 0 || weight > 1.0 {
				weight = 0.5
			}

			rel := domain.NewRelationship(
				job.ID, sourceEntity.ID, targetEntity.ID,
				r.Relation, weight, chunk.ID,
			)
			allRels = append(allRels, rel)
		}
	}

	state.Relationships = allRels
	job.Metrics.RelationshipsExtracted = len(allRels)
	s.logger.Info("relationship extraction complete", "relationships", len(allRels))
	return nil
}

const relExtractionPrompt = `You are an expert relationship extraction system.
Given entities and text, extract relationships. Return JSON:
{
  "relationships": [
    {"source": "<entity_name>", "target": "<entity_name>", "relation": "<relationship_type>", "weight": <0.0-1.0>}
  ]
}
Use lowercase snake_case for relation types (e.g. works_for, located_in, founded_by).
weight is confidence: 1.0 = certain, 0.5 = plausible.`
