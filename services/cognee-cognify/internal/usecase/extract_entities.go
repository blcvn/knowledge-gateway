package usecase

import (
	"context"
	"log/slog"

	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/domain"
	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/usecase/port"
)

// llmEntityResponse matches the JSON schema sent to the LLM.
type llmEntityResponse struct {
	Entities []struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Description string `json:"description"`
	} `json:"entities"`
}

// ExtractEntitiesStage extracts named entities from chunks via LLM NER.
type ExtractEntitiesStage struct {
	llm    port.LLMClient
	logger *slog.Logger
}

func NewExtractEntitiesStage(llm port.LLMClient, logger *slog.Logger) *ExtractEntitiesStage {
	return &ExtractEntitiesStage{llm: llm, logger: logger.With("stage", "extract_entities")}
}

func (s *ExtractEntitiesStage) Name() domain.StageType { return domain.StageExtractEntities }

func (s *ExtractEntitiesStage) Execute(ctx context.Context, job *domain.CognifyJob, state *PipelineState) error {
	var allEntities []*domain.Entity

	for _, chunk := range state.Chunks {
		resp := &llmEntityResponse{}
		err := s.llm.CompleteStructured(ctx, entityExtractionPrompt, chunk.Text, resp)
		if err != nil {
			s.logger.Warn("entity extraction failed for chunk", "chunk_id", chunk.ID, "error", err)
			job.Metrics.LLMCallsTotal++
			continue
		}
		job.Metrics.LLMCallsTotal++

		for _, e := range resp.Entities {
			entity := domain.NewEntity(
				job.ID,
				e.Name,
				domain.EntityType(e.Type),
				e.Description,
				chunk.ID,
			)
			allEntities = append(allEntities, entity)
		}
	}

	state.Entities = allEntities
	job.Metrics.EntitiesExtracted = len(allEntities)
	s.logger.Info("entity extraction complete", "entities", len(allEntities), "chunks_processed", len(state.Chunks))
	return nil
}

const entityExtractionPrompt = `You are an expert Named Entity Recognition (NER) system.
Extract all entities from the given text. Return JSON:
{
  "entities": [
    {"name": "<entity name>", "type": "<PERSON|ORGANIZATION|LOCATION|EVENT|CONCEPT|TECHNOLOGY|PRODUCT|DATE|OTHER>", "description": "<brief description>"}
  ]
}
Only extract clearly identifiable entities. Be concise.`
