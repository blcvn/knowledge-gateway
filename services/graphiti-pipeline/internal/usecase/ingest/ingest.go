package ingest

import (
	"context"
	"fmt"

	"graphiti-pipeline/internal/domain/ingestion"
	"graphiti-pipeline/internal/usecase/port"
)

type IngestEpisodeUseCase struct {
	episodeRepo port.EpisodeRepository
	sagaOrch    SagaOrchestrator
}

type SagaOrchestrator interface {
	Execute(ctx context.Context, episode ingestion.Episode) error
}

func NewIngestEpisodeUseCase(repo port.EpisodeRepository, orch SagaOrchestrator) *IngestEpisodeUseCase {
	return &IngestEpisodeUseCase{
		episodeRepo: repo,
		sagaOrch:    orch,
	}
}

func (u *IngestEpisodeUseCase) Execute(ctx context.Context, episode ingestion.Episode) error {
	isDup, err := u.episodeRepo.CheckDuplicate(ctx, episode.ContentHash)
	if err != nil {
		return err
	}
	if isDup {
		return ingestion.ErrDuplicateEpisode
	}

	if err := u.episodeRepo.Save(ctx, episode); err != nil {
		return err
	}

	return u.sagaOrch.Execute(ctx, episode)
}

type BulkIngestUseCase struct {
	episodeRepo port.EpisodeRepository
	sagaOrch    SagaOrchestrator
}

func NewBulkIngestUseCase(repo port.EpisodeRepository, orch SagaOrchestrator) *BulkIngestUseCase {
	return &BulkIngestUseCase{
		episodeRepo: repo,
		sagaOrch:    orch,
	}
}

func (u *BulkIngestUseCase) Execute(ctx context.Context, episodes []ingestion.Episode) error {
	for _, ep := range episodes {
		if err := u.ExecuteSingle(ctx, ep); err != nil {
			// Depending on policy, we might log and continue, or fail fast
			fmt.Printf("failed to ingest episode %s: %v\n", ep.ID, err)
		}
	}
	return nil
}

func (u *BulkIngestUseCase) ExecuteSingle(ctx context.Context, episode ingestion.Episode) error {
	isDup, err := u.episodeRepo.CheckDuplicate(ctx, episode.ContentHash)
	if err != nil {
		return err
	}
	if isDup {
		return nil // skip duplicate
	}

	if err := u.episodeRepo.Save(ctx, episode); err != nil {
		return err
	}

	return u.sagaOrch.Execute(ctx, episode)
}
