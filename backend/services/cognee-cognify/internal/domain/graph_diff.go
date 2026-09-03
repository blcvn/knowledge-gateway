// Package domain defines graph diff types for non-destructive Memify enrichment.
// TASK-CE-006: Memify UseCase (Non-Destructive Graph Enrichment)
package domain

import "time"

// GraphFact — LLM-derived relationship (S, P, O)
type GraphFact struct {
	Subject   string
	Predicate string
	Object    string
}

// GraphDiff — delta between existing and derived graph (additions only, no deletes)
type GraphDiff struct {
	Nodes []GraphNode // new nodes inferred (not yet in graph)
	Edges []GraphEdge // new edges derived (not yet in graph)
}

// PipelineRun — tracks async cognify/memify job status in cognee_pipeline_runs table
type PipelineRun struct {
	ID        string
	DatasetID string
	TenantID  string
	Type      string    // "cognify" | "memify"
	Status    string    // QUEUED | RUNNING | COMPLETED | FAILED
	NewNodes  int
	NewEdges  int
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
}
