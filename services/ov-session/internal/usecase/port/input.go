package port

import (
	"context"

	"github.com/vnp-memory/services/ov-session/internal/domain/model"
	"github.com/vnp-memory/services/ov-session/internal/usecase/dto"
)

type SessionUseCase interface {
	CreateSession(ctx context.Context, req dto.CreateSessionReq) (*model.Session, error)
	AddMessage(ctx context.Context, req dto.AddMessageReq) error
	GetMessages(ctx context.Context, req dto.GetMessagesReq) ([]*model.Message, error)
}
