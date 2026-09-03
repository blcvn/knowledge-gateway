package grpc

import (
	"context"
	"go.uber.org/zap"
)

// ThreadServiceProxy implements the legacy ZepThreadService gRPC interface.
// It acts as a transparent reverse proxy forwarding requests to `zep-core`.
type ThreadServiceProxy struct {
	coreClient interface{}
	logger     *zap.Logger
}

func NewThreadServiceProxy(coreClient interface{}, logger *zap.Logger) *ThreadServiceProxy {
	return &ThreadServiceProxy{
		coreClient: coreClient,
		logger:     logger,
	}
}

// AddThread intercepts the legacy request and forwards it to zep-core.
func (p *ThreadServiceProxy) AddThread(ctx context.Context, req interface{}) (interface{}, error) {
	p.logger.Warn("DEPRECATED API CALLED: AddThread. Forwarding to zep-core...")
	// Forward request logic...
	return nil, nil
}
