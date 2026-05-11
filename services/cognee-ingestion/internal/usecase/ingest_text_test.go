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
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/usecase/dto"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/usecase/port/mock"
)

func TestIngestTextUseCase_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dsRepo := mock.NewMockDatasetRepository(ctrl)
	itemRepo := mock.NewMockDataItemRepository(ctrl)
	eventPub := mock.NewMockEventPublisher(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	uc := usecase.NewIngestTextUseCase(dsRepo, itemRepo, eventPub, logger)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		req := dto.IngestTextRequest{
			DatasetID:  uuid.New(),
			TenantID:   "tenant-1",
			Text:       "sample text content",
			SourceName: "test-source",
			Metadata:   map[string]string{"key": "value"},
		}

		dsRepo.EXPECT().GetByID(ctx, req.TenantID, req.DatasetID).Return(&domain.Dataset{ID: req.DatasetID, TenantID: req.TenantID}, nil)
		itemRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
		dsRepo.EXPECT().IncrementItems(ctx, req.TenantID, req.DatasetID, gomock.Any()).Return(nil)
		eventPub.EXPECT().PublishDataIngested(ctx, gomock.Any()).Return(nil)

		res, err := uc.Execute(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, req.DatasetID, res.DatasetID)
		assert.Equal(t, "sample text content", res.TextPreview)
	})

	t.Run("EmptyText", func(t *testing.T) {
		req := dto.IngestTextRequest{
			DatasetID: uuid.New(),
			TenantID:  "tenant-1",
			Text:      "",
		}

		res, err := uc.Execute(ctx, req)
		assert.ErrorIs(t, err, domain.ErrEmptyText)
		assert.Nil(t, res)
	})

	t.Run("DatasetNotFound", func(t *testing.T) {
		req := dto.IngestTextRequest{
			DatasetID: uuid.New(),
			TenantID:  "tenant-1",
			Text:      "test",
		}

		dsRepo.EXPECT().GetByID(ctx, req.TenantID, req.DatasetID).Return(nil, nil)

		res, err := uc.Execute(ctx, req)
		assert.ErrorIs(t, err, domain.ErrDatasetNotFound)
		assert.Nil(t, res)
	})
}
