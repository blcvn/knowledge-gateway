// Package grpc implements the CogneeIngestionService gRPC handler.
package grpc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/domain"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/usecase/dto"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/usecase/port"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const tenantIDKey = "x-tenant-id"

// Handler implements the CogneeIngestionService gRPC server.
type Handler struct {
	fileIngester   port.FileIngester
	textIngester   port.TextIngester
	urlIngester    port.URLIngester
	datasetManager port.DatasetManager
	logger         *slog.Logger
}

// NewHandler creates a new gRPC handler.
func NewHandler(
	fileIngester port.FileIngester,
	textIngester port.TextIngester,
	urlIngester port.URLIngester,
	datasetManager port.DatasetManager,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		fileIngester:   fileIngester,
		textIngester:   textIngester,
		urlIngester:    urlIngester,
		datasetManager: datasetManager,
		logger:         logger.With("handler", "grpc"),
	}
}

// CreateDataset creates a new dataset for the tenant.
func (h *Handler) CreateDataset(ctx context.Context, req *CreateDatasetRequest) (*DatasetProto, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	ds, err := h.datasetManager.Create(ctx, tenantID, req.Name, req.Description)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return DatasetToProto(ds), nil
}

// DeleteDataset removes a dataset and all its data items.
func (h *Handler) DeleteDataset(ctx context.Context, req *DeleteDatasetRequest) (*emptypb.Empty, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.DatasetId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid dataset_id: %v", err)
	}

	if err := h.datasetManager.Delete(ctx, tenantID, id); err != nil {
		return nil, mapDomainError(err)
	}
	return &emptypb.Empty{}, nil
}

// ListDatasets returns datasets for the tenant with pagination.
func (h *Handler) ListDatasets(ctx context.Context, req *ListDatasetsRequest) (*ListDatasetsResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	datasets, nextCursor, err := h.datasetManager.List(ctx, tenantID, req.Cursor, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list datasets: %v", err)
	}

	protos := make([]*DatasetProto, len(datasets))
	for i, ds := range datasets {
		protos[i] = DatasetToProto(ds)
	}

	return &ListDatasetsResponse{
		Datasets:   protos,
		NextCursor: nextCursor,
	}, nil
}

// GetDatasetStatus returns the current status and metrics for a dataset.
func (h *Handler) GetDatasetStatus(ctx context.Context, req *GetDatasetStatusRequest) (*DatasetStatusProto, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.DatasetId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid dataset_id: %v", err)
	}

	st, err := h.datasetManager.GetStatus(ctx, tenantID, id)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &DatasetStatusProto{
		Id:             st.ID.String(),
		Name:           st.Name,
		Status:         string(st.Status),
		FileCount:      int32(st.FileCount),
		TotalSizeBytes: st.TotalSizeBytes,
		CreatedAt:      timestamppb.New(st.CreatedAt),
		UpdatedAt:      timestamppb.New(st.UpdatedAt),
	}, nil
}

// AddText ingests direct text content.
func (h *Handler) AddText(ctx context.Context, req *AddTextRequest) (*AddTextResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	dsID, err := uuid.Parse(req.DatasetId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid dataset_id: %v", err)
	}

	result, err := h.textIngester.Execute(ctx, dto.IngestTextRequest{
		DatasetID:  dsID,
		TenantID:   tenantID,
		Text:       req.Text,
		SourceName: req.SourceName,
		Metadata:   req.Metadata,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &AddTextResponse{
		ItemId:      result.ItemID.String(),
		SizeBytes:   result.SizeBytes,
		TextPreview: result.TextPreview,
	}, nil
}

// AddUrl ingests content from a URL via scraping.
func (h *Handler) AddUrl(ctx context.Context, req *AddUrlRequest) (*AddUrlResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	dsID, err := uuid.Parse(req.DatasetId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid dataset_id: %v", err)
	}

	result, err := h.urlIngester.Execute(ctx, dto.IngestURLRequest{
		DatasetID: dsID,
		TenantID:  tenantID,
		URL:       req.Url,
		Metadata:  req.Metadata,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &AddUrlResponse{
		ItemId:      result.ItemID.String(),
		SizeBytes:   result.SizeBytes,
		TextPreview: result.TextPreview,
	}, nil
}

// AddData handles streaming file upload.
// The first message must contain DataHeader (filename, mime_type).
// Subsequent messages contain file chunks.
func (h *Handler) AddData(stream AddDataServer) error {
	ctx := stream.Context()
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return err
	}

	// First message must be header
	firstMsg, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive header: %v", err)
	}

	header := firstMsg.GetHeader()
	if header == nil {
		return status.Error(codes.InvalidArgument, "first message must be DataHeader")
	}

	dsID, err := uuid.Parse(header.DatasetId)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid dataset_id: %v", err)
	}

	// Accumulate file chunks
	var buf bytes.Buffer
	var totalSize int64

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "receive chunk: %v", err)
		}

		chunk := msg.GetChunk()
		if chunk == nil {
			continue
		}
		n, _ := buf.Write(chunk)
		totalSize += int64(n)
	}

	result, err := h.fileIngester.Execute(ctx, dto.IngestFileRequest{
		DatasetID: dsID,
		TenantID:  tenantID,
		Filename:  header.Filename,
		MimeType:  domain.MimeType(header.MimeType),
		Reader:    &buf,
		Size:      totalSize,
		Metadata:  header.Metadata,
	})
	if err != nil {
		return mapDomainError(err)
	}

	return stream.SendAndClose(&AddDataResponse{
		ItemId:      result.ItemID.String(),
		SizeBytes:   result.SizeBytes,
		TextPreview: result.TextPreview,
		IsDuplicate: result.IsDuplicate,
	})
}

// extractTenantID extracts the tenant ID from gRPC metadata.
func extractTenantID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}
	vals := md.Get(tenantIDKey)
	if len(vals) == 0 || vals[0] == "" {
		return "", status.Errorf(codes.Unauthenticated, "missing %s metadata", tenantIDKey)
	}
	return vals[0], nil
}

// mapDomainError converts domain errors to gRPC status errors.
func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domain.ErrDatasetNotFound), errors.Is(err, domain.ErrDataItemNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrDuplicateDataset):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrDuplicateFile):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrMissingTenantID),
		errors.Is(err, domain.ErrMissingDatasetName),
		errors.Is(err, domain.ErrMissingDatasetID),
		errors.Is(err, domain.ErrEmptyText),
		errors.Is(err, domain.ErrEmptyURL):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		var unsupported *domain.ErrUnsupportedFormat
		if errors.As(err, &unsupported) {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		var extractionErr *domain.ErrExtractionFailed
		if errors.As(err, &extractionErr) {
			return status.Error(codes.Internal, err.Error())
		}
		var scrapeErr *domain.ErrScrapeFailed
		if errors.As(err, &scrapeErr) {
			return status.Error(codes.Internal, err.Error())
		}
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}
