package dto

import (
	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/domain"
)

// CognifyJobResult is the output of a completed cognification pipeline.
type CognifyJobResult struct {
	JobID       uuid.UUID             `json:"job_id"`
	DatasetID   uuid.UUID             `json:"dataset_id"`
	Status      domain.JobStatus      `json:"status"`
	Metrics     domain.PipelineMetrics `json:"metrics"`
	DurationMs  int64                 `json:"duration_ms"`
}
