package grpc

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	ingestionpb "github.com/vnp-memory/api/proto/cognee/ingestion/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"vnp-memory/services/cognee-ingestion/internal/domain"
	"vnp-memory/services/cognee-ingestion/internal/usecase"
)

// IngestionHandler implements the gRPC IngestionService.
type IngestionHandler struct {
	ingestionpb.UnimplementedIngestionServiceServer
	addDataUC  *usecase.AddDataUseCase
	logger     *slog.Logger
}

// NewIngestionHandler creates a new IngestionHandler.
func NewIngestionHandler(addDataUC *usecase.AddDataUseCase, logger *slog.Logger) *IngestionHandler {
	return &IngestionHandler{addDataUC: addDataUC, logger: logger}
}

// AddData ingests content items into a dataset and emits a NATS event for cognify.
func (h *IngestionHandler) AddData(ctx context.Context, req *ingestionpb.AddDataRequest) (*ingestionpb.AddDataResponse, error) {
	datasetID := uuid.UUID{}
	if req.DatasetId != "" {
		var err error
		datasetID, err = uuid.Parse(req.DatasetId)
		if err != nil { return nil, status.Errorf(codes.InvalidArgument, "invalid dataset_id: %v", err) }
	}

	items := make([]usecase.DataItem, 0, len(req.Items))
	for _, pbItem := range req.Items {
		item := usecase.DataItem{
			Content:     pbItem.Content,
			URL:         pbItem.Url,
			ContentType: domain.ContentType(pbItem.ContentType),
			Metadata:    pbMetadataToMap(pbItem.Metadata),
		}
		if pbItem.Config != nil {
			item.Config = &usecase.DataItemConfig{PDFMode: pbItem.Config.PdfMode}
		}
		items = append(items, item)
	}

	result, err := h.addDataUC.Execute(ctx, usecase.AddDataRequest{
		DatasetID:   datasetID,
		DatasetName: req.DatasetName,
		TenantID:    req.TenantId,
		Items:       items,
		NodeSets:    req.NodeSets, // [NEW] propagate node_sets from proto
	})
	if err != nil { return nil, status.Errorf(codes.Internal, "add data: %v", err) }

	return &ingestionpb.AddDataResponse{
		DatasetId: result.DatasetID,
		EntryIds:  result.EntryIDs,
		Count:     int32(result.Count),
	}, nil
}

// ListDatasets lists all datasets for a tenant.
func (h *IngestionHandler) ListDatasets(ctx context.Context, req *ingestionpb.ListDatasetsRequest) (*ingestionpb.ListDatasetsResponse, error) {
	// Stub — real implementation queries DB
	return &ingestionpb.ListDatasetsResponse{}, nil
}

// ListDataEntries lists entries within a dataset.
func (h *IngestionHandler) ListDataEntries(ctx context.Context, req *ingestionpb.ListDataEntriesRequest) (*ingestionpb.ListDataEntriesResponse, error) {
	// Stub — real implementation queries DB
	return &ingestionpb.ListDataEntriesResponse{}, nil
}

// DeleteDataset deletes a dataset and all associated data.
func (h *IngestionHandler) DeleteDataset(ctx context.Context, req *ingestionpb.DeleteDatasetRequest) (*ingestionpb.DeleteDatasetResponse, error) {
	// Stub — real implementation cascades to Neo4j + Qdrant + PostgreSQL
	return &ingestionpb.DeleteDatasetResponse{Deleted: true, DatasetId: req.DatasetId}, nil
}

// AddDataPoints ingests structured data points with explicit relations.
func (h *IngestionHandler) AddDataPoints(ctx context.Context, req *ingestionpb.AddDataPointsRequest) (*ingestionpb.AddDataPointsResponse, error) {
	// Stub — real implementation converts DataPoints to DataEntries
	return &ingestionpb.AddDataPointsResponse{Upserted: int32(len(req.DataPoints))}, nil
}

// pbMetadataToMap converts a proto string map to map[string]any.
func pbMetadataToMap(m map[string]string) map[string]any {
	if m == nil { return nil }
	out := make(map[string]any, len(m))
	for k, v := range m { out[k] = v }
	return out
}
