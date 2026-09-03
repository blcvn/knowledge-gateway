// Package steps implements the SUMMARIZE_COMMUNITY pipeline step.
// Uses LLM to generate summaries for detected communities.
package steps

import (
	"context"
	"fmt"

	"vnp-memory/services/cognee-cognify/internal/domain"
	"vnp-memory/services/cognee-cognify/internal/usecase"
	"vnp-memory/services/cognee-cognify/internal/usecase/port"
)

const communitySummaryPrompt = `You are analyzing a knowledge graph community (cluster of related entities).
Given the entities and relationships in this community, write a concise 2-3 sentence summary
that describes: what this community is about, the key entities, and their main relationships.`

// SummarizeCommunityStep uses LLM to generate natural-language summaries for communities.
type SummarizeCommunityStep struct {
	llmClient port.LLMClient
	graphRepo port.GraphRepository
}

func NewSummarizeCommunityStep(llm port.LLMClient, graph port.GraphRepository) *SummarizeCommunityStep {
	return &SummarizeCommunityStep{llmClient: llm, graphRepo: graph}
}

func (s *SummarizeCommunityStep) Name() domain.PipelineStep { return domain.StepSummarizeCommunity }

func (s *SummarizeCommunityStep) Execute(ctx context.Context, state *usecase.PipelineState) (*usecase.PipelineState, error) {
	if len(state.Nodes) == 0 {
		return state, nil
	}

	// Group nodes by community (in production: communities assigned by DetectCommunityStep)
	// For each community: build context and call LLM for summary

	// Example (stub — production uses real community IDs from Neo4j):
	communityContext := buildCommunityContext(state.Nodes, state.Edges)
	if communityContext == "" {
		return state, nil
	}

	summary, err := s.llmClient.Complete(ctx, communitySummaryPrompt, communityContext)
	if err != nil {
		// Non-fatal — summarization failure doesn't block pipeline
		return state, nil
	}

	// Store community summary as a Community node in Neo4j
	community := &domain.Community{
		ID:      fmt.Sprintf("community-%s", state.DatasetID),
		Summary: summary,
		Members: extractNodeIDs(state.Nodes),
	}
	_ = s.graphRepo.UpsertCommunity(ctx, state.TenantID, community)

	return state, nil
}

func buildCommunityContext(nodes []domain.GraphNode, edges []domain.GraphEdge) string {
	if len(nodes) == 0 {
		return ""
	}
	context := "Entities:\n"
	for _, n := range nodes {
		context += fmt.Sprintf("- %s (%s)\n", n.Name, n.Type)
	}
	if len(edges) > 0 {
		context += "\nRelationships:\n"
		for _, e := range edges {
			context += fmt.Sprintf("- %s %s %s\n", e.Subject, e.Predicate, e.Object)
		}
	}
	return context
}

func extractNodeIDs(nodes []domain.GraphNode) []string {
	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	return ids
}
