// Package gateway provides gRPC-to-REST proxying for the embedded gateway.
//
// It connects to embedded gRPC services running on localhost and forwards
// incoming REST requests. This replicates the gateway/internal/adapter/client
// registry.go pattern without modifying the original gateway code.
package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// GRPCRegistry manages gRPC client connections to embedded services.
// It replicates gateway/internal/adapter/client/registry.go patterns,
// connecting to localhost instead of remote Kubernetes DNS addresses.
type GRPCRegistry struct {
	mu       sync.RWMutex
	conns    map[string]*grpc.ClientConn
	services map[string]string // service name → localhost:port
	logger   *slog.Logger
}

// NewGRPCRegistry creates a registry connected to the given service map.
// The map keys are service names and values are "localhost:PORT" addresses.
//
// Example:
//
//	services := map[string]string{
//	    "cognee-ingestion": "localhost:9011",
//	    "cognee-cognify":   "localhost:9012",
//	    "cognee-search":    "localhost:9013",
//	}
func NewGRPCRegistry(services map[string]string, logger *slog.Logger) *GRPCRegistry {
	return &GRPCRegistry{
		conns:    make(map[string]*grpc.ClientConn),
		services: services,
		logger:   logger.With("component", "grpc-registry"),
	}
}

// Connect establishes gRPC connections to all registered services.
// Uses insecure transport since all connections are to localhost.
func (r *GRPCRegistry) Connect(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, addr := range r.services {
		r.logger.Info("connecting to embedded service", "service", name, "addr", addr)

		conn, err := grpc.NewClient(addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(64*1024*1024), // 64MB for large datasets
			),
		)
		if err != nil {
			return fmt.Errorf("dial %s at %s: %w", name, addr, err)
		}

		r.conns[name] = conn
		r.logger.Info("connected to embedded service", "service", name, "addr", addr)
	}

	return nil
}

// GetConn returns the gRPC connection for a service.
func (r *GRPCRegistry) GetConn(serviceName string) (*grpc.ClientConn, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	conn, ok := r.conns[serviceName]
	if !ok {
		return nil, fmt.Errorf("service %q not registered in gRPC registry", serviceName)
	}
	return conn, nil
}

// HealthCheckAll probes all registered services via gRPC health check.
func (r *GRPCRegistry) HealthCheckAll(ctx context.Context) map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[string]string, len(r.conns))
	for name, conn := range r.conns {
		client := grpc_health_v1.NewHealthClient(conn)
		checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		resp, err := client.Check(checkCtx, &grpc_health_v1.HealthCheckRequest{
			Service: name,
		})
		cancel()

		if err != nil {
			results[name] = "unhealthy: " + err.Error()
		} else {
			results[name] = resp.Status.String()
		}
	}
	return results
}

// Close closes all gRPC connections.
func (r *GRPCRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var firstErr error
	for name, conn := range r.conns {
		if err := conn.Close(); err != nil {
			r.logger.Error("failed to close gRPC connection", "service", name, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	r.conns = make(map[string]*grpc.ClientConn)
	return firstErr
}
