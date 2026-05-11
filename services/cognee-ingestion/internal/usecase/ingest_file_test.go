package usecase_test

import (
	"bytes"
	"context"
	"errors"
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

func TestIngestFileUseCase_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dsRepo := mock.NewMockDatasetRepository(ctrl)
	itemRepo := mock.NewMockDataItemRepository(ctrl)
	fileStorage := mock.NewMockFileStorage(ctrl)
	extractor := mock.NewMockTextExtractor(ctrl)
	eventPub := mock.NewMockEventPublisher(ctrl)
	hashComputer := mock.NewMockHashComputer(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	uc := usecase.NewIngestFileUseCase(dsRepo, itemRepo, fileStorage, extractor, eventPub, hashComputer, logger)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		req := dto.IngestFileRequest{
			DatasetID: uuid.New(),
			TenantID:  "tenant-1",
			Filename:  "test.txt",
			MimeType:  domain.MimePlainText,
			Size:      100,
			Reader:    bytes.NewReader([]byte("test content")),
			Metadata:  map[string]string{"key": "value"},
		}

		dsRepo.EXPECT().GetByID(ctx, req.TenantID, req.DatasetID).Return(&domain.Dataset{ID: req.DatasetID, TenantID: req.TenantID}, nil)
		hashComputer.EXPECT().ComputeSHA256(req.Reader).Return("hash123", req.Reader, nil)
		itemRepo.EXPECT().ExistsByHash(ctx, req.DatasetID, "hash123").Return(false, nil)
		fileStorage.EXPECT().Upload(ctx, gomock.Any(), gomock.Any(), req.Size).Return("path/to/storage", nil)
		extractor.EXPECT().Extract(ctx, gomock.Any(), req.MimeType).Return("extracted text", nil)
		itemRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
		dsRepo.EXPECT().IncrementItems(ctx, req.TenantID, req.DatasetID, req.Size).Return(nil)
		eventPub.EXPECT().PublishDataIngested(ctx, gomock.Any()).Return(nil)

		res, err := uc.Execute(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.False(t, res.IsDuplicate)
		assert.Equal(t, req.DatasetID, res.DatasetID)
		assert.Equal(t, "extracted text", res.TextPreview)
	})

	t.Run("DatasetNotFound", func(t *testing.T) {
		req := dto.IngestFileRequest{
			DatasetID: uuid.New(),
			TenantID:  "tenant-1",
			Filename:  "test.txt",
			MimeType:  domain.MimePlainText,
			Size:      100,
			Reader:    bytes.NewReader([]byte("test content")),
		}

		dsRepo.EXPECT().GetByID(ctx, req.TenantID, req.DatasetID).Return(nil, nil)

		res, err := uc.Execute(ctx, req)
		assert.ErrorIs(t, err, domain.ErrDatasetNotFound)
		assert.Nil(t, res)
	})

	t.Run("DuplicateFile", func(t *testing.T) {
		req := dto.IngestFileRequest{
			DatasetID: uuid.New(),
			TenantID:  "tenant-1",
			Filename:  "test.txt",
			MimeType:  domain.MimePlainText,
			Size:      100,
			Reader:    bytes.NewReader([]byte("test content")),
		}

		dsRepo.EXPECT().GetByID(ctx, req.TenantID, req.DatasetID).Return(&domain.Dataset{ID: req.DatasetID, TenantID: req.TenantID}, nil)
		hashComputer.EXPECT().ComputeSHA256(req.Reader).Return("hash123", req.Reader, nil)
		itemRepo.EXPECT().ExistsByHash(ctx, req.DatasetID, "hash123").Return(true, nil)

		res, err := uc.Execute(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.True(t, res.IsDuplicate)
	})

	t.Run("UnsupportedMimeType", func(t *testing.T) {
		req := dto.IngestFileRequest{
			DatasetID: uuid.New(),
			TenantID:  "tenant-1",
			Filename:  "test.exe",
			MimeType:  domain.MimeType("application/x-msdownload"),
		}
		
		dsRepo.EXPECT().GetByID(ctx, req.TenantID, req.DatasetID).Return(&domain.Dataset{ID: req.DatasetID, TenantID: req.TenantID}, nil)

		res, err := uc.Execute(ctx, req)
		var unsupportedErr *domain.ErrUnsupportedFormat
		assert.True(t, errors.As(err, &unsupportedErr))
		assert.Nil(t, res)
	})
}

func TestIngestFileUseCase_Errors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dsRepo := mock.NewMockDatasetRepository(ctrl)
	itemRepo := mock.NewMockDataItemRepository(ctrl)
	fileStorage := mock.NewMockFileStorage(ctrl)
	extractor := mock.NewMockTextExtractor(ctrl)
	eventPub := mock.NewMockEventPublisher(ctrl)
	hashComputer := mock.NewMockHashComputer(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	uc := usecase.NewIngestFileUseCase(dsRepo, itemRepo, fileStorage, extractor, eventPub, hashComputer, logger)
	ctx := context.Background()

	t.Run("GetByIDError", func(t *testing.T) {
		req := dto.IngestFileRequest{DatasetID: uuid.New(), TenantID: "tenant-1"}
		dsRepo.EXPECT().GetByID(ctx, req.TenantID, req.DatasetID).Return(nil, errors.New("db error"))
		_, err := uc.Execute(ctx, req)
		assert.ErrorContains(t, err, "db error")
	})

	t.Run("ComputeHashError", func(t *testing.T) {
		req := dto.IngestFileRequest{DatasetID: uuid.New(), TenantID: "tenant-1", MimeType: domain.MimePlainText, Reader: bytes.NewReader([]byte{})}
		dsRepo.EXPECT().GetByID(ctx, req.TenantID, req.DatasetID).Return(&domain.Dataset{}, nil)
		hashComputer.EXPECT().ComputeSHA256(req.Reader).Return("", nil, errors.New("hash error"))
		_, err := uc.Execute(ctx, req)
		assert.ErrorContains(t, err, "hash error")
	})

	t.Run("UploadError", func(t *testing.T) {
		req := dto.IngestFileRequest{DatasetID: uuid.New(), TenantID: "tenant-1", MimeType: domain.MimePlainText, Reader: bytes.NewReader([]byte{})}
		dsRepo.EXPECT().GetByID(ctx, req.TenantID, req.DatasetID).Return(&domain.Dataset{}, nil)
		hashComputer.EXPECT().ComputeSHA256(req.Reader).Return("hash123", req.Reader, nil)
		itemRepo.EXPECT().ExistsByHash(ctx, req.DatasetID, "hash123").Return(false, nil)
		fileStorage.EXPECT().Upload(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return("", errors.New("upload error"))
		_, err := uc.Execute(ctx, req)
		assert.ErrorContains(t, err, "upload error")
	})
}
