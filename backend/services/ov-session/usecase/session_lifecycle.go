package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-memory/services/ov-session/domain"
	"github.com/vnp-memory/services/ov-session/domain/model"
	"github.com/vnp-memory/services/ov-session/domain/repository"
	"github.com/vnp-memory/services/ov-session/usecase/dto"
	"github.com/vnp-memory/services/ov-session/usecase/port"
)

type sessionUseCaseImpl struct {
	sessionRepo repository.SessionRepository
	messageRepo repository.MessageRepository
}

func NewSessionUseCase(sessionRepo repository.SessionRepository, messageRepo repository.MessageRepository) port.SessionUseCase {
	return &sessionUseCaseImpl{
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
	}
}

func (uc *sessionUseCaseImpl) CreateSession(ctx context.Context, req dto.CreateSessionReq) (*model.Session, error) {
	agentID := req.AgentID
	if agentID == "" {
		agentID = "default"
	}

	session := &model.Session{
		ID:                 uuid.New().String(),
		AccountID:          req.AccountID,
		UserID:             req.UserID,
		AgentID:            agentID,
		Title:              req.Title,
		Status:             model.SessionStatusActive,
		CompressionVersion: string(model.CompressionVersionV2),
		Metadata:           req.Metadata,
		CreatedAt:          time.Now(),
	}

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	// Initialize WM
	wm := &model.WorkingMemory{
		SessionID: session.ID,
		Title:     "New Session",
		State:     model.WMStateOngoing,
		Goals:     []string{},
		Facts:     []model.Fact{},
		Errors:    []model.ErrorState{},
		Context:   map[string]interface{}{},
		UpdatedAt: time.Now(),
	}
	_ = uc.sessionRepo.UpdateWorkingMemory(ctx, wm)

	return session, nil
}

func (uc *sessionUseCaseImpl) AddMessage(ctx context.Context, req dto.AddMessageReq) error {
	session, err := uc.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		return err
	}
	if session.Status != model.SessionStatusActive {
		return domain.ErrAlreadyCommitted
	}

	messages, err := uc.messageRepo.GetMessagesBySession(ctx, req.SessionID)
	if err != nil {
		return err
	}

	seq := len(messages) + 1

	msg := &model.Message{
		ID:         uuid.New().String(),
		SessionID:  req.SessionID,
		Role:       req.Role,
		Content:    req.Content,
		ToolCalls:  req.ToolCalls,
		TokenCount: req.TokenCount,
		Sequence:   seq,
		CreatedAt:  time.Now(),
	}

	return uc.messageRepo.AddMessage(ctx, msg)
}

func (uc *sessionUseCaseImpl) GetMessages(ctx context.Context, req dto.GetMessagesReq) ([]*model.Message, error) {
	return uc.messageRepo.GetMessagesBySession(ctx, req.SessionID)
}
