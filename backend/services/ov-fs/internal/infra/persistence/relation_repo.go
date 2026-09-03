package persistence

import (
	"context"

	"vnp-memory/services/ov-fs/internal/domain/model"
	"vnp-memory/services/ov-fs/internal/domain/repository"
)

type relationRepo struct {
	// db *sql.DB
}

var _ repository.RelationRepository = (*relationRepo)(nil)

func NewRelationRepo() repository.RelationRepository {
	return &relationRepo{}
}

func (r *relationRepo) GetRelations(ctx context.Context, accountID, path string, relationType *model.RelationType) ([]*model.FileRelation, error) {
	return nil, nil
}

func (r *relationRepo) AddRelation(ctx context.Context, relation *model.FileRelation) error {
	return nil
}

func (r *relationRepo) DeleteRelationsByFile(ctx context.Context, accountID, path string) error {
	return nil
}
