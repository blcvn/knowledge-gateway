// Package client provides gRPC client infrastructure for downstream service communication.
package client

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"

	"github.com/vnp-community/vnp-memory/gateway/domain"
	mw "github.com/vnp-community/vnp-memory/gateway/infra/middleware"
)

// GRPCRegistry manages gRPC connections to all 35 downstream services.
// Implements port.ServiceRegistry.
type GRPCRegistry struct {
	conns   map[string]*grpc.ClientConn
	targets map[string]*domain.RouteTarget
	mu      sync.RWMutex
	logger  *slog.Logger
}

// NewGRPCRegistry creates a new registry and establishes connections to all configured services.
func NewGRPCRegistry(services map[string]string, defaultTimeout time.Duration, logger *slog.Logger) (*GRPCRegistry, func(), error) {
	reg := &GRPCRegistry{
		conns:   make(map[string]*grpc.ClientConn, len(services)),
		targets: make(map[string]*domain.RouteTarget, len(services)),
		logger:  logger,
	}

	for svc, addr := range services {
		conn, err := grpc.NewClient(addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                10 * time.Second,
				Timeout:             3 * time.Second,
				PermitWithoutStream: true,
			}),
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(16*1024*1024), // 16MB
			),
		)
		if err != nil {
			// Close already opened connections
			for _, c := range reg.conns {
				c.Close()
			}
			return nil, nil, fmt.Errorf("dial %s (%s): %w", svc, addr, err)
		}
		reg.conns[svc] = conn
		reg.targets[svc] = &domain.RouteTarget{
			Service: svc,
			Address: addr,
			Timeout: defaultTimeout,
		}
		logger.Debug("registered gRPC service", "service", svc, "address", addr)
	}

	cleanup := func() {
		logger.Info("closing all gRPC connections", "count", len(reg.conns))
		for svc, conn := range reg.conns {
			if err := conn.Close(); err != nil {
				logger.Error("failed to close connection", "service", svc, "error", err)
			}
		}
	}

	logger.Info("gRPC registry initialized", "services", len(reg.conns))
	return reg, cleanup, nil
}

// Resolve looks up the RouteTarget for a named service.
func (r *GRPCRegistry) Resolve(service string) (*domain.RouteTarget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	target, ok := r.targets[service]
	if !ok {
		return nil, fmt.Errorf("unknown service: %s", service)
	}
	return target, nil
}

// Forward sends a request to the target service via gRPC and returns the response.
// Deprecated: Use ForwardWithContext for method-level routing.
func (r *GRPCRegistry) Forward(ctx context.Context, target *domain.RouteTarget, req []byte) ([]byte, error) {
	return r.ForwardWithContext(ctx, target, &domain.ForwardRequest{
		Body: req,
	})
}

// ForwardWithContext sends a request with HTTP context to the target service.
// Uses the ForwardService.Forward RPC method. The service routes internally
// based on the path and HTTP method fields.
func (r *GRPCRegistry) ForwardWithContext(ctx context.Context, target *domain.RouteTarget, req *domain.ForwardRequest) ([]byte, error) {
	r.mu.RLock()
	conn, ok := r.conns[target.Service]
	r.mu.RUnlock()
	if !ok {
		return nil, domain.ErrCircuitOpen
	}

	// Propagate tenant metadata to downstream service
	auth, ok := mw.AuthFromContext(ctx)
	if ok && auth != nil {
		ctx = metadata.AppendToOutgoingContext(ctx,
			"x-tenant-id", auth.TenantID,
			"x-user-id", auth.UserID,
			"x-request-id", mw.RequestIDFromContext(ctx),
		)
	}

	// Also propagate the HTTP path as metadata for service-side routing
	ctx = metadata.AppendToOutgoingContext(ctx,
		"x-forward-path", req.Path,
		"x-forward-method", req.HTTPMethod,
	)

	// Apply per-target timeout
	ctx, cancel := context.WithTimeout(ctx, target.Timeout)
	defer cancel()

	// Use the typed ForwardService.Forward RPC method.
	// The gRPC method path follows: /<package>.<ServiceName>/<MethodName>
	var resp []byte
	err := conn.Invoke(ctx, "/vnp.gateway.forward.v1.ForwardService/Forward", req.Body, &resp)
	if err != nil {
		r.logger.Error("forward failed",
			"service", target.Service,
			"path", req.Path,
			"method", req.HTTPMethod,
			"error", err,
		)
		return nil, fmt.Errorf("forward to %s: %w", target.Service, err)
	}

	return resp, nil
}

// HealthCheck returns the health status of a downstream service by checking connection state.
func (r *GRPCRegistry) HealthCheck(service string) (bool, error) {
	r.mu.RLock()
	conn, ok := r.conns[service]
	r.mu.RUnlock()
	if !ok {
		return false, fmt.Errorf("unknown service: %s", service)
	}

	state := conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle, nil
}

// StartHealthCheck runs a background goroutine that periodically checks all service connections.
func (r *GRPCRegistry) StartHealthCheck(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("health check stopped")
			return
		case <-ticker.C:
			r.mu.RLock()
			for svc, conn := range r.conns {
				state := conn.GetState()
				if state != connectivity.Ready && state != connectivity.Idle {
					r.logger.Warn("service unhealthy",
						"service", svc,
						"state", state.String(),
					)
				}
			}
			r.mu.RUnlock()
		}
	}
}

// ConnectedServices returns a list of currently connected service names.
func (r *GRPCRegistry) ConnectedServices() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]string, 0, len(r.conns))
	for svc := range r.conns {
		services = append(services, svc)
	}
	return services
}

// ServiceStatus returns the connection state for each service.
func (r *GRPCRegistry) ServiceStatus() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := make(map[string]string, len(r.conns))
	for svc, conn := range r.conns {
		status[svc] = conn.GetState().String()
	}
	return status
}
