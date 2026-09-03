package grpc_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"vnp-memory/services/cognee-ingestion/internal/adapter/grpc"
	"vnp-memory/services/cognee-ingestion/internal/domain"
	"vnp-memory/services/cognee-ingestion/internal/usecase/dto"
	"vnp-memory/services/cognee-ingestion/internal/usecase/port/mock"
)

func TestHandler_CreateDataset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileMock := mock.NewMockFileIngester(ctrl)
	textMock := mock.NewMockTextIngester(ctrl)
	urlMock := mock.NewMockURLIngester(ctrl)
	dsMock := mock.NewMockDatasetManager(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := grpc.NewHandler(fileMock, textMock, urlMock, dsMock, logger)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-tenant-id", "tenant-1"))

	t.Run("Success", func(t *testing.T) {
		req := &grpc.CreateDatasetRequest{
			Name:        "ds1",
			Description: "desc",
		}

		expectedDs := &domain.Dataset{
			ID:          uuid.New(),
			TenantID:    "tenant-1",
			Name:        "ds1",
			Description: "desc",
			Status:      domain.DatasetPending,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		dsMock.EXPECT().Create(ctx, "tenant-1", "ds1", "desc").Return(expectedDs, nil)

		res, err := handler.CreateDataset(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, expectedDs.ID.String(), res.Id)
	})

	t.Run("MissingTenantID", func(t *testing.T) {
		req := &grpc.CreateDatasetRequest{Name: "ds1"}
		res, err := handler.CreateDataset(context.Background(), req)
		assert.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		assert.Nil(t, res)
	})

	t.Run("DuplicateName", func(t *testing.T) {
		req := &grpc.CreateDatasetRequest{Name: "ds1"}
		dsMock.EXPECT().Create(ctx, "tenant-1", "ds1", "").Return(nil, domain.ErrDuplicateDataset)
		res, err := handler.CreateDataset(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, codes.AlreadyExists, status.Code(err))
		assert.Nil(t, res)
	})
}

func TestHandler_AddText(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	textMock := mock.NewMockTextIngester(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := grpc.NewHandler(nil, textMock, nil, nil, logger)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-tenant-id", "tenant-1"))

	t.Run("Success", func(t *testing.T) {
		dsID := uuid.New()
		req := &grpc.AddTextRequest{
			DatasetId: dsID.String(),
			Text:      "test text",
		}

		textMock.EXPECT().Execute(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, r dto.IngestTextRequest) (*dto.IngestResult, error) {
			assert.Equal(t, "tenant-1", r.TenantID)
			assert.Equal(t, dsID, r.DatasetID)
			assert.Equal(t, "test text", r.Text)
			return &dto.IngestResult{
				ItemID:      uuid.New(),
				SizeBytes:   100,
				TextPreview: "test",
			}, nil
		})

		res, err := handler.AddText(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, int64(100), res.SizeBytes)
	})
}
