package saga

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"graphiti-pipeline/internal/domain/ingestion"
	"graphiti-pipeline/internal/usecase/port"
)

type SagaStep struct {
	Name       ingestion.PipelineStep
	Execute    func(ctx context.Context, episode ingestion.Episode) error
	Compensate func(ctx context.Context, episode ingestion.Episode) error
}

type SagaOrchestrator struct {
	sagaRepo  port.SagaRepository
	publisher port.EventPublisher
	store     port.StoreClient
	lock      port.GroupLock
	knowledge KnowledgeUsecase
	logger    *zap.Logger
	tracer    trace.Tracer
}

type KnowledgeUsecase interface {
	ExtractEntities(ctx context.Context, episode ingestion.Episode) error
	ResolveEntities(ctx context.Context, episode ingestion.Episode) error
	ExtractEdges(ctx context.Context, episode ingestion.Episode) error
	ResolveEdges(ctx context.Context, episode ingestion.Episode) error
	GenerateEmbeddings(ctx context.Context, episode ingestion.Episode) error
	UpdateCommunity(ctx context.Context, episode ingestion.Episode) error
}

func NewSagaOrchestrator(
	sagaRepo port.SagaRepository,
	publisher port.EventPublisher,
	store port.StoreClient,
	lock port.GroupLock,
	knowledge KnowledgeUsecase,
	logger *zap.Logger,
	tracer trace.Tracer,
) *SagaOrchestrator {
	return &SagaOrchestrator{
		sagaRepo:  sagaRepo,
		publisher: publisher,
		store:     store,
		lock:      lock,
		knowledge: knowledge,
		logger:    logger,
		tracer:    tracer,
	}
}

func (s *SagaOrchestrator) Execute(ctx context.Context, episode ingestion.Episode) error {
	ctx, span := s.tracer.Start(ctx, "SagaOrchestrator.Execute")
	defer span.End()

	s.logger.Info("Starting Saga pipeline", zap.String("episode_id", string(episode.ID)), zap.String("group_id", string(episode.GroupID)))

	unlock, err := s.lock.Acquire(ctx, episode.GroupID)
	if err != nil {
		s.logger.Error("Failed to acquire group lock", zap.Error(err))
		return fmt.Errorf("failed to acquire lock for group %s: %w", episode.GroupID, err)
	}
	defer unlock()

	steps := []SagaStep{
		{Name: ingestion.StepExtractEntities, Execute: s.knowledge.ExtractEntities, Compensate: nil},
		{Name: ingestion.StepResolveEntities, Execute: s.knowledge.ResolveEntities, Compensate: nil},
		{Name: ingestion.StepExtractEdges, Execute: s.knowledge.ExtractEdges, Compensate: nil},
		{Name: ingestion.StepResolveEdges, Execute: s.knowledge.ResolveEdges, Compensate: nil},
		{Name: ingestion.StepGenerateEmbeddings, Execute: s.knowledge.GenerateEmbeddings, Compensate: nil},
		{Name: ingestion.StepSaveBulk, Execute: func(ctx context.Context, ep ingestion.Episode) error {
			return s.store.SaveBulk(ctx, port.SaveBulkRequest{Episode: ep})
		}, Compensate: func(ctx context.Context, ep ingestion.Episode) error {
			s.logger.Warn("Rolling back Store bulk save", zap.String("episode_id", string(ep.ID)))
			return s.store.RollbackBulk(ctx, string(ep.ID))
		}},
		{Name: ingestion.StepUpdateCommunity, Execute: s.knowledge.UpdateCommunity, Compensate: nil},
	}

	return s.executeSteps(ctx, episode, steps)
}

func (s *SagaOrchestrator) executeSteps(ctx context.Context, episode ingestion.Episode, steps []SagaStep) error {
	sagaState := ingestion.Saga{
		EpisodeID: episode.ID,
		GroupID:   episode.GroupID,
		State:     ingestion.SagaStateProcessing,
	}
	s.sagaRepo.Save(ctx, sagaState)

	for _, step := range steps {
		s.logger.Debug("Executing Saga step", zap.String("step", string(step.Name)))
		s.sagaRepo.UpdateState(ctx, string(sagaState.ID), ingestion.SagaStateProcessing, step.Name)
		
		if err := step.Execute(ctx, episode); err != nil {
			s.logger.Error("Saga step failed", zap.String("step", string(step.Name)), zap.Error(err))
			s.sagaRepo.UpdateState(ctx, string(sagaState.ID), ingestion.SagaStateFailed, step.Name)
			
			if step.Compensate != nil {
				if compErr := step.Compensate(ctx, episode); compErr != nil {
					s.logger.Error("Saga compensation failed", zap.Error(compErr))
				}
			}
			
			s.publisher.PublishEpisodeFailed(ctx, ingestion.EpisodeFailed{EpisodeID: string(episode.ID), GroupID: string(episode.GroupID), Reason: err.Error()})
			return fmt.Errorf("saga step %s failed: %w", step.Name, err)
		}
	}
	
	s.logger.Info("Saga completed successfully")
	s.sagaRepo.UpdateState(ctx, string(sagaState.ID), ingestion.SagaStateCompleted, "DONE")
	s.publisher.PublishEpisodeIngested(ctx, ingestion.EpisodeIngested{EpisodeID: string(episode.ID), GroupID: string(episode.GroupID)})
	return nil
}
