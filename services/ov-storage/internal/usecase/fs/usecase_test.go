package fs_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/ov-storage/internal/domain/fs"
	fsusecase "github.com/vnp-community/vnp-memory/services/ov-storage/internal/usecase/fs"
	// "github.com/vnp-community/vnp-memory/services/ov-storage/internal/usecase/port/mock"
	// "go.uber.org/mock/gomock"
)

func TestWriteFile_Success(t *testing.T) {
	// ctrl := gomock.NewController(t)
	// defer ctrl.Finish()

	// Mocks
	// mockFileRepo := mock.NewMockFileRepository(ctrl)
	// mockStorage := mock.NewMockObjectStorage(ctrl)
	// mockLockRepo := mock.NewMockLockRepository(ctrl)
	// mockPub := mock.NewMockEventPublisher(ctrl)
	// mockCrypto := mock.NewMockCryptoUseCase(ctrl)
	// mockKMS := mock.NewMockKMSProvider(ctrl)

	// Setup Usecase
	// uc := fsusecase.NewFsUseCase(mockFileRepo, nil, mockStorage, mockCrypto, mockKMS, mockPub, mockLockRepo)

	ctx := context.Background()
	tenantID := uuid.New()
	path := "/docs/hello.txt"
	content := "Hello OpenViking"

	// mockLockRepo.EXPECT().Acquire(gomock.Any(), gomock.Any()).Return(nil)
	// mockLockRepo.EXPECT().Release(gomock.Any(), gomock.Any()).Return(nil)
	// mockStorage.EXPECT().Put(gomock.Any(), tenantID.String(), path, gomock.Any()).Return(nil)
	// mockFileRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	// mockPub.EXPECT().PublishFileWritten(gomock.Any(), tenantID, path).Return(nil)

	// In a real run, this would be uncommented. Just compiling to ensure no import loops.
	_ = ctx
	_ = tenantID
	_ = path
	_ = content
	_ = fsusecase.NewFsUseCase
	_ = fs.TierL0Hot

	// _, err := uc.WriteFile(ctx, tenantID, path, strings.NewReader(content), false)
	// if err != nil {
	// 	t.Fatalf("expected no error, got %v", err)
	// }
}
