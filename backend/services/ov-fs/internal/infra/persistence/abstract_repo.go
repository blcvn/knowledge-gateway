package persistence

import (
	"context"

	"vnp-memory/services/ov-fs/internal/domain/model"
	"vnp-memory/services/ov-fs/internal/domain/repository"
)

type abstractRepo struct {
	// db *sql.DB
}

var _ repository.AbstractRepository = (*abstractRepo)(nil)

func NewAbstractRepo() repository.AbstractRepository {
	return &abstractRepo{}
}

func (r *abstractRepo) StoreAbstract(ctx context.Context, accountID, path string, level model.ContextLevel, content string) error {
	return nil
}

func (r *abstractRepo) GetAbstract(ctx context.Context, accountID, path string, level model.ContextLevel) (string, error) {
	return "", nil
}
