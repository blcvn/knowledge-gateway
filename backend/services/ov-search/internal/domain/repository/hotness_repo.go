package repository

import (
	"context"

	"vnp-memory/ov-search/internal/domain/model"
)

type HotnessRepository interface {
	Get(ctx context.Context, accountID, path string) (*model.HotnessScore, error)
	GetMulti(ctx context.Context, accountID string, paths []string) (map[string]*model.HotnessScore, error)
	Save(ctx context.Context, score *model.HotnessScore) error
	GetAll(ctx context.Context) ([]*model.HotnessScore, error) // For periodic recompute
}
