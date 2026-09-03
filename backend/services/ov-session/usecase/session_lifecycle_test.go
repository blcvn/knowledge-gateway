package usecase_test

import (
	"context"
	"testing"

	"github.com/vnp-memory/services/ov-session/domain/model"
	"github.com/vnp-memory/services/ov-session/mocks"
	"github.com/vnp-memory/services/ov-session/usecase"
	"github.com/vnp-memory/services/ov-session/usecase/dto"
	"go.uber.org/mock/gomock"
)

func TestCreateSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionRepo := mocks.NewMockSessionRepository(ctrl)
	mockMsgRepo := mocks.NewMockMessageRepository(ctrl)

	uc := usecase.NewSessionUseCase(mockSessionRepo, mockMsgRepo)

	req := dto.CreateSessionReq{
		AccountID: "tenant-1",
		UserID:    "user-1",
		Title:     "Test Session",
	}

	mockSessionRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	mockSessionRepo.EXPECT().UpdateWorkingMemory(gomock.Any(), gomock.Any()).Return(nil)

	session, err := uc.CreateSession(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if session.AccountID != "tenant-1" {
		t.Errorf("expected account id to be tenant-1")
	}
	if session.AgentID != "default" {
		t.Errorf("expected default agent ID")
	}
}

func TestAddMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionRepo := mocks.NewMockSessionRepository(ctrl)
	mockMsgRepo := mocks.NewMockMessageRepository(ctrl)

	uc := usecase.NewSessionUseCase(mockSessionRepo, mockMsgRepo)

	sessionID := "sess-1"

	mockSessionRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&model.Session{
		ID:     sessionID,
		Status: model.SessionStatusActive,
	}, nil)

	mockMsgRepo.EXPECT().GetMessagesBySession(gomock.Any(), sessionID).Return([]*model.Message{}, nil)
	mockMsgRepo.EXPECT().AddMessage(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, msg *model.Message) error {
		if msg.Sequence != 1 {
			t.Errorf("expected sequence to be 1, got %d", msg.Sequence)
		}
		return nil
	})

	err := uc.AddMessage(context.Background(), dto.AddMessageReq{
		SessionID: sessionID,
		Role:      model.MessageRoleUser,
		Content:   "Hello",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
