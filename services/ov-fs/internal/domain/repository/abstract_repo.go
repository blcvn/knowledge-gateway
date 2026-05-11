package repository

import (
	"context"

	"vnp-memory/services/ov-fs/internal/domain/model"
)

type AbstractRepository interface {
	StoreAbstract(ctx context.Context, accountID, path string, level model.ContextLevel, content string) error
	GetAbstract(ctx context.Context, accountID, path string, level model.ContextLevel) (string, error)
}
