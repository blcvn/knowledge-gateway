package grpc

import (
	"context"
	"go.uber.org/zap"
)

// UserServiceProxy implements the legacy ZepUserService gRPC interface.
// It acts as a transparent reverse proxy forwarding requests to `zep-core`.
type UserServiceProxy struct {
	coreClient interface{}
	logger     *zap.Logger
}

func NewUserServiceProxy(coreClient interface{}, logger *zap.Logger) *UserServiceProxy {
	return &UserServiceProxy{
		coreClient: coreClient,
		logger:     logger,
	}
}

// AddUser intercepts the legacy request and forwards it to zep-core.
func (p *UserServiceProxy) AddUser(ctx context.Context, req interface{}) (interface{}, error) {
	p.logger.Warn("DEPRECATED API CALLED: AddUser. Forwarding to zep-core...")
	// Forward request logic...
	return nil, nil
}
