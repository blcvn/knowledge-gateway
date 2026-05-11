package repository

import (
	"context"

	"openviking.com/ov-resource/internal/domain/model"
)

type ResourceRepository interface {
	Create(ctx context.Context, resource *model.Resource) error
	Update(ctx context.Context, resource *model.Resource) error
	GetByID(ctx context.Context, id, accountID string) (*model.Resource, error)
	GetByHash(ctx context.Context, accountID, hash string) (*model.Resource, error)
	UpdateStatus(ctx context.Context, id, accountID string, status model.ResourceStatus, errorMessage string) error
}
