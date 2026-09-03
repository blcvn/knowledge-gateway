package usecase

import (
	"github.com/google/uuid"
	"context"
	"log/slog"

	"vnp-memory/services/cognee-cognify/internal/domain"
	"vnp-memory/services/cognee-cognify/internal/usecase/dto"
	"vnp-memory/services/cognee-cognify/internal/usecase/port"
)

// CognifyOrchestrator implements port.CognifyUseCase.
// It runs all 8 stages sequentially with state machine persistence.
type CognifyOrchestrator struct {
	jobRepo    port.JobRepository
	dataReader port.DataItemReader
	eventPub   port.EventPublisher
	stages     []Stage
	logger     *slog.Logger
}

// NewCognifyOrchestrator creates the orchestrator with all 8 stages wired in.
func NewCognifyOrchestrator(
	jobRepo port.JobRepository,
	dataReader port.DataItemReader,
	eventPub port.EventPublisher,
	llm port.LLMClient,
	embedder port.EmbedderClient,
	graphRepo port.GraphRepository,
	vectorRepo port.VectorRepository,
	logger *slog.Logger,
) *CognifyOrchestrator {
	return &CognifyOrchestrator{
		jobRepo:    jobRepo,
		dataReader: dataReader,
		eventPub:   eventPub,
		stages: []Stage{
			NewClassifyStage(llm, logger),
			NewChunkStage(logger),
			NewExtractEntitiesStage(llm, logger),
			NewExtractRelationshipsStage(llm, logger),
			NewDeduplicateStage(llm, logger),
			NewBuildGraphStage(graphRepo, logger),
			NewEmbedStage(embedder, vectorRepo, logger),
			NewSummarizeStage(llm, graphRepo, logger),
		},
		logger: logger.With("usecase", "cognify_orchestrator"),
	}
}

// Execute runs the full 8-stage cognification pipeline.
func (uc *CognifyOrchestrator) Execute(ctx context.Context, req dto.TriggerCognifyRequest) (*dto.CognifyJobResult, error) {
	// Validate input
	if req.TenantID == "" {
		return nil, domain.ErrMissingTenantID
	}

	// Build config
	cfg := domain.DefaultCognifyConfig()
	if req.ChunkSize > 0 {
		cfg.ChunkSize = req.ChunkSize
	}
	if req.ChunkOverlap > 0 {
		cfg.ChunkOverlap = req.ChunkOverlap
	}
	cfg.SkipDedup = req.SkipDedup
	cfg.SkipSummarize = req.SkipSummarize
	cfg.OntologyID = req.OntologyID

	// Create job
	job := domain.NewCognifyJob(uuid.New(), req.DatasetID, req.TenantID, req.EntryIDs, req.NodeSets, cfg)
	if err := uc.jobRepo.Create(ctx, job); err != nil {
		return nil, err
	}

	uc.logger.Info("cognify pipeline started",
		"job_id", job.ID,
		"dataset_id", req.DatasetID,
		"tenant_id", req.TenantID,
	)

	// Load text from ingested data
	textItems, err := uc.dataReader.GetTextByDataset(ctx, req.TenantID, req.DatasetID)
	if err != nil {
		job.Fail(err)
		_ = uc.jobRepo.Update(ctx, job)
		return nil, err
	}

	// Initialize pipeline state
	state := &CognifyPipelineState{TextItems: textItems}

	// Execute all stages
	allStages := domain.AllStages()
	for i, stage := range uc.stages {
		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			job.Fail(err)
			_ = uc.jobRepo.Update(ctx, job)
			return nil, err
		}

		// Skip stages if configured
		if stage.Name() == domain.StageDeduplicate && cfg.SkipDedup {
			continue
		}
		if stage.Name() == domain.StageSummarize && cfg.SkipSummarize {
			continue
		}

		// Advance job stage
		job.AdvanceStage(allStages[i])
		_ = uc.jobRepo.Update(ctx, job)

		// Publish stage event
		_ = uc.eventPub.PublishStageAdvanced(ctx, domain.NewStageAdvancedEvent(job))

		// Execute stage
		if err := stage.Execute(ctx, job, state); err != nil {
			pipelineErr := &domain.ErrPipelineFailed{Stage: stage.Name(), Cause: err}
			job.Fail(pipelineErr)
			_ = uc.jobRepo.Update(ctx, job)
			return nil, pipelineErr
		}
	}

	// Complete
	job.Complete(job.Metrics)
	job.Metrics.TotalDurationMs = job.DurationMs()
	_ = uc.jobRepo.Update(ctx, job)

	// Publish completion event
	completedEvent := domain.NewPipelineCompletedEvent(job)
	_ = uc.eventPub.PublishPipelineCompleted(ctx, completedEvent)

	uc.logger.Info("cognify pipeline completed",
		"job_id", job.ID,
		"duration_ms", job.Metrics.TotalDurationMs,
		"entities", job.Metrics.EntitiesExtracted,
		"relationships", job.Metrics.RelationshipsExtracted,
		"communities", job.Metrics.CommunitiesFound,
	)

	return &dto.CognifyJobResult{
		JobID:      job.ID,
		DatasetID:  job.DatasetID,
		Status:     job.Status,
		Metrics:    job.Metrics,
		DurationMs: job.Metrics.TotalDurationMs,
	}, nil
}
