// Package port defines output interfaces for pipeline-service.
package port

import (
	"context"

	"vnp-memory/services/pipeline-service/internal/domain/knowledge"
	"vnp-memory/services/pipeline-service/internal/domain/pipeline"
)

// ── Pipeline Repos ─────────────────────────────────────────────────────────

// JobRepository persists pipeline jobs.
type JobRepository interface {
	Create(ctx context.Context, job *pipeline.Job) error
	GetByID(ctx context.Context, id string) (*pipeline.Job, error)
	GetStats(ctx context.Context, engine string) (pipeline.PipelineJobCount, error)
	ListByEngine(ctx context.Context, engine string, filter pipeline.JobFilter) ([]*pipeline.Job, int, error)
	UpdateStatus(ctx context.Context, id, status, errMsg string) error
}

// WorkerRegistry tracks live worker processes.
type WorkerRegistry interface {
	Register(ctx context.Context, worker *pipeline.Worker) error
	Heartbeat(ctx context.Context, workerID string) error
	ListByEngine(ctx context.Context, engine string) ([]*pipeline.Worker, error)
	GetQueues(ctx context.Context) ([]*pipeline.Queue, error)
}

// EventPublisher publishes domain events to NATS.
type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}

// ── Knowledge Repos ────────────────────────────────────────────────────────

// PRDRepository persists PRDs.
type PRDRepository interface {
	Create(ctx context.Context, prd *knowledge.PRD) error
	GetByID(ctx context.Context, id string) (*knowledge.PRD, error)
	List(ctx context.Context, tenantID string, limit, offset int) ([]*knowledge.PRD, int, error)
	UpdateStatus(ctx context.Context, id, status string) error
}

// OutlineRepository persists outlines.
type OutlineRepository interface {
	Create(ctx context.Context, outline *knowledge.Outline) error
	GetByPRD(ctx context.Context, prdID string) (*knowledge.Outline, error)
	UpdateStatus(ctx context.Context, id, status string) error
}

// Queue is the interface to the Redis task queue.
type Queue interface {
	Push(ctx context.Context, queueName string, payload []byte) error
	Pop(ctx context.Context, queueName string, timeout int) ([]byte, error)
	Size(ctx context.Context, queueName string) (int64, error)
}
