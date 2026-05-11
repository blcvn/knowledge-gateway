package repository

import (
	"context"

	"openviking.com/ov-resource/internal/domain/model"
)

type WatchRepository interface {
	Create(ctx context.Context, task *model.WatchTask) error
	Update(ctx context.Context, task *model.WatchTask) error
	Delete(ctx context.Context, id, accountID string) error
	GetActiveTasks(ctx context.Context) ([]*model.WatchTask, error)
	UpdateLastPoll(ctx context.Context, id string) error
}
