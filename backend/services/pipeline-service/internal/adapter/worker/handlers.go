// Package worker implements task handlers for the pipeline worker binary.
//
// This is the worker adapter — task handlers registered with the Redis Consumer.
// Implements: HandleIndexPRD, HandleGenOutline + cross-engine pipeline handlers.
// (MERGE-P3-T1)
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	domk "vnp-memory/services/pipeline-service/internal/domain/knowledge"
	redisqueue "vnp-memory/services/pipeline-service/internal/infra/redis"
	uck "vnp-memory/services/pipeline-service/internal/usecase/knowledge"
	ucp "vnp-memory/services/pipeline-service/internal/usecase/pipeline"
)

// Registry maps task types to handlers.
type Registry struct {
	consumer  *redisqueue.Consumer
	indexUC   *uck.IndexUseCase
	pipelineUC *ucp.PipelineUseCase
	logger    *slog.Logger
}

// NewRegistry creates a Registry.
func NewRegistry(consumer *redisqueue.Consumer, indexUC *uck.IndexUseCase, pipelineUC *ucp.PipelineUseCase, logger *slog.Logger) *Registry {
	return &Registry{consumer: consumer, indexUC: indexUC, pipelineUC: pipelineUC, logger: logger}
}

// RegisterAll registers all known task handlers.
func (reg *Registry) RegisterAll() {
	reg.consumer.RegisterHandler(domk.TaskTypeIndexPRD, reg.HandleIndexPRD)
	reg.consumer.RegisterHandler(domk.TaskTypeGenOutline, reg.HandleGenOutline)
	reg.consumer.RegisterHandler(domk.TaskTypeGraphitiIngest, reg.HandleGraphitiIngest)
	reg.consumer.RegisterHandler(domk.TaskTypeCogneeCognify, reg.HandleCogneeCognify)
	reg.consumer.RegisterHandler(domk.TaskTypeMemobaseFlush, reg.HandleMemobaseFlush)
	reg.logger.Info("Registered task handlers",
		"handlers", []string{
			domk.TaskTypeIndexPRD,
			domk.TaskTypeGenOutline,
			domk.TaskTypeGraphitiIngest,
			domk.TaskTypeCogneeCognify,
			domk.TaskTypeMemobaseFlush,
		},
	)
}

// ── Knowledge Task Handlers ──────────────────────────────────────────────────

type indexPRDPayload struct {
	PRDID string `json:"prd_id"`
}

// HandleIndexPRD processes "index_prd" tasks.
func (reg *Registry) HandleIndexPRD(ctx context.Context, job redisqueue.TaskJob) error {
	var payload indexPRDPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal index_prd payload: %w", err)
	}
	reg.logger.Info("Handling index_prd", "prd_id", payload.PRDID)

	indexJob := domk.IndexJob{
		Type:  domk.TaskTypeIndexPRD,
		PRDID: payload.PRDID,
	}
	if err := reg.indexUC.HandleIndexPRD(ctx, indexJob); err != nil {
		return fmt.Errorf("index_prd %s: %w", payload.PRDID, err)
	}
	reg.logger.Info("Completed index_prd", "prd_id", payload.PRDID)
	return nil
}

type genOutlinePayload struct {
	PRDID string `json:"prd_id"`
}

// HandleGenOutline processes "gen_outline" tasks.
func (reg *Registry) HandleGenOutline(ctx context.Context, job redisqueue.TaskJob) error {
	var payload genOutlinePayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal gen_outline payload: %w", err)
	}
	reg.logger.Info("Handling gen_outline", "prd_id", payload.PRDID)

	indexJob := domk.IndexJob{
		Type:  domk.TaskTypeGenOutline,
		PRDID: payload.PRDID,
	}
	if err := reg.indexUC.HandleGenOutline(ctx, indexJob); err != nil {
		return fmt.Errorf("gen_outline %s: %w", payload.PRDID, err)
	}
	reg.logger.Info("Completed gen_outline", "prd_id", payload.PRDID)
	return nil
}

// ── Cross-Engine Pipeline Handlers ───────────────────────────────────────────

type graphitiIngestPayload struct {
	Content  string `json:"content"`
	TenantID string `json:"tenant_id"`
	Source   string `json:"source"`
}

// HandleGraphitiIngest processes "graphiti.ingest" tasks.
func (reg *Registry) HandleGraphitiIngest(ctx context.Context, job redisqueue.TaskJob) error {
	var payload graphitiIngestPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal graphiti.ingest payload: %w", err)
	}
	reg.logger.Info("Handling graphiti.ingest", "tenant_id", payload.TenantID)
	// TODO: call kg-service gRPC ForwardService to ingest
	// For MVP: mark as completed
	return nil
}

type cogneeCognifyPayload struct {
	DatasetID string `json:"dataset_id"`
}

// HandleCogneeCognify processes "cognee.cognify" tasks.
func (reg *Registry) HandleCogneeCognify(ctx context.Context, job redisqueue.TaskJob) error {
	var payload cogneeCognifyPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal cognee.cognify payload: %w", err)
	}
	reg.logger.Info("Handling cognee.cognify", "dataset_id", payload.DatasetID)
	// TODO: call kg-service Cognee endpoint
	return nil
}

type memobaseFlushPayload struct {
	UserID string `json:"user_id"`
}

// HandleMemobaseFlush processes "memobase.flush" tasks.
func (reg *Registry) HandleMemobaseFlush(ctx context.Context, job redisqueue.TaskJob) error {
	var payload memobaseFlushPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal memobase.flush payload: %w", err)
	}
	reg.logger.Info("Handling memobase.flush", "user_id", payload.UserID)
	// TODO: call memory-service flush endpoint
	return nil
}
