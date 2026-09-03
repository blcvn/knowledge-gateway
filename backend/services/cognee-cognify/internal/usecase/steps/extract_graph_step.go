// Package steps implements the EXTRACT_GRAPH pipeline step.
// Uses LLM to extract entities and relationships from text chunks.
package steps

import (
	"context"
	"encoding/json"
	"fmt"

	"vnp-memory/services/cognee-cognify/internal/domain"
	"vnp-memory/services/cognee-cognify/internal/usecase"
	"vnp-memory/services/cognee-cognify/internal/usecase/port"
)

const graphExtractionPrompt = `Extract entities and relationships from the text.
Return JSON: {"nodes": [{"id": "...", "name": "...", "type": "..."}], "edges": [{"source": "...", "target": "...", "label": "..."}]}
Only extract explicitly stated facts. Use concise relationship labels.`

// ExtractGraphStep uses LLM to extract entities and relationships from chunks.
type ExtractGraphStep struct {
	llmClient  port.LLMClient
	graphRepo  port.GraphRepository
	embedder   port.EmbedderClient
}

func NewExtractGraphStep(llm port.LLMClient, graph port.GraphRepository, embedder port.EmbedderClient) *ExtractGraphStep {
	return &ExtractGraphStep{llmClient: llm, graphRepo: graph, embedder: embedder}
}

func (s *ExtractGraphStep) Name() domain.PipelineStep { return domain.StepExtractGraph }

func (s *ExtractGraphStep) Execute(ctx context.Context, state *usecase.PipelineState) (*usecase.PipelineState, error) {
	prompt := graphExtractionPrompt
	if state.Options.CustomPrompt != "" {
		prompt = state.Options.CustomPrompt
	}

	for _, chunk := range state.Chunks {
		resp, err := s.llmClient.Complete(ctx, prompt, chunk.Content)
		if err != nil {
			continue // skip bad chunks — best effort
		}

		var extracted struct {
			Nodes []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"nodes"`
			Edges []struct {
				Source string `json:"source"`
				Target string `json:"target"`
				Label  string `json:"label"`
			} `json:"edges"`
		}
		if err := json.Unmarshal([]byte(resp), &extracted); err != nil {
			continue
		}

		for _, n := range extracted.Nodes {
			labels := append([]string{n.Type}, state.NodeSets...)
			node := domain.GraphNode{
				ID:     n.ID,
				Name:   n.Name,
				Type:   n.Type,
				Labels: labels,
				Properties: map[string]any{
					"dataset_id": state.DatasetID,
					"tenant_id":  state.TenantID,
				},
			}
			state.Nodes = append(state.Nodes, node)
		}

		for _, e := range extracted.Edges {
			edge := domain.GraphEdge{
				Subject:   e.Source,
				Predicate: e.Label,
				Object:    e.Target,
				Properties: map[string]any{"dataset_id": state.DatasetID},
			}
			state.Edges = append(state.Edges, edge)
		}
	}

	// Persist extracted graph to Neo4j
	for _, node := range state.Nodes {
		if _, err := s.graphRepo.UpsertEntity(ctx, state.TenantID, toEntity(node)); err != nil {
			return nil, fmt.Errorf("upsert entity: %w", err)
		}
	}

	return state, nil
}

func toEntity(n domain.GraphNode) *domain.Entity {
	return &domain.Entity{
		ID:         n.ID,
		Name:       n.Name,
		Type:       n.Type,
		Properties: n.Properties,
	}
}
