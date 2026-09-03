package usecase_test

import (
	"context"
	"testing"

	"github.com/vnp-memory/services/ov-session/domain/model"
	"github.com/vnp-memory/services/ov-session/mocks"
	"github.com/vnp-memory/services/ov-session/usecase"
	"go.uber.org/mock/gomock"
)

func TestCommitSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionRepo := mocks.NewMockSessionRepository(ctrl)
	mockMsgRepo := mocks.NewMockMessageRepository(ctrl)
	mockLLM := mocks.NewMockLLMClient(ctrl)
	mockFS := mocks.NewMockFSClient(ctrl)
	mockPublisher := mocks.NewMockPublisher(ctrl)

	compUC := usecase.NewCompressorUseCase(mockLLM)
	extUC := usecase.NewMemoryExtractorUseCase(mockLLM)
	dedupUC := usecase.NewMemoryDeduplicatorUseCase(mockLLM)

	uc := usecase.NewCommitUseCase(mockSessionRepo, mockMsgRepo, compUC, extUC, dedupUC, mockFS, mockPublisher)

	sessionID := "sess-1"

	session := &model.Session{
		ID:        sessionID,
		AccountID: "acc-1",
		Status:    model.SessionStatusActive,
	}

	// Phase 1 Expectations
	mockSessionRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(session, nil)
	mockMsgRepo.EXPECT().GetMessagesBySession(gomock.Any(), sessionID).Return([]*model.Message{}, nil)
	mockLLM.EXPECT().CompressSession(gomock.Any(), gomock.Any(), model.CompressionVersionV2).Return("archive_data", nil)
	mockFS.EXPECT().WriteArchive(gomock.Any(), "acc-1", gomock.Any(), "archive_data").Return("viking://acc/archive", nil)
	
	// Expect update session to committed
	mockSessionRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	mockPublisher.EXPECT().PublishSessionCommitted(gomock.Any(), gomock.Any()).Return(nil)

	// Phase 2 Expectations
	mockLLM.EXPECT().ExtractMemories(gomock.Any(), "archive_data").Return([]model.CandidateMemory{
		{Category: model.MemoryCategoryFact, Content: "fact1"},
	}, nil)

	// FS & DB for extracted memory
	mockFS.EXPECT().WriteMemory(gomock.Any(), "acc-1", string(model.MemoryCategoryFact), "fact1").Return("viking://mem", nil)
	mockSessionRepo.EXPECT().SaveMemory(gomock.Any(), gomock.Any()).Return(nil)
	
	// Update session with memory count
	mockSessionRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	mockPublisher.EXPECT().PublishMemoryExtracted(gomock.Any(), gomock.Any()).Return(nil)

	res, err := uc.CommitSession(context.Background(), sessionID, true, model.CompressionVersionV2)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.ArchivePath != "viking://acc/archive" {
		t.Errorf("unexpected archive path: %v", res.ArchivePath)
	}
	if res.MemoriesCount != 1 {
		t.Errorf("expected 1 memory extracted, got %d", res.MemoriesCount)
	}
}
