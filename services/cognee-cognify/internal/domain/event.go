package domain

import (
	"time"

	"github.com/google/uuid"
)

// PipelineCompletedEvent is published when all stages complete successfully.
type PipelineCompletedEvent struct {
	JobID     uuid.UUID       `json:"job_id"`
	TenantID  string          `json:"tenant_id"`
	DatasetID uuid.UUID       `json:"dataset_id"`
	Metrics   PipelineMetrics `json:"metrics"`
	Timestamp time.Time       `json:"timestamp"`
}

// NewPipelineCompletedEvent creates a completion event from a finished job.
func NewPipelineCompletedEvent(job *CognifyJob) PipelineCompletedEvent {
	job.Metrics.TotalDurationMs = job.DurationMs()
	return PipelineCompletedEvent{
		JobID:     job.ID,
		TenantID:  job.TenantID,
		DatasetID: job.DatasetID,
		Metrics:   job.Metrics,
		Timestamp: time.Now().UTC(),
	}
}

// StageAdvancedEvent is published when the pipeline transitions to a new stage.
type StageAdvancedEvent struct {
	JobID    uuid.UUID `json:"job_id"`
	TenantID string    `json:"tenant_id"`
	Stage    StageType `json:"stage"`
	Progress float64   `json:"progress"`
	Timestamp time.Time `json:"timestamp"`
}

// NewStageAdvancedEvent creates a stage transition event.
func NewStageAdvancedEvent(job *CognifyJob) StageAdvancedEvent {
	return StageAdvancedEvent{
		JobID:     job.ID,
		TenantID:  job.TenantID,
		Stage:     job.Stage,
		Progress:  job.Progress,
		Timestamp: time.Now().UTC(),
	}
}
