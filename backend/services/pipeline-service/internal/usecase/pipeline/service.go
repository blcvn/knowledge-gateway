// Package pipeline implements PipelineUseCase.
//
// Absorbed from: vnp-pipelines (pipeline management API)
// (MERGE-P3-T1)
package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	dom "vnp-memory/services/pipeline-service/internal/domain/pipeline"
	"vnp-memory/services/pipeline-service/internal/usecase/port"
)

// knownEngines is the fixed list of pipeline engines.
var knownEngines = []string{"graphiti", "cognee", "memobase", "knowledge"}

// defaultTemplates provides built-in pipeline templates.
var defaultTemplates = []*dom.PipelineTemplate{
	{
		ID:          "tpl-graphiti-ingest",
		Name:        "Graphiti Episode Ingestion",
		Engine:      "graphiti",
		Description: "Ingest text content into the Graphiti knowledge graph",
		Config:      map[string]any{"batch_size": 10, "embedding": true},
	},
	{
		ID:          "tpl-cognee-cognify",
		Name:        "Cognee Cognification",
		Engine:      "cognee",
		Description: "Run Cognee cognification pipeline on a dataset",
		Config:      map[string]any{"search_type": "GRAPH_COMPLETION"},
	},
	{
		ID:          "tpl-memobase-flush",
		Name:        "Memobase Buffer Flush",
		Engine:      "memobase",
		Description: "Flush user memory buffer to long-term storage",
		Config:      map[string]any{"flush_threshold": 20},
	},
	{
		ID:          "tpl-knowledge-index",
		Name:        "PRD Knowledge Indexing",
		Engine:      "knowledge",
		Description: "Index a PRD document into the knowledge base",
		Config:      map[string]any{"generate_outline": true},
	},
}

// PipelineUseCase manages pipeline status and job lifecycle.
type PipelineUseCase struct {
	jobs    port.JobRepository
	workers port.WorkerRegistry
	pub     port.EventPublisher
}

// NewPipelineUseCase creates a PipelineUseCase.
func NewPipelineUseCase(jobs port.JobRepository, workers port.WorkerRegistry, pub port.EventPublisher) *PipelineUseCase {
	return &PipelineUseCase{jobs: jobs, workers: workers, pub: pub}
}

// Status aggregates pipeline status for all engines.
func (uc *PipelineUseCase) Status(ctx context.Context) ([]*dom.Pipeline, error) {
	pipelines := make([]*dom.Pipeline, 0, len(knownEngines))
	for _, engine := range knownEngines {
		stats, _ := uc.jobs.GetStats(ctx, engine)
		workers, _ := uc.workers.ListByEngine(ctx, engine)
		pipelines = append(pipelines, &dom.Pipeline{
			Engine:   engine,
			Name:     engine + "-pipeline",
			Status:   dom.InferStatus(stats, workers),
			JobCount: stats,
			Workers:  workers,
		})
	}
	return pipelines, nil
}

// GetEngine returns pipeline status for a specific engine.
func (uc *PipelineUseCase) GetEngine(ctx context.Context, engine string) (*dom.Pipeline, error) {
	stats, err := uc.jobs.GetStats(ctx, engine)
	if err != nil {
		return nil, fmt.Errorf("get pipeline stats: %w", err)
	}
	workers, _ := uc.workers.ListByEngine(ctx, engine)
	return &dom.Pipeline{
		Engine:   engine,
		Name:     engine + "-pipeline",
		Status:   dom.InferStatus(stats, workers),
		JobCount: stats,
		Workers:  workers,
	}, nil
}

// ListJobs lists jobs for an engine with optional filter.
func (uc *PipelineUseCase) ListJobs(ctx context.Context, engine string, filter dom.JobFilter) ([]*dom.Job, int, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	return uc.jobs.ListByEngine(ctx, engine, filter)
}

// GetJob retrieves a job by ID.
func (uc *PipelineUseCase) GetJob(ctx context.Context, engine, jobID string) (*dom.Job, error) {
	return uc.jobs.GetByID(ctx, jobID)
}

// Queues returns all queue depths from the worker registry.
func (uc *PipelineUseCase) Queues(ctx context.Context) ([]*dom.Queue, error) {
	return uc.workers.GetQueues(ctx)
}

// Workers returns all registered workers.
func (uc *PipelineUseCase) Workers(ctx context.Context) ([]*dom.Worker, error) {
	var all []*dom.Worker
	for _, engine := range knownEngines {
		ws, _ := uc.workers.ListByEngine(ctx, engine)
		all = append(all, ws...)
	}
	return all, nil
}

// Templates returns available pipeline templates.
func (uc *PipelineUseCase) Templates(_ context.Context) ([]*dom.PipelineTemplate, error) {
	return defaultTemplates, nil
}

// EnqueueJob creates and enqueues a new job.
func (uc *PipelineUseCase) EnqueueJob(ctx context.Context, engine, jobType string, payload map[string]any) (*dom.Job, error) {
	now := time.Now()
	job := &dom.Job{
		ID:        uuid.New().String(),
		Engine:    engine,
		Type:      jobType,
		Status:    "pending",
		Payload:   payload,
		Priority:  0,
		CreatedAt: now,
	}
	if err := uc.jobs.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	if uc.pub != nil {
		_ = uc.pub.Publish(ctx, fmt.Sprintf("pipeline.%s.%s.enqueued", engine, jobType), job)
	}
	return job, nil
}
