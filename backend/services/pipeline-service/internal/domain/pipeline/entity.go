// Package pipeline defines domain entities for the pipeline-service.
//
// Absorbed from: vnp-pipelines, ba-knowledge-service, ba-knowledge-worker
// (MERGE-P3-T1)
package pipeline

import "time"

// Pipeline aggregates status for one processing engine.
type Pipeline struct {
	Engine   string           `json:"engine"` // "graphiti"|"cognee"|"memobase"|"knowledge"
	Name     string           `json:"name"`
	Status   string           `json:"status"` // "idle"|"running"|"paused"|"error"
	JobCount PipelineJobCount `json:"job_count"`
	Workers  []*Worker        `json:"workers,omitempty"`
	Config   map[string]any   `json:"config,omitempty"`
}

// PipelineJobCount holds job statistics for one engine.
type PipelineJobCount struct {
	Pending   int `json:"pending"`
	Running   int `json:"running"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

// Job is a single async processing unit.
type Job struct {
	ID          string         `json:"id"`
	Engine      string         `json:"engine"`
	Type        string         `json:"type"`   // "ingest"|"index"|"sync"|"cognify"
	Status      string         `json:"status"` // "pending"|"running"|"completed"|"failed"
	Payload     map[string]any `json:"payload,omitempty"`
	Result      map[string]any `json:"result,omitempty"`
	Error       string         `json:"error,omitempty"`
	Priority    int            `json:"priority"`
	CreatedAt   time.Time      `json:"created_at"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}

// Queue describes a named Redis-backed job queue.
type Queue struct {
	Name    string `json:"name"`
	Engine  string `json:"engine"`
	Size    int    `json:"size"`
	MaxSize int    `json:"max_size"`
	Workers int    `json:"workers"`
}

// Worker describes a running pipeline worker process.
type Worker struct {
	ID       string    `json:"id"`
	Engine   string    `json:"engine"`
	Status   string    `json:"status"` // "idle"|"busy"|"offline"
	JobID    string    `json:"job_id,omitempty"`
	LastSeen time.Time `json:"last_seen"`
}

// PipelineTemplate is a reusable pipeline config template.
type PipelineTemplate struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Engine      string         `json:"engine"`
	Description string         `json:"description"`
	Config      map[string]any `json:"config,omitempty"`
}

// JobFilter is a filter for listing jobs.
type JobFilter struct {
	Status string `json:"status,omitempty"`
	Type   string `json:"type,omitempty"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// inferStatus derives pipeline status from job stats and workers.
func InferStatus(stats PipelineJobCount, workers []*Worker) string {
	for _, w := range workers {
		if w.Status == "error" {
			return "error"
		}
	}
	if stats.Running > 0 {
		return "running"
	}
	if stats.Pending > 0 {
		return "idle"
	}
	return "idle"
}
