package bus

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	gwDomain "github.com/vnp-community/vnp-memory/gateway/domain"
	gwPort "github.com/vnp-community/vnp-memory/gateway/usecase/port"
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
	conn, err := r.bus.GetConn()
	if err != nil {
		return nil, err
	}

	var reply []byte
	err = conn.Invoke(ctx, "/"+target.Service+"/GenericCall", payload, &reply)
	return reply, err
}

func (r *InProcessRegistry) HealthCheck(service string) (bool, error) {
	return r.bus.IsRegistered(service), nil
}
