package repository

import (
	"context"

	"vnp-memory/services/ov-fs/internal/domain/model"
)

type RelationRepository interface {
	GetRelations(ctx context.Context, accountID, path string, relationType *model.RelationType) ([]*model.FileRelation, error)
	AddRelation(ctx context.Context, relation *model.FileRelation) error
	DeleteRelationsByFile(ctx context.Context, accountID, path string) error
}
