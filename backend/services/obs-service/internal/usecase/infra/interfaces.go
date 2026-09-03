// Package infra — interface re-exports for main.go DI wiring.
package infra

import (
	"context"

	dominfra "vnp-memory/services/obs-service/internal/domain/infra"
)

// DockerClientInterface is the interface to Docker container introspection.
type DockerClientInterface interface {
	ListContainers(ctx context.Context) ([]*dominfra.ServiceInfo, error)
	GetResources(ctx context.Context) ([]*dominfra.Resource, error)
}

// ServiceRegistryInterface lists registered gateway upstreams.
type ServiceRegistryInterface interface {
	ListAll() []string
	HealthCheck(ctx context.Context, service string) (bool, error)
}

// DBInspectorInterface inspects database connectivity.
type DBInspectorInterface interface {
	InspectAll(ctx context.Context) ([]*dominfra.Database, error)
}
