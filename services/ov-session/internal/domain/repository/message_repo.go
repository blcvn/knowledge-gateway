package repository

import (
	"context"

	"github.com/vnp-memory/services/ov-session/internal/domain/model"
)

type MessageRepository interface {
	AddMessage(ctx context.Context, msg *model.Message) error
	GetMessagesBySession(ctx context.Context, sessionID string) ([]*model.Message, error)
}
