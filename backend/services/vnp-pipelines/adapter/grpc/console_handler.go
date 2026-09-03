// Package grpc provides stub handlers for vnp-pipelines console endpoints.
// Returns mock data matching UI's pipeline.ts types.
package grpc

import (
	"context"
	"encoding/json"
	"time"
)

// PipelinesHandler provides stub console endpoints for pipeline monitoring.
type PipelinesHandler struct{}

func NewPipelinesHandler() *PipelinesHandler {
	return &PipelinesHandler{}
}

type EngineStatus struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	ActiveJobs int    `json:"active_jobs"`
	QueueDepth int    `json:"queue_depth"`
	Workers    int    `json:"workers"`
}

type PipelineJob struct {
	ID        string  `json:"id"`
	Engine    string  `json:"engine"`
	Status    string  `json:"status"`
	Progress  float64 `json:"progress"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type QueueMetrics struct {
	Depth      int     `json:"depth"`
	Throughput float64 `json:"throughput"`
	RetryCount int     `json:"retry_count"`
}

type WorkerStatus struct {
	Engine  string `json:"engine"`
	Running int    `json:"running"`
	Idle    int    `json:"idle"`
}

// GetStatus returns aggregated pipeline status.
func (h *PipelinesHandler) GetStatus(_ context.Context) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	data := map[string]interface{}{
		"engines": map[string]EngineStatus{
			"cognee":      {Name: "cognee", Status: "healthy", ActiveJobs: 3, QueueDepth: 5, Workers: 4},
			"graphiti":    {Name: "graphiti", Status: "healthy", ActiveJobs: 1, QueueDepth: 2, Workers: 2},
			"memobase":    {Name: "memobase", Status: "healthy", ActiveJobs: 2, QueueDepth: 0, Workers: 3},
			"openviking":  {Name: "openviking", Status: "degraded", ActiveJobs: 0, QueueDepth: 8, Workers: 1},
			"zep":         {Name: "zep", Status: "healthy", ActiveJobs: 4, QueueDepth: 3, Workers: 4},
			"supermemory": {Name: "supermemory", Status: "healthy", ActiveJobs: 1, QueueDepth: 1, Workers: 2},
		},
		"total_jobs":  11,
		"queue_depth": 19,
		"workers":     16,
		"updated_at":  now,
	}
	return json.Marshal(data)
}

// GetQueues returns queue metrics.
func (h *PipelinesHandler) GetQueues(_ context.Context) ([]byte, error) {
	data := QueueMetrics{Depth: 19, Throughput: 45.2, RetryCount: 3}
	return json.Marshal(data)
}

// GetWorkers returns worker status per engine.
func (h *PipelinesHandler) GetWorkers(_ context.Context) ([]byte, error) {
	data := []WorkerStatus{
		{Engine: "cognee", Running: 3, Idle: 1},
		{Engine: "graphiti", Running: 1, Idle: 1},
		{Engine: "memobase", Running: 2, Idle: 1},
		{Engine: "openviking", Running: 0, Idle: 1},
		{Engine: "zep", Running: 3, Idle: 1},
		{Engine: "supermemory", Running: 1, Idle: 1},
	}
	return json.Marshal(data)
}

// ListJobs returns stub pipeline jobs.
func (h *PipelinesHandler) ListJobs(_ context.Context, engine string) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	data := []PipelineJob{
		{ID: "job-001", Engine: engine, Status: "Running", Progress: 0.65, CreatedAt: now, UpdatedAt: now},
		{ID: "job-002", Engine: engine, Status: "Completed", Progress: 1.0, CreatedAt: now, UpdatedAt: now},
		{ID: "job-003", Engine: engine, Status: "Queued", Progress: 0.0, CreatedAt: now, UpdatedAt: now},
	}
	return json.Marshal(data)
}

// GetJob returns a single pipeline job.
func (h *PipelinesHandler) GetJob(_ context.Context, engine, jobID string) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	data := PipelineJob{ID: jobID, Engine: engine, Status: "Running", Progress: 0.65, CreatedAt: now, UpdatedAt: now}
	return json.Marshal(data)
}
