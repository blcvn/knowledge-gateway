package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	cogneev1 "github.com/vnp-community/vnp-memory/gateway/gen/go/cognee/v1"
	"github.com/vnp-community/vnp-memory/services/cognee-pipeline/internal/usecase/port"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

var tracer = otel.Tracer("cognee-pipeline/adapter/grpc")

type Handler struct {
	cogneev1.UnimplementedCogneeIngestionServiceServer
	ingestUC port.IngestionUseCase
	cognifyUC port.CognifyUseCase
	logger   *zap.Logger
}

func NewHandler(ingestUC port.IngestionUseCase, cognifyUC port.CognifyUseCase, logger *zap.Logger) *Handler {
	return &Handler{
		ingestUC:  ingestUC,
		cognifyUC: cognifyUC,
		logger:    logger,
	}
}

func (h *Handler) CreateDataset(ctx context.Context, req *cogneev1.CreateDatasetRequest) (*cogneev1.CreateDatasetResponse, error) {
	ctx, span := tracer.Start(ctx, "Handler.CreateDataset")
	defer span.End()

	// In a real multi-tenant scenario, tenantID comes from context metadata
	// tenantID, _ := ctx.Value("tenant_id").(string)
	tenantID := uuid.New() // Placeholder

	ds, err := h.ingestUC.CreateDataset(ctx, tenantID, req.Name, req.Description)
	if err != nil {
		h.logger.Error("Failed to create dataset", zap.Error(err))
		return nil, fmt.Errorf("failed to create dataset: %w", err)
	}

	return &cogneev1.CreateDatasetResponse{
		Id:     ds.ID.String(),
		Status: ds.Status,
	}, nil
}

func (h *Handler) UploadData(ctx context.Context, req *cogneev1.UploadDataRequest) (*cogneev1.UploadDataResponse, error) {
	ctx, span := tracer.Start(ctx, "Handler.UploadData")
	defer span.End()

	dsID, err := uuid.Parse(req.DatasetId)
	if err != nil {
		return nil, fmt.Errorf("invalid dataset ID: %w", err)
	}

	// Assuming SourceURI is handled internally or uploaded to MinIO. 
	// For this snippet, we pass raw data.
	item, err := h.ingestUC.AddDataItem(ctx, dsID, "file", "s3://bucket/placeholder", req.ContentType)
	if err != nil {
		h.logger.Error("Failed to add data item", zap.Error(err))
		return nil, fmt.Errorf("failed to process data: %w", err)
	}

	return &cogneev1.UploadDataResponse{
		Id:     item.ID.String(),
		Status: "ingested", // Pipeline auto-triggers in background
	}, nil
}

func (h *Handler) Cognify(ctx context.Context, req *cogneev1.CognifyRequest) (*cogneev1.CognifyResponse, error) {
	ctx, span := tracer.Start(ctx, "Handler.Cognify")
	defer span.End()

	dsID, err := uuid.Parse(req.DatasetId)
	if err != nil {
		return nil, fmt.Errorf("invalid dataset ID: %w", err)
	}
	tenantID := uuid.New() // Placeholder

	job, err := h.cognifyUC.StartCognify(ctx, tenantID, dsID)
	if err != nil {
		h.logger.Error("Failed to trigger cognify", zap.Error(err))
		return nil, fmt.Errorf("failed to trigger cognify: %w", err)
	}

	return &cogneev1.CognifyResponse{
		TaskId: job.ID.String(),
		Status: string(job.Status),
	}, nil
}
