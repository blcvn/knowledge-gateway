package repository

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/domain/model"
)

type ProfileReadRepository interface {
	GetProfiles(ctx context.Context, userID, projectID string) ([]*model.Profile, error)
	SearchProfiles(ctx context.Context, userID, projectID string, queryEmbedding []float32, limit int) ([]*model.Profile, error)
}
