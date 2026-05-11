package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/domain"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type SagaOrchestrator struct {
	knowledgeClient KnowledgeClient
	storeClient     StoreClient
	sagaRepo        SagaStateRepo
	eventPublisher  EventPublisher
}

func NewSagaOrchestrator(
	kClient KnowledgeClient,
	sClient StoreClient,
	sagaRepo SagaStateRepo,
	pub EventPublisher,
) *SagaOrchestrator {
	return &SagaOrchestrator{
		knowledgeClient: kClient,
		storeClient:     sClient,
		sagaRepo:        sagaRepo,
		eventPublisher:  pub,
	}
}

func (o *SagaOrchestrator) StartIngestionSaga(ctx context.Context, episode *domain.Episode) error {
	sagaID := uuid.New().String()
	episode.SagaID = &sagaID

	state := &domain.SagaState{
		ID:          sagaID,
		EpisodeID:   episode.UUID,
		GroupID:     episode.GroupID,
		CurrentStep: domain.StepExtractEntities,
		Status:      domain.SagaStatusRunning,
		StartedAt:   time.Now(),
	}

	if err := o.sagaRepo.Create(ctx, state); err != nil {
		return fmt.Errorf("failed to create saga state: %w", err)
	}

	go o.executeSaga(context.Background(), state, episode)
	return nil
}

func (o *SagaOrchestrator) executeSaga(ctx context.Context, state *domain.SagaState, episode *domain.Episode) {
	tracer := otel.Tracer("graphiti-ingestion/orchestrator")
	ctx, span := tracer.Start(ctx, "SagaOrchestrator.ExecuteSaga")
	defer span.End()

	logger := zap.L().With(zap.String("saga_id", state.ID), zap.String("episode_id", episode.UUID))
	logger.Info("Starting saga execution")

	var err error

	// 1. Extract Entities
	var entities []map[string]interface{}
	if err = o.updateState(ctx, state, domain.StepExtractEntities, domain.SagaStatusRunning, ""); err == nil {
		entities, err = o.knowledgeClient.ExtractEntities(ctx, *episode)
	}

	// 2. Resolve Entities
	if err == nil {
		if err = o.updateState(ctx, state, domain.StepResolveEntities, domain.SagaStatusRunning, ""); err == nil {
			err = o.knowledgeClient.ResolveEntities(ctx, episode.GroupID, entities)
		}
	}

	// 3. Extract Edges
	var edges []map[string]interface{}
	if err == nil {
		if err = o.updateState(ctx, state, domain.StepExtractEdges, domain.SagaStatusRunning, ""); err == nil {
			edges, err = o.knowledgeClient.ExtractEdges(ctx, *episode, entities)
		}
	}

	// 4. Resolve Edges
	if err == nil {
		if err = o.updateState(ctx, state, domain.StepResolveEdges, domain.SagaStatusRunning, ""); err == nil {
			err = o.knowledgeClient.ResolveEdges(ctx, episode.GroupID, edges)
		}
	}

	// 5. Save Bulk
	if err == nil {
		if err = o.updateState(ctx, state, domain.StepSaveBulk, domain.SagaStatusRunning, ""); err == nil {
			data := map[string]interface{}{
				"entities": entities,
				"edges":    edges,
			}
			err = o.storeClient.SaveBulk(ctx, episode.GroupID, data)
		}
	}

	// 6. Update Community
	if err == nil {
		if err = o.updateState(ctx, state, domain.StepUpdateCommunity, domain.SagaStatusRunning, ""); err == nil {
			err = o.knowledgeClient.UpdateCommunity(ctx, episode.GroupID)
		}
	}

	if err != nil {
		logger.Error("Saga execution failed", zap.Error(err), zap.String("step", string(state.CurrentStep)))
		o.compensateSaga(ctx, state, err)
		return
	}

	// Complete Saga
	if err = o.updateState(ctx, state, domain.StepCompleted, domain.SagaStatusCompleted, ""); err == nil {
		logger.Info("Saga completed successfully")
		o.eventPublisher.Publish(ctx, domain.EpisodeIngested{
			EpisodeID:  episode.UUID,
			GroupID:    episode.GroupID,
			NodesCount: len(entities),
			EdgesCount: len(edges),
			Timestamp:  time.Now(),
		})
	}
}

func (o *SagaOrchestrator) compensateSaga(ctx context.Context, state *domain.SagaState, failErr error) {
	tracer := otel.Tracer("graphiti-ingestion/orchestrator")
	ctx, span := tracer.Start(ctx, "SagaOrchestrator.CompensateSaga")
	defer span.End()

	logger := zap.L().With(zap.String("saga_id", state.ID))
	logger.Info("Starting saga compensation")

	o.updateState(ctx, state, state.CurrentStep, domain.SagaStatusFailed, failErr.Error())
	
	// Perform rollback on store
	err := o.storeClient.RollbackBulk(ctx, state.GroupID, state.ID)
	
	finalStatus := domain.SagaStatusRollback
	errMsg := failErr.Error()
	if err != nil {
		errMsg = fmt.Sprintf("%s; rollback failed: %v", errMsg, err)
	}

	o.updateState(ctx, state, state.CurrentStep, finalStatus, errMsg)

	o.eventPublisher.Publish(ctx, domain.EpisodeFailed{
		EpisodeID: state.EpisodeID,
		GroupID:   state.GroupID,
		Reason:    errMsg,
		Timestamp: time.Now(),
	})
}

func (o *SagaOrchestrator) updateState(ctx context.Context, state *domain.SagaState, step domain.PipelineStep, status domain.SagaStatus, errMsg string) error {
	if err := state.Transition(step, status, errMsg); err != nil {
		return err
	}
	return o.sagaRepo.Update(ctx, state)
}
