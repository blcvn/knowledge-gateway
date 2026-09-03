package bus

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	gwDomain "github.com/vnp-community/vnp-memory/gateway/domain"
	gwPort "github.com/vnp-community/vnp-memory/gateway/usecase/port"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type InProcessRegistry struct {
	bus          *GRPCBus
	logger       *slog.Logger
	fallbackAddr string
}

var _ gwPort.ServiceRegistry = (*InProcessRegistry)(nil)

func NewInProcessRegistry(bus *GRPCBus, logger *slog.Logger) *InProcessRegistry {
	return &InProcessRegistry{bus: bus, logger: logger}
}

func (r *InProcessRegistry) WithFallback(externalAddr string) *InProcessRegistry {
	r.fallbackAddr = externalAddr
	return r
}

func (r *InProcessRegistry) Resolve(service string) (*gwDomain.RouteTarget, error) {
	if !r.bus.IsRegistered(service) {
		if r.fallbackAddr != "" {
			return &gwDomain.RouteTarget{
				Service: service,
				Address: r.fallbackAddr,
				Timeout: 30 * time.Second,
			}, nil
		}
		return nil, fmt.Errorf("service %s not registered in-process", service)
	}
	return &gwDomain.RouteTarget{
		Service: service,
		Address: "bufconn://inprocess",
		Timeout: 30 * time.Second,
	}, nil
}

func (r *InProcessRegistry) Forward(ctx context.Context, target *gwDomain.RouteTarget, payload []byte) ([]byte, error) {
	return r.ForwardWithContext(ctx, target, &gwDomain.ForwardRequest{
		Body: payload,
	})
}

// ForwardWithContext sends a request with HTTP context to the target service over the in-process gRPC bus.
func (r *InProcessRegistry) ForwardWithContext(ctx context.Context, target *gwDomain.RouteTarget, req *gwDomain.ForwardRequest) ([]byte, error) {
	conn, err := r.bus.GetConn()
	if err != nil {
		return nil, err
	}

	// Propagate the HTTP path and method as metadata for service-side routing
	ctx = metadata.AppendToOutgoingContext(ctx,
		"x-forward-path", req.Path,
		"x-forward-method", req.HTTPMethod,
	)

	// Use the typed ForwardService.Forward RPC method with protobuf message.
	var reply wrapperspb.BytesValue
	reqBytes := &wrapperspb.BytesValue{Value: req.Body}
	err = conn.Invoke(ctx, "/vnp.gateway.forward.v1.ForwardService/Forward", reqBytes, &reply)
	if err != nil {
		r.logger.Error("in-process forward failed",
			"service", target.Service,
			"path", req.Path,
			"error", err,
		)
		return nil, err
	}
	return reply.Value, nil
}

func (r *InProcessRegistry) HealthCheck(service string) (bool, error) {
	return r.bus.IsRegistered(service), nil
}
