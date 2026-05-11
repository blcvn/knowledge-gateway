package usecase_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/domain"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/usecase"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/usecase/port/mock"
)

func TestManageDatasetUseCase_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dsRepo := mock.NewMockDatasetRepository(ctrl)
	itemRepo := mock.NewMockDataItemRepository(ctrl)
	fileStorage := mock.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	uc := usecase.NewManageDatasetUseCase(dsRepo, itemRepo, fileStorage, logger)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		tenantID := "tenant-1"
		name := "test-dataset"
		desc := "description"

		dsRepo.EXPECT().ExistsByName(ctx, tenantID, name).Return(false, nil)
		dsRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		ds, err := uc.Create(ctx, tenantID, name, desc)
		assert.NoError(t, err)
		assert.NotNil(t, ds)
		assert.Equal(t, tenantID, ds.TenantID)
		assert.Equal(t, name, ds.Name)
	})

	t.Run("Duplicate", func(t *testing.T) {
		tenantID := "tenant-1"
		name := "test-dataset"
		desc := "description"

		dsRepo.EXPECT().ExistsByName(ctx, tenantID, name).Return(true, nil)

		ds, err := uc.Create(ctx, tenantID, name, desc)
		assert.ErrorIs(t, err, domain.ErrDuplicateDataset)
		assert.Nil(t, ds)
	})
}

func TestManageDatasetUseCase_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dsRepo := mock.NewMockDatasetRepository(ctrl)
	itemRepo := mock.NewMockDataItemRepository(ctrl)
	fileStorage := mock.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	uc := usecase.NewManageDatasetUseCase(dsRepo, itemRepo, fileStorage, logger)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		tenantID := "tenant-1"
		id := uuid.New()

		expectedDs := &domain.Dataset{ID: id, TenantID: tenantID}
		dsRepo.EXPECT().GetByID(ctx, tenantID, id).Return(expectedDs, nil)

		ds, err := uc.Get(ctx, tenantID, id)
		assert.NoError(t, err)
		assert.Equal(t, expectedDs, ds)
	})

	t.Run("NotFound", func(t *testing.T) {
		tenantID := "tenant-1"
		id := uuid.New()

		dsRepo.EXPECT().GetByID(ctx, tenantID, id).Return(nil, nil)

		ds, err := uc.Get(ctx, tenantID, id)
		assert.ErrorIs(t, err, domain.ErrDatasetNotFound)
		assert.Nil(t, ds)
	})
}

func TestManageDatasetUseCase_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dsRepo := mock.NewMockDatasetRepository(ctrl)
	itemRepo := mock.NewMockDataItemRepository(ctrl)
	fileStorage := mock.NewMockFileStorage(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	uc := usecase.NewManageDatasetUseCase(dsRepo, itemRepo, fileStorage, logger)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		tenantID := "tenant-1"
		id := uuid.New()

		dsRepo.EXPECT().GetByID(ctx, tenantID, id).Return(&domain.Dataset{ID: id, TenantID: tenantID}, nil)
		fileStorage.EXPECT().DeletePrefix(ctx, gomock.Any()).Return(nil)
		itemRepo.EXPECT().DeleteByDataset(ctx, id).Return(nil)
		dsRepo.EXPECT().Delete(ctx, tenantID, id).Return(nil)

		err := uc.Delete(ctx, tenantID, id)
		assert.NoError(t, err)
	})
}

func TestManageDatasetUseCase_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	dsRepo := mock.NewMockDatasetRepository(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	uc := usecase.NewManageDatasetUseCase(dsRepo, nil, nil, logger)
	ctx := context.Background()

	dsRepo.EXPECT().List(ctx, "tenant-1", "", 20).Return([]*domain.Dataset{{ID: uuid.New()}}, "next", nil)
	ds, next, err := uc.List(ctx, "tenant-1", "", 0)
	assert.NoError(t, err)
	assert.Len(t, ds, 1)
	assert.Equal(t, "next", next)
}

func TestManageDatasetUseCase_GetStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	dsRepo := mock.NewMockDatasetRepository(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	uc := usecase.NewManageDatasetUseCase(dsRepo, nil, nil, logger)
	ctx := context.Background()

	id := uuid.New()
	dsRepo.EXPECT().GetByID(ctx, "tenant-1", id).Return(&domain.Dataset{ID: id, Status: domain.DatasetReady}, nil)
	status, err := uc.GetStatus(ctx, "tenant-1", id)
	assert.NoError(t, err)
	assert.Equal(t, domain.DatasetReady, status.Status)
}
