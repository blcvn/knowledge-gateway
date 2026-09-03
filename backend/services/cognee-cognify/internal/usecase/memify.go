// Package usecase implements the Memify non-destructive graph enrichment use case.
// TASK-CE-006: Memify UseCase (SOL-001 §2.1–§2.8)
//
// Memify differs from StartCognify:
//   - Reads existing graph (read-only step)
//   - Derives NEW facts via LLM without deleting existing nodes/edges
//   - Upsert-only (MERGE in Neo4j, no DELETE)
//   - Re-embeds only newly derived triplets
//   - Runs fully in background, tracks progress via cognee_pipeline_runs
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"vnp-memory/services/cognee-cognify/internal/domain"
	"vnp-memory/services/cognee-cognify/internal/usecase/port"
)

const (
	defaultMemifyBatchSize   = 50
	memifyDefaultDeriveFacts = true
	memifyDefaultEmbed       = true
)

// MemifyRequest is the input for MemifyUseCase.Execute.
type MemifyRequest struct {
	DatasetID uuid.UUID
	TenantID  string
	Config    MemifyConfig
}

// MemifyConfig holds per-run tuning parameters.
type MemifyConfig struct {
	DeriveFacts   bool // default: true
	EmbedTriplets bool // default: true
	BatchSize     int  // default: 50
}

// MemifyResult is returned immediately (async execution is tracked via PipelineRun).
type MemifyResult struct {
	PipelineRunID string
	Status        string // QUEUED | RUNNING | COMPLETED | FAILED
	NewNodes      int
	NewEdges      int
}

// MemifyPorts groups all port interfaces needed by MemifyUseCase.
// Separate from StartCognifyUseCase ports to keep DI explicit.
type MemifyPorts struct {
	GraphRepo port.MemifyGraphRepository
	VectorRepo port.MemifyVectorRepository
	LLMClient  port.LLMClient
	Embedder   port.EmbedderClient
	RunRepo    port.PipelineRunRepository
	EventPub   port.EventPublisher
}

// MemifyUseCase implements non-destructive graph enrichment.
type MemifyUseCase struct {
	graphRepo  port.MemifyGraphRepository
	vectorRepo port.MemifyVectorRepository
	llmClient  port.LLMClient
	embedder   port.EmbedderClient
	runRepo    port.PipelineRunRepository
	eventPub   port.EventPublisher
}

// NewMemifyUseCase constructs the use case with all dependencies injected.
func NewMemifyUseCase(p MemifyPorts) *MemifyUseCase {
	return &MemifyUseCase{
		graphRepo:  p.GraphRepo,
		vectorRepo: p.VectorRepo,
		llmClient:  p.LLMClient,
		embedder:   p.Embedder,
		runRepo:    p.RunRepo,
		eventPub:   p.EventPub,
	}
}

// Execute creates a PipelineRun record and launches memify as a background goroutine.
// Returns immediately with QUEUED status.
func (uc *MemifyUseCase) Execute(ctx context.Context, req MemifyRequest) (*MemifyResult, error) {
	// Apply defaults
	if !req.Config.DeriveFacts {
		req.Config.DeriveFacts = memifyDefaultDeriveFacts
	}
	if !req.Config.EmbedTriplets {
		req.Config.EmbedTriplets = memifyDefaultEmbed
	}
	if req.Config.BatchSize == 0 {
		req.Config.BatchSize = defaultMemifyBatchSize
	}

	runID := uuid.New().String()
	run := domain.PipelineRun{
		ID:        runID,
		DatasetID: req.DatasetID.String(),
		TenantID:  req.TenantID,
		Type:      "memify",
		Status:    "QUEUED",
		CreatedAt: time.Now(),
	}
	if err := uc.runRepo.Save(ctx, run); err != nil {
		return nil, fmt.Errorf("save pipeline run: %w", err)
	}

	// Detach from request context — runs independently
	go uc.runMemify(context.Background(), runID, req)

	return &MemifyResult{PipelineRunID: runID, Status: "QUEUED"}, nil
}

// GetStatus retrieves the current status of a pipeline run.
func (uc *MemifyUseCase) GetStatus(ctx context.Context, runID string) (*domain.PipelineRun, error) {
	return uc.runRepo.GetByID(ctx, runID)
}

// runMemify is the background goroutine that executes the 5-step enrichment flow.
func (uc *MemifyUseCase) runMemify(ctx context.Context, runID string, req MemifyRequest) {
	_ = uc.runRepo.SetStatus(ctx, runID, "RUNNING")
	_ = uc.eventPub.PublishPipelineEvent(ctx, "cognee.cognify.memify.started", map[string]any{
		"pipeline_run_id": runID,
		"dataset_id":      req.DatasetID,
		"tenant_id":       req.TenantID,
	})

	result, err := uc.executeSteps(ctx, req)
	if err != nil {
		_ = uc.runRepo.SetStatusWithError(ctx, runID, "FAILED", err.Error())
		_ = uc.eventPub.PublishPipelineEvent(ctx, "cognee.cognify.memify.failed", map[string]any{
			"pipeline_run_id": runID,
			"error":           err.Error(),
		})
		return
	}

	_ = uc.runRepo.SetStatusWithResult(ctx, runID, "COMPLETED", result.NewNodes, result.NewEdges)
	_ = uc.eventPub.PublishPipelineEvent(ctx, "cognee.cognify.memify.completed", map[string]any{
		"pipeline_run_id": runID,
		"dataset_id":      req.DatasetID,
		"tenant_id":       req.TenantID,
		"new_nodes":       result.NewNodes,
		"new_edges":       result.NewEdges,
	})
}

// executeSteps runs the 5-step Memify enrichment pipeline.
func (uc *MemifyUseCase) executeSteps(ctx context.Context, req MemifyRequest) (*MemifyResult, error) {
	result := &MemifyResult{}

	// Step 1: Load existing graph (read-only)
	existingNodes, existingEdges, err := uc.graphRepo.GetDatasetGraph(ctx, req.DatasetID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("load graph: %w", err)
	}

	// Step 2: Derive new facts via LLM
	var derivedFacts []domain.GraphFact
	if req.Config.DeriveFacts {
		derivedFacts, err = uc.deriveFacts(ctx, existingNodes, existingEdges, req.Config.BatchSize)
		if err != nil {
			return nil, fmt.Errorf("derive facts: %w", err)
		}
	}

	// Step 3: Build diff (only NEW facts not already in graph)
	diff := buildEnrichmentDiff(existingEdges, derivedFacts)

	// Step 4: Upsert-only (non-destructive) — MERGE in Neo4j, no DELETE
	if len(diff.Nodes) > 0 || len(diff.Edges) > 0 {
		if err := uc.graphRepo.UpsertGraphDiff(ctx, req.DatasetID, req.TenantID, diff); err != nil {
			return nil, fmt.Errorf("upsert graph diff: %w", err)
		}
		result.NewNodes = len(diff.Nodes)
		result.NewEdges = len(diff.Edges)
	}

	// Step 5: Re-embed only new triplets (non-fatal if fails)
	if req.Config.EmbedTriplets && len(diff.Edges) > 0 {
		_ = uc.embedAndIndexTriplets(ctx, req.DatasetID, req.TenantID, diff.Edges)
	}

	return result, nil
}

// deriveFacts calls the LLM to infer new relationships from existing graph content.
func (uc *MemifyUseCase) deriveFacts(ctx context.Context,
	nodes []domain.GraphNode, edges []domain.GraphEdge, batchSize int,
) ([]domain.GraphFact, error) {

	const systemPrompt = `You are a knowledge graph enricher. Given existing entities and their relationships,
infer NEW relationships that can be logically derived but are not explicitly stated.
Return only relationships that are definitively true based on the given facts.
Format: JSON array of {"subject": "name", "predicate": "relation_type", "object": "name"}`

	var allFacts []domain.GraphFact
	for i := 0; i < len(nodes); i += batchSize {
		end := i + batchSize
		if end > len(nodes) {
			end = len(nodes)
		}
		batch := nodes[i:end]

		userMsg := buildGraphContext(batch, filterEdgesForNodes(edges, batch))
		resp, err := uc.llmClient.Complete(ctx, systemPrompt, userMsg)
		if err != nil {
			continue // skip bad batch — best-effort
		}

		var facts []domain.GraphFact
		if err := json.Unmarshal([]byte(resp), &facts); err != nil {
			continue
		}
		allFacts = append(allFacts, facts...)
	}
	return allFacts, nil
}

// embedAndIndexTriplets embeds triplet text and upserts into the vector store.
func (uc *MemifyUseCase) embedAndIndexTriplets(ctx context.Context,
	datasetID uuid.UUID, tenantID string, edges []domain.GraphEdge,
) error {
	collectionName := fmt.Sprintf("cognee_%s", tenantID)
	for _, edge := range edges {
		text := fmt.Sprintf("%s %s %s", edge.Subject, edge.Predicate, edge.Object)
		vec, err := uc.embedder.EmbedSingle(ctx, text)
		if err != nil {
			continue
		}
		_ = uc.vectorRepo.UpsertTripletPoint(ctx, collectionName, edge.ID, vec, map[string]any{
			"dataset_id": datasetID.String(),
			"subject":    edge.Subject,
			"predicate":  edge.Predicate,
			"object":     edge.Object,
		})
	}
	return nil
}

// buildEnrichmentDiff returns only facts not already in existing edges.
func buildEnrichmentDiff(existing []domain.GraphEdge, derived []domain.GraphFact) domain.GraphDiff {
	existingSet := make(map[string]bool, len(existing))
	for _, e := range existing {
		key := fmt.Sprintf("%s|%s|%s", e.Subject, e.Predicate, e.Object)
		existingSet[key] = true
	}

	var diff domain.GraphDiff
	for _, f := range derived {
		key := fmt.Sprintf("%s|%s|%s", f.Subject, f.Predicate, f.Object)
		if !existingSet[key] {
			diff.Edges = append(diff.Edges, domain.GraphEdge{
				ID:        uuid.New().String(),
				Subject:   f.Subject,
				Predicate: f.Predicate,
				Object:    f.Object,
				Properties: map[string]any{"derived": true},
			})
		}
	}
	return diff
}

// buildGraphContext builds a structured LLM prompt context from nodes + edges.
func buildGraphContext(nodes []domain.GraphNode, edges []domain.GraphEdge) string {
	var sb strings.Builder
	sb.WriteString("Entities:\n")
	for _, n := range nodes {
		sb.WriteString(fmt.Sprintf("- %s (%s)\n", n.Name, n.Type))
	}
	sb.WriteString("\nExisting relationships:\n")
	for _, e := range edges {
		sb.WriteString(fmt.Sprintf("- %s %s %s\n", e.Subject, e.Predicate, e.Object))
	}
	sb.WriteString("\nDerive new relationships (JSON array):")
	return sb.String()
}

// filterEdgesForNodes returns only edges where subject or object is in the node batch.
func filterEdgesForNodes(edges []domain.GraphEdge, nodes []domain.GraphNode) []domain.GraphEdge {
	nodeNames := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		nodeNames[n.Name] = true
	}
	var result []domain.GraphEdge
	for _, e := range edges {
		if nodeNames[e.Subject] || nodeNames[e.Object] {
			result = append(result, e)
		}
	}
	return result
}
