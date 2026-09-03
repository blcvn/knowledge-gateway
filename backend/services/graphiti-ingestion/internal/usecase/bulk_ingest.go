package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/domain"
	"golang.org/x/sync/errgroup"
)

type BulkIngestUseCase struct {
	episodeRepo  EpisodeRepo
	orchestrator *SagaOrchestrator
}

func NewBulkIngestUseCase(repo EpisodeRepo, orch *SagaOrchestrator) *BulkIngestUseCase {
	return &BulkIngestUseCase{
		episodeRepo:  repo,
		orchestrator: orch,
	}
}

type BulkIngestInput struct {
	Name          string
	Body          string
	Source        domain.EpisodeType
	ReferenceTime time.Time
}

func (uc *BulkIngestUseCase) Execute(ctx context.Context, groupID string, inputs []BulkIngestInput) ([]*domain.Episode, error) {
	var episodes []*domain.Episode
	
	// Create all episodes
	for _, input := range inputs {
		ep, err := domain.NewEpisode(input.Name, groupID, input.Body, input.Source, input.ReferenceTime)
		if err != nil {
			return nil, err
		}
		
		if err := uc.episodeRepo.Create(ctx, ep); err != nil {
			if err == domain.ErrDuplicateEpisode {
				continue // skip duplicates
			}
			return nil, err
		}
		episodes = append(episodes, ep)
	}

	// Process concurrently with ErrGroup
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(10) // Max 10 concurrent orchestrations

	for i := range episodes {
		ep := episodes[i]
		g.Go(func() error {
			// Trigger saga for each episode
			state, err := uc.orchestrator.StartSaga(gCtx, ep)
			if err != nil {
				return fmt.Errorf("failed to start saga for episode %s: %w", ep.UUID, err)
			}
			ep.SagaID = &state.ID
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return episodes, fmt.Errorf("bulk ingestion encountered errors: %w", err)
	}

	return episodes, nil
}
