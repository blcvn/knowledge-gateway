package usecase_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"vnp-memory/services/cognee-ingestion/internal/domain"
	"vnp-memory/services/cognee-ingestion/internal/usecase"
	"vnp-memory/services/cognee-ingestion/internal/usecase/dto"
	"vnp-memory/services/cognee-ingestion/internal/usecase/port/mock"
)

func TestIngestURLUseCase_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dsRepo := mock.NewMockDatasetRepository(ctrl)
	itemRepo := mock.NewMockDataItemRepository(ctrl)
	scraper := mock.NewMockURLScraper(ctrl)
	eventPub := mock.NewMockEventPublisher(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	uc := usecase.NewIngestURLUseCase(dsRepo, itemRepo, scraper, eventPub, logger)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		req := dto.IngestURLRequest{
			DatasetID: uuid.New(),
			TenantID:  "tenant-1",
			URL:       "https://example.com",
			Metadata:  map[string]string{"key": "value"},
		}

		dsRepo.EXPECT().GetByID(ctx, req.TenantID, req.DatasetID).Return(&domain.Dataset{ID: req.DatasetID, TenantID: req.TenantID}, nil)
		scraper.EXPECT().Scrape(ctx, req.URL).Return("scraped text content", nil)
		itemRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
		dsRepo.EXPECT().IncrementItems(ctx, req.TenantID, req.DatasetID, gomock.Any()).Return(nil)
		eventPub.EXPECT().PublishDataIngested(ctx, gomock.Any()).Return(nil)

		res, err := uc.Execute(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, req.DatasetID, res.DatasetID)
		assert.Equal(t, req.URL, res.Filename)
		assert.Equal(t, "scraped text content", res.TextPreview)
	})

	t.Run("EmptyURL", func(t *testing.T) {
		req := dto.IngestURLRequest{
			DatasetID: uuid.New(),
			TenantID:  "tenant-1",
			URL:       "",
		}

		res, err := uc.Execute(ctx, req)
		assert.ErrorIs(t, err, domain.ErrEmptyURL)
		assert.Nil(t, res)
	})

	t.Run("DatasetNotFound", func(t *testing.T) {
		req := dto.IngestURLRequest{
			DatasetID: uuid.New(),
			TenantID:  "tenant-1",
			URL:       "https://example.com",
		}

		dsRepo.EXPECT().GetByID(ctx, req.TenantID, req.DatasetID).Return(nil, nil)

		res, err := uc.Execute(ctx, req)
		assert.ErrorIs(t, err, domain.ErrDatasetNotFound)
		assert.Nil(t, res)
	})
}
