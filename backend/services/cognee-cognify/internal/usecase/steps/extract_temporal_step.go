// Package steps implements the EXTRACT_TEMPORAL_GRAPH pipeline step.
// Extracts time-anchored events and temporal relationships from content.
package steps

import (
	"context"
	"encoding/json"
	"fmt"

	"vnp-memory/services/cognee-cognify/internal/domain"
	"vnp-memory/services/cognee-cognify/internal/usecase"
	"vnp-memory/services/cognee-cognify/internal/usecase/port"
)

const temporalExtractionPrompt = `Extract temporal events and time-anchored relationships from the text.
Return JSON: {"events": [{"id": "...", "name": "...", "date": "YYYY-MM-DD or relative", "type": "EVENT|PERIOD|MILESTONE"}],
"temporal_edges": [{"source": "...", "target": "...", "label": "BEFORE|AFTER|DURING|CAUSES"}]}
Only extract explicitly stated temporal facts.`

// ExtractTemporalGraphStep extracts time-anchored events and relationships.
// Used in the TEMPORAL pipeline template where chronological context matters.
type ExtractTemporalGraphStep struct {
	llmClient port.LLMClient
	graphRepo port.GraphRepository
}

func NewExtractTemporalGraphStep(llm port.LLMClient, graph port.GraphRepository) *ExtractTemporalGraphStep {
	return &ExtractTemporalGraphStep{llmClient: llm, graphRepo: graph}
}

func (s *ExtractTemporalGraphStep) Name() domain.PipelineStep { return domain.StepExtractTemporalGraph }

func (s *ExtractTemporalGraphStep) Execute(ctx context.Context, state *usecase.PipelineState) (*usecase.PipelineState, error) {
	for _, chunk := range state.Chunks {
		resp, err := s.llmClient.Complete(ctx, temporalExtractionPrompt, chunk.Content)
		if err != nil {
			continue
		}

		var extracted struct {
			Events []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Date string `json:"date"`
				Type string `json:"type"`
			} `json:"events"`
			TemporalEdges []struct {
				Source string `json:"source"`
				Target string `json:"target"`
				Label  string `json:"label"`
			} `json:"temporal_edges"`
		}
		if err := json.Unmarshal([]byte(resp), &extracted); err != nil {
			continue
		}

		for _, e := range extracted.Events {
			node := domain.GraphNode{
				ID:   e.ID,
				Name: e.Name,
				Type: e.Type,
				Labels: []string{e.Type, "TemporalEvent"},
				Properties: map[string]any{
					"date":       e.Date,
					"dataset_id": state.DatasetID,
					"tenant_id":  state.TenantID,
				},
			}
			state.Nodes = append(state.Nodes, node)
			if _, err := s.graphRepo.UpsertEntity(ctx, state.TenantID, toEntity(node)); err != nil {
				return nil, fmt.Errorf("upsert temporal entity: %w", err)
			}
		}

		for _, te := range extracted.TemporalEdges {
			edge := domain.GraphEdge{
				Subject:   te.Source,
				Predicate: te.Label,
				Object:    te.Target,
				Properties: map[string]any{
					"temporal": true,
					"dataset_id": state.DatasetID,
				},
			}
			state.Edges = append(state.Edges, edge)
		}
	}

	return state, nil
}
