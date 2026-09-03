// Package cognify defines domain entities for the Cognee cognification pipeline.
package cognify

import (
	"time"

	"github.com/google/uuid"
)

// CognifyJob represents a cognification pipeline execution.
type CognifyJob struct {
	ID        uuid.UUID     `json:"id"`
	TenantID  uuid.UUID     `json:"tenant_id"`
	DatasetID uuid.UUID     `json:"dataset_id"`
	Status    JobStatus     `json:"status"`
	Stage     PipelineStage `json:"stage"`
	Progress  float64       `json:"progress"` // 0.0 - 1.0
	Error     string        `json:"error,omitempty"`
	StartedAt time.Time     `json:"started_at"`
	EndedAt   *time.Time    `json:"ended_at,omitempty"`
}

// JobStatus represents cognify job lifecycle.
type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
)

// PipelineStage represents the 7 cognify pipeline stages.
type PipelineStage string

const (
	StageExtractEntities    PipelineStage = "extract_entities"
	StageResolveEntities    PipelineStage = "resolve_entities"
	StageExtractEdges       PipelineStage = "extract_edges"
	StageResolveEdges       PipelineStage = "resolve_edges"
	StageGenerateEmbeddings PipelineStage = "generate_embeddings"
	StageUpdateCommunities  PipelineStage = "update_communities"
	StagePostProcess        PipelineStage = "post_process"
)

// Ontology represents a domain-specific knowledge schema.
type Ontology struct {
	ID         uuid.UUID        `json:"id"`
	TenantID   uuid.UUID        `json:"tenant_id"`
	Name       string           `json:"name"`
	Categories []OntologyCategory `json:"categories"`
}

// OntologyCategory is a classification bucket within an ontology.
type OntologyCategory struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
