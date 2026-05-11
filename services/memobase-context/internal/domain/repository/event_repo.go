package repository

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/domain/model"
)

type EventGistSearchRepository interface {
	SearchBySimilarity(ctx context.Context, userID, projectID string, threshold float32, windowDays int, limit int) ([]*model.EventGist, error)
}
