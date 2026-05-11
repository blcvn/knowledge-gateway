package port

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/domain/model"
)

type ProfileCache interface {
	GetProfiles(ctx context.Context, userID, projectID string) ([]*model.Profile, error)
	SetProfiles(ctx context.Context, userID, projectID string, profiles []*model.Profile, ttlSeconds int) error
	DeleteProfiles(ctx context.Context, userID, projectID string) error
}

type Embedder interface {
	EmbedQuery(ctx context.Context, query string) ([]float32, error)
}
