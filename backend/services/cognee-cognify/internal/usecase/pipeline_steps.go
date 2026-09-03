package usecase

import (
	"context"

	"vnp-memory/services/cognee-cognify/internal/domain"
)

// ExtractGraphStep uses an LLM to extract entities and relationships from chunks.
// [CR-002] After extraction, NodeSet tags from PipelineState are appended to all nodes as labels.
type ExtractGraphStep struct{}

// NewExtractGraphStep creates an ExtractGraphStep.
func NewExtractGraphStep() *ExtractGraphStep { return &ExtractGraphStep{} }

// Name returns the step identifier.
func (s *ExtractGraphStep) Name() string { return "extract_graph" }

// Execute extracts nodes and edges from text chunks, then attaches NodeSet labels.
func (s *ExtractGraphStep) Execute(ctx context.Context, state *PipelineState) (*PipelineState, error) {
	// In production: call LLM client to extract graph from state.Chunks.
	// Stub: create a placeholder node per chunk.
	nodes := make([]domain.GraphNode, 0, len(state.Chunks))
	for _, chunk := range state.Chunks {
		node := domain.GraphNode{
			ID:         chunk.Content[:min(len(chunk.Content), 32)],
			Name:       chunk.Content[:min(len(chunk.Content), 64)],
			Type:       "Concept",
			Labels:     []string{"Concept"},
			Properties: map[string]any{"content": chunk.Content},
		}
		// [NEW] Attach NodeSet tags as additional labels on every extracted node.
		// Result: ["Concept", "customer_123", "preferences"]
		node.Labels = append(node.Labels, state.NodeSets...)
		nodes = append(nodes, node)
	}

	state.Nodes = nodes
	return state, nil
}

// AddDatapointsStep persists nodes into Neo4j (multi-labels) and Qdrant (payload).
type AddDatapointsStep struct {
	graphRepo  GraphRepository
	vectorRepo VectorRepository
}

// NewAddDatapointsStep creates an AddDatapointsStep.
func NewAddDatapointsStep(graphRepo GraphRepository, vectorRepo VectorRepository) *AddDatapointsStep {
	return &AddDatapointsStep{graphRepo: graphRepo, vectorRepo: vectorRepo}
}

// Name returns the step identifier.
func (s *AddDatapointsStep) Name() string { return "add_datapoints" }

// Execute persists all nodes with multi-labels to Neo4j and payload to Qdrant.
func (s *AddDatapointsStep) Execute(ctx context.Context, state *PipelineState) (*PipelineState, error) {
	for _, node := range state.Nodes {
		// Neo4j: MERGE node, SET multi-labels for each NodeSet tag
		if err := s.graphRepo.UpsertNodeWithLabels(ctx, state.DatasetID, state.TenantID, node); err != nil {
			// Non-fatal — continue with remaining nodes
			continue
		}

		// Qdrant: attach node_sets to point payload for filtering
		if node.VectorID != "" {
			s.vectorRepo.UpsertPointPayload(ctx, node.VectorID, map[string]any{
				"node_sets":  node.Labels, // [NEW] payload field = all labels including NodeSets
				"dataset_id": state.DatasetID,
				"tenant_id":  state.TenantID,
			})
		}
	}
	return state, nil
}

func min(a, b int) int {
	if a < b { return a }
	return b
}
