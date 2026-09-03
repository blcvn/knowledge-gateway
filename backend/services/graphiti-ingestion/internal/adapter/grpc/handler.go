package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/metadata"
)

type Handler struct {
	// dependencies like usecases
}

func NewHandler() *Handler {
	return &Handler{}
}

func extractTenantID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("missing metadata")
	}
	values := md.Get("x-tenant-id")
	if len(values) == 0 {
		return "", errors.New("missing x-tenant-id")
	}
	return values[0], nil
}

// Example gRPC method implementation
func (h *Handler) IngestEpisode(ctx context.Context, req interface{}) (interface{}, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}
	_ = tenantID
	// Call usecase with tenantID as GroupID
	return nil, nil
}
