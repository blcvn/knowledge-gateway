package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/domain"
)

type IngestEpisodeUseCase struct {
	episodeRepo  EpisodeRepo
	orchestrator *SagaOrchestrator
}

func NewIngestEpisodeUseCase(repo EpisodeRepo, orch *SagaOrchestrator) *IngestEpisodeUseCase {
	return &IngestEpisodeUseCase{
		episodeRepo:  repo,
		orchestrator: orch,
	}
}

func (uc *IngestEpisodeUseCase) Execute(ctx context.Context, name, groupID, body string, source domain.EpisodeType, refTime time.Time) (*domain.Episode, error) {
	hash := domain.ComputeContentHash(name, groupID, refTime)

	// Dedup check
	existing, err := uc.episodeRepo.GetByHash(ctx, hash)
	if err == nil && existing != nil {
		return nil, domain.ErrDuplicateEpisode
	}

	episode := &domain.Episode{
		UUID:          uuid.New().String(),
		Name:          name,
		GroupID:       groupID,
		Body:          body,
		Source:        source,
		ReferenceTime: refTime,
		ContentHash:   hash,
		CreatedAt:     time.Now(),
	}

	if err := episode.Validate(); err != nil {
		return nil, err
	}

	if err := uc.episodeRepo.Create(ctx, episode); err != nil {
		return nil, fmt.Errorf("failed to persist episode: %w", err)
	}

	if err := uc.orchestrator.StartIngestionSaga(ctx, episode); err != nil {
		return nil, fmt.Errorf("failed to start saga: %w", err)
	}

	return episode, nil
}
