package usecase

import (
	"context"
	"fmt"

	"vnp-memory/services/cognee-cognify/internal/domain"
)

// ─── Request ──────────────────────────────────────────────────────────────────

// CognifyRequest is the input to StartCognifyUseCase.Execute().
type CognifyRequest struct {
	DatasetID string
	TenantID  string
	EntryIDs  []string
	NodeSets  []string // [NEW] CR-002 — NodeSet tags for multi-label assignment
	Config    domain.PipelineConfig
}

// ─── Pipeline State ───────────────────────────────────────────────────────────

// PipelineState is the shared state passed between pipeline steps.
type PipelineState struct {
	DatasetID   string
	TenantID    string
	EntryIDs    []string
	NodeSets    []string        // [NEW] CR-002 — carried through all steps
	RawContent  []string
	ContentType string
	Chunks      []Chunk
	Nodes       []domain.GraphNode
	Edges       []domain.GraphEdge
	Embeddings  map[string][]float32
	Options     domain.PipelineOptions
}

// Chunk is a unit of text after chunking.
type Chunk struct {
	Content  string
	Metadata map[string]any
}

// ─── Step Interface ───────────────────────────────────────────────────────────

// PipelineStep is a single step in the cognify pipeline.
type PipelineStep interface {
	Name() string
	Execute(ctx context.Context, state *PipelineState) (*PipelineState, error)
}

// ─── Ports ────────────────────────────────────────────────────────────────────

// GraphRepository persists nodes and edges with multi-label support.
type GraphRepository interface {
	UpsertNodeWithLabels(ctx context.Context, datasetID, tenantID string, node domain.GraphNode) error
}

// VectorRepository persists vector embeddings with payload.
type VectorRepository interface {
	UpsertPointPayload(ctx context.Context, vectorID string, payload map[string]any) error
}

// ─── Use Case ─────────────────────────────────────────────────────────────────

// StartCognifyUseCase runs the knowledge graph extraction pipeline.
type StartCognifyUseCase struct {
	graphRepo  GraphRepository
	vectorRepo VectorRepository
	steps      []PipelineStep
}

// NewStartCognifyUseCase creates a new StartCognifyUseCase with ordered steps.
func NewStartCognifyUseCase(graphRepo GraphRepository, vectorRepo VectorRepository, steps []PipelineStep) *StartCognifyUseCase {
	return &StartCognifyUseCase{graphRepo: graphRepo, vectorRepo: vectorRepo, steps: steps}
}

// Execute runs all pipeline steps in order, propagating NodeSets through state.
func (uc *StartCognifyUseCase) Execute(ctx context.Context, req CognifyRequest) error {
	state := &PipelineState{
		DatasetID: req.DatasetID,
		TenantID:  req.TenantID,
		EntryIDs:  req.EntryIDs,
		NodeSets:  req.NodeSets,   // [NEW] carry NodeSets through pipeline
		Options:   req.Config.Options,
	}

	var err error
	for _, step := range uc.steps {
		state, err = step.Execute(ctx, state)
		if err != nil {
			return fmt.Errorf("step %s: %w", step.Name(), err)
		}
	}
	return nil
}
