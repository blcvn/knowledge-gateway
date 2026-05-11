package domain

import (
	"time"

	"github.com/google/uuid"
)

// CognifyJob represents an execution of the 8-stage cognification pipeline.
// It is a state machine: PENDING → RUNNING → COMPLETED | FAILED.
type CognifyJob struct {
	ID        uuid.UUID     `json:"id"`
	TenantID  string        `json:"tenant_id"`
	DatasetID uuid.UUID     `json:"dataset_id"`
	Status    JobStatus     `json:"status"`
	Stage     StageType     `json:"stage"`
	Progress  float64       `json:"progress"` // 0.0–1.0
	Error     string        `json:"error,omitempty"`
	Config    CognifyConfig `json:"config"`
	Metrics   PipelineMetrics `json:"metrics"`
	StartedAt time.Time     `json:"started_at"`
	EndedAt   *time.Time    `json:"ended_at,omitempty"`
}

// NewCognifyJob creates a new job in PENDING state.
func NewCognifyJob(tenantID string, datasetID uuid.UUID, cfg CognifyConfig) *CognifyJob {
	return &CognifyJob{
		ID:        uuid.New(),
		TenantID:  tenantID,
		DatasetID: datasetID,
		Status:    JobPending,
		Stage:     StageNone,
		Progress:  0.0,
		Config:    cfg,
		Metrics:   PipelineMetrics{},
		StartedAt: time.Now().UTC(),
	}
}

// AdvanceStage transitions the job to the next pipeline stage.
func (j *CognifyJob) AdvanceStage(stage StageType, progress float64) {
	j.Status = JobRunning
	j.Stage = stage
	j.Progress = progress
}

// Complete marks the job as successfully completed.
func (j *CognifyJob) Complete() {
	j.Status = JobCompleted
	j.Progress = 1.0
	now := time.Now().UTC()
	j.EndedAt = &now
}

// Fail marks the job as failed with an error message.
func (j *CognifyJob) Fail(err error) {
	j.Status = JobFailed
	j.Error = err.Error()
	now := time.Now().UTC()
	j.EndedAt = &now
}

// DurationMs returns the pipeline execution duration in milliseconds.
func (j *CognifyJob) DurationMs() int64 {
	end := time.Now().UTC()
	if j.EndedAt != nil {
		end = *j.EndedAt
	}
	return end.Sub(j.StartedAt).Milliseconds()
}

// CognifyConfig holds user-configurable pipeline parameters.
type CognifyConfig struct {
	ChunkSize       int    `json:"chunk_size"`        // tokens per chunk (default: 1024)
	ChunkOverlap    int    `json:"chunk_overlap"`     // overlap between chunks (default: 128)
	SkipDedup       bool   `json:"skip_dedup"`        // skip deduplication stage
	SkipSummarize   bool   `json:"skip_summarize"`    // skip community summarization
	OntologyID      string `json:"ontology_id"`       // optional ontology for guided extraction
	MaxLLMConcurrency int  `json:"max_llm_concurrency"` // bulkhead limit (default: 5)
}

// DefaultCognifyConfig returns production defaults.
func DefaultCognifyConfig() CognifyConfig {
	return CognifyConfig{
		ChunkSize:         1024,
		ChunkOverlap:      128,
		SkipDedup:         false,
		SkipSummarize:     false,
		MaxLLMConcurrency: 5,
	}
}

// Chunk represents a segment of text produced by the chunking stage.
type Chunk struct {
	ID         uuid.UUID `json:"id"`
	JobID      uuid.UUID `json:"job_id"`
	Index      int       `json:"index"`      // position in source document
	Text       string    `json:"text"`
	TokenCount int       `json:"token_count"`
	SourceItem string    `json:"source_item"` // DataItem ID reference
}

// NewChunk creates a chunk with the given text.
func NewChunk(jobID uuid.UUID, index int, text string, tokenCount int, sourceItem string) *Chunk {
	return &Chunk{
		ID:         uuid.New(),
		JobID:      jobID,
		Index:      index,
		Text:       text,
		TokenCount: tokenCount,
		SourceItem: sourceItem,
	}
}

// Entity represents a named entity extracted by LLM NER.
type Entity struct {
	ID          uuid.UUID  `json:"id"`
	JobID       uuid.UUID  `json:"job_id"`
	Name        string     `json:"name"`
	EntityType  EntityType `json:"entity_type"`
	Description string     `json:"description"`
	SourceChunk uuid.UUID  `json:"source_chunk"` // Chunk that produced this entity
}

// NewEntity creates an entity from LLM extraction output.
func NewEntity(jobID uuid.UUID, name string, entityType EntityType, description string, sourceChunk uuid.UUID) *Entity {
	return &Entity{
		ID:          uuid.New(),
		JobID:       jobID,
		Name:        name,
		EntityType:  entityType,
		Description: description,
		SourceChunk: sourceChunk,
	}
}

// Relationship represents a directed edge between two entities.
type Relationship struct {
	ID          uuid.UUID `json:"id"`
	JobID       uuid.UUID `json:"job_id"`
	SourceID    uuid.UUID `json:"source_id"`    // Entity ID
	TargetID    uuid.UUID `json:"target_id"`    // Entity ID
	Relation    string    `json:"relation"`     // e.g. "works_for", "located_in"
	Weight      float64   `json:"weight"`       // confidence 0.0–1.0
	SourceChunk uuid.UUID `json:"source_chunk"` // Chunk that produced this relationship
}

// NewRelationship creates a relationship from LLM extraction output.
func NewRelationship(jobID, sourceID, targetID uuid.UUID, relation string, weight float64, sourceChunk uuid.UUID) *Relationship {
	return &Relationship{
		ID:          uuid.New(),
		JobID:       jobID,
		SourceID:    sourceID,
		TargetID:    targetID,
		Relation:    relation,
		Weight:      weight,
		SourceChunk: sourceChunk,
	}
}

// Community represents a cluster of related entities with a summary.
type Community struct {
	ID        uuid.UUID   `json:"id"`
	JobID     uuid.UUID   `json:"job_id"`
	EntityIDs []uuid.UUID `json:"entity_ids"`
	Summary   string      `json:"summary"` // LLM-generated summary
	Level     int         `json:"level"`   // hierarchy level (0 = leaf)
}

// NewCommunity creates a community cluster.
func NewCommunity(jobID uuid.UUID, entityIDs []uuid.UUID, level int) *Community {
	return &Community{
		ID:        uuid.New(),
		JobID:     jobID,
		EntityIDs: entityIDs,
		Level:     level,
	}
}

// Ontology represents a domain-specific knowledge schema for guided extraction.
type Ontology struct {
	ID         uuid.UUID          `json:"id"`
	TenantID   string             `json:"tenant_id"`
	Name       string             `json:"name"`
	Categories []OntologyCategory `json:"categories"`
}

// OntologyCategory is a classification bucket within an ontology.
type OntologyCategory struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	EntityTypes []string `json:"entity_types"` // expected entity types in this category
}

// PipelineMetrics aggregates statistics across all pipeline stages.
type PipelineMetrics struct {
	ChunksCreated          int   `json:"chunks_created"`
	EntitiesExtracted      int   `json:"entities_extracted"`
	RelationshipsExtracted int   `json:"relationships_extracted"`
	EntitiesDeduplicated   int   `json:"entities_deduplicated"`
	CommunitiesFound       int   `json:"communities_found"`
	EmbeddingsGenerated    int   `json:"embeddings_generated"`
	LLMCallsTotal          int   `json:"llm_calls_total"`
	LLMTokensUsed          int   `json:"llm_tokens_used"`
	TotalDurationMs        int64 `json:"total_duration_ms"`
}
