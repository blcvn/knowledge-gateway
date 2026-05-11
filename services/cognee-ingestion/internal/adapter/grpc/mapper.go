package grpc

import (
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DatasetToProto converts a domain Dataset to its proto representation.
func DatasetToProto(ds *domain.Dataset) *DatasetProto {
	if ds == nil {
		return nil
	}
	return &DatasetProto{
		Id:             ds.ID.String(),
		TenantId:       ds.TenantID,
		Name:           ds.Name,
		Description:    ds.Description,
		Status:         string(ds.Status),
		FileCount:      int32(ds.FileCount),
		TotalSizeBytes: ds.TotalSizeBytes,
		Metadata:       ds.Metadata,
		CreatedAt:      timestamppb.New(ds.CreatedAt),
		UpdatedAt:      timestamppb.New(ds.UpdatedAt),
	}
}
