// Package steps implements the BUILD_ONTOLOGY pipeline step.
// Derives ontology classes from extracted entity types.
package steps

import (
	"context"

	"vnp-memory/services/cognee-cognify/internal/domain"
	"vnp-memory/services/cognee-cognify/internal/usecase"
	"vnp-memory/services/cognee-cognify/internal/usecase/port"
)

// BuildOntologyStep derives ontology structure from extracted entity types.
type BuildOntologyStep struct {
	llmClient port.LLMClient
}

func NewBuildOntologyStep(llm port.LLMClient) *BuildOntologyStep {
	return &BuildOntologyStep{llmClient: llm}
}

func (s *BuildOntologyStep) Name() domain.PipelineStep { return domain.StepBuildOntology }

func (s *BuildOntologyStep) Execute(ctx context.Context, state *usecase.PipelineState) (*usecase.PipelineState, error) {
	if len(state.Nodes) == 0 {
		return state, nil // nothing to build ontology from
	}

	// Collect unique entity types
	typeSet := make(map[string]bool)
	for _, node := range state.Nodes {
		typeSet[node.Type] = true
	}

	// In production: use LLM to build class hierarchy
	// For now: passthrough — types are already labeled on nodes
	_ = typeSet

	return state, nil
}
