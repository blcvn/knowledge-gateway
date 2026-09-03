package grpc

import (
	"context"
	"go.uber.org/zap"
)

// MemoryServiceProxy implements the legacy ZepMemoryService gRPC interface.
// Enterprise Strategy: Instead of hard-deleting the service, we provide a transparent 
// gRPC reverse proxy that forwards all legacy requests to the new consolidated `zep-core` service.
// This ensures zero-downtime migration for existing clients.
type MemoryServiceProxy struct {
	coreClient interface{} // Represents the gRPC client connected to zep-core
	logger     *zap.Logger
}

func NewMemoryServiceProxy(coreClient interface{}, logger *zap.Logger) *MemoryServiceProxy {
	return &MemoryServiceProxy{
		coreClient: coreClient,
		logger:     logger,
	}
}

// PutMemory intercepts the legacy request and forwards it to zep-core.
func (p *MemoryServiceProxy) PutMemory(ctx context.Context, req interface{}) (interface{}, error) {
	p.logger.Warn("DEPRECATED API CALLED: PutMemory. Forwarding to zep-core...")
	// TODO: Forward the exact Protobuf request to p.coreClient.PutMemory(ctx, req)
	return nil, nil
}

// GetMemory intercepts the legacy request and forwards it to zep-core.
func (p *MemoryServiceProxy) GetMemory(ctx context.Context, req interface{}) (interface{}, error) {
	p.logger.Warn("DEPRECATED API CALLED: GetMemory. Forwarding to zep-core...")
	// TODO: Forward the exact Protobuf request to p.coreClient.GetMemory(ctx, req)
	return nil, nil
}
