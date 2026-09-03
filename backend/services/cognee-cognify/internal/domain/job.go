// Package domain defines CognifyJob and PipelineMetrics for tracking cognify pipeline execution.
// These types are referenced by domain/event.go for event publishing.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// CognifyJob tracks the lifecycle of a knowledge graph construction job.
type CognifyJob struct {
	ID         uuid.UUID
	TenantID   string
	DatasetID  uuid.UUID
	Status     JobStatus
	Stage      StageType
	Progress   float64
	EntryIDs   []string
	NodeSets   []string
	Config     CognifyConfig
	Metrics    PipelineMetrics
	Error      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
	EndedAt    *time.Time // alias for FinishedAt (legacy compat)
}

// CognifyConfig holds per-job processing configuration.
type CognifyConfig struct {
	Template      string
	Steps         []string
	ChunkSize     int
	ChunkOverlap  int
	CustomPrompt  string
	SkipDedup     bool
	SkipSummarize bool
	OntologyID    string
}

// DurationMs returns the elapsed duration in milliseconds.
func (j *CognifyJob) DurationMs() int64 {
	if j.StartedAt == nil {
		return 0
	}
	end := time.Now()
	if j.FinishedAt != nil {
		end = *j.FinishedAt
	}
	return end.Sub(*j.StartedAt).Milliseconds()
}

// PipelineMetrics holds performance counters for a cognify run.
type PipelineMetrics struct {
	TotalDurationMs    int64 `json:"total_duration_ms"`
	ChunksCreated      int   `json:"chunks_created"`
	EntitiesFound      int   `json:"entities_found"`
	RelationsFound     int   `json:"relations_found"`
	EmbeddingsCreated  int   `json:"embeddings_created"`
	LLMCallsTotal        int   `json:"llm_calls_total"`
	EmbeddingsGenerated  int   `json:"embeddings_generated"`
	EntitiesDeduplicated  int   `json:"entities_deduplicated"`
	EntitiesExtracted     int   `json:"entities_extracted"`
	RelationshipsExtracted int  `json:"relationships_extracted"`
	CommunitiesFound     int   `json:"communities_found"`
}

// NewCognifyJob creates a new job in PENDING state.
func NewCognifyJob(id, datasetID uuid.UUID, tenantID string, entryIDs, nodeSets []string, config CognifyConfig) *CognifyJob {
	return &CognifyJob{
		ID:        id,
		TenantID:  tenantID,
		DatasetID: datasetID,
		Status:    JobPending,
		Stage:     StageNone,
		EntryIDs:  entryIDs,
		NodeSets:  nodeSets,
		Config:    config,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Fail transitions the job to failed state with an error message.
func (j *CognifyJob) Fail(err error) {
	j.Status = JobFailed
	if err != nil {
		j.Error = err.Error()
	}
	now := time.Now()
	j.FinishedAt = &now
	j.EndedAt = &now
	j.UpdatedAt = now
}

// Complete transitions the job to completed state.
func (j *CognifyJob) Complete(metrics PipelineMetrics) {
	j.Status = JobCompleted
	j.Metrics = metrics
	now := time.Now()
	j.FinishedAt = &now
	j.EndedAt = &now
	j.UpdatedAt = now
}

// AdvanceStage moves the job to the next pipeline stage.
func (j *CognifyJob) AdvanceStage(stage StageType) {
	j.Stage = stage
	j.Progress = StageProgress(indexOf(AllStages(), stage))
	j.UpdatedAt = time.Now()
}

func indexOf(stages []StageType, target StageType) int {
	for i, s := range stages {
		if s == target {
			return i
		}
	}
	return 0
}
