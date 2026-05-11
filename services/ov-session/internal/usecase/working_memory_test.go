package usecase_test

import (
	"context"
	"testing"

	"github.com/vnp-memory/services/ov-session/internal/domain/model"
	"github.com/vnp-memory/services/ov-session/internal/mocks"
	"github.com/vnp-memory/services/ov-session/internal/usecase"
	"go.uber.org/mock/gomock"
)

func TestGetWorkingMemory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionRepo := mocks.NewMockSessionRepository(ctrl)
	uc := usecase.NewWorkingMemoryUseCase(mockSessionRepo)

	sessionID := "sess-1"
	expectedWM := &model.WorkingMemory{
		SessionID: sessionID,
		State:     model.WMStateOngoing,
	}

	mockSessionRepo.EXPECT().GetWorkingMemory(gomock.Any(), sessionID).Return(expectedWM, nil)

	wm, err := uc.GetWorkingMemory(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if wm.SessionID != sessionID {
		t.Errorf("expected session id to match")
	}
}

func TestUpdateWorkingMemory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionRepo := mocks.NewMockSessionRepository(ctrl)
	uc := usecase.NewWorkingMemoryUseCase(mockSessionRepo)

	wm := &model.WorkingMemory{
		SessionID: "sess-1",
		State:     model.WMStateOngoing,
	}

	mockSessionRepo.EXPECT().UpdateWorkingMemory(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, updatedWM *model.WorkingMemory) error {
		if updatedWM.UpdatedAt.IsZero() {
			t.Errorf("expected UpdatedAt to be set")
		}
		return nil
	})

	_, err := uc.UpdateWorkingMemory(context.Background(), wm)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
