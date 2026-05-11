package postgres

import (
	"context"

	"graphiti-pipeline/internal/domain/ingestion"
	"graphiti-pipeline/internal/usecase/port"
)

type EpisodeRepo struct {
	// db *sql.DB
}

func NewEpisodeRepo() port.EpisodeRepository {
	return &EpisodeRepo{}
}

func (r *EpisodeRepo) Save(ctx context.Context, episode ingestion.Episode) error {
	// Implement save logic with group_id scope
	return nil
}

func (r *EpisodeRepo) CheckDuplicate(ctx context.Context, hash ingestion.ContentHash) (bool, error) {
	// Implement deduplication logic based on content_hash (HTTP 409 eq)
	return false, nil
}
