// Package infra implements InfraService.
//
// Absorbed from: vnp-infra
// (MERGE-P3-T2)
package infra

import (
	"context"
	"fmt"
	"runtime"
	"time"

	dominfra "vnp-memory/services/obs-service/internal/domain/infra"
)

// InfraService manages service topology and infrastructure status.
type InfraService struct {
	docker   DockerClientInterface
	registry ServiceRegistryInterface
	db       DBInspectorInterface
}

// NewInfraService creates an InfraService.
func NewInfraService(docker DockerClientInterface, registry ServiceRegistryInterface, db DBInspectorInterface) *InfraService {
	return &InfraService{docker: docker, registry: registry, db: db}
}

// Topology returns the service dependency graph.
func (s *InfraService) Topology(ctx context.Context) (*dominfra.TopologyGraph, error) {
	services, err := s.ListServices(ctx)
	if err != nil {
		// Fallback: static service list
		services = staticServiceList()
	}
	return &dominfra.TopologyGraph{
		Services:  services,
		Edges:     dominfra.StaticEdges,
		UpdatedAt: time.Now(),
	}, nil
}

// ListServices returns running services from Docker or static config.
func (s *InfraService) ListServices(ctx context.Context) ([]*dominfra.ServiceInfo, error) {
	if s.docker != nil {
		containers, err := s.docker.ListContainers(ctx)
		if err == nil {
			return containers, nil
		}
	}
	// Fallback: gateway registry
	if s.registry != nil {
		names := s.registry.ListAll()
		services := make([]*dominfra.ServiceInfo, 0, len(names))
		for _, name := range names {
			healthy, _ := s.registry.HealthCheck(ctx, name)
			status := "healthy"
			if !healthy {
				status = "down"
			}
			services = append(services, &dominfra.ServiceInfo{
				Name:        name,
				Status:      status,
				LastCheckAt: time.Now(),
			})
		}
		return services, nil
	}
	// Final fallback: static list
	return staticServiceList(), nil
}

// GetService returns details for a single service.
func (s *InfraService) GetService(ctx context.Context, name string) (*dominfra.ServiceInfo, error) {
	services, err := s.ListServices(ctx)
	if err != nil {
		return nil, err
	}
	for _, svc := range services {
		if svc.Name == name {
			return svc, nil
		}
	}
	return nil, fmt.Errorf("service not found: %s", name)
}

// Databases returns database status via DBInspector.
func (s *InfraService) Databases(ctx context.Context) ([]*dominfra.Database, error) {
	if s.db != nil {
		return s.db.InspectAll(ctx)
	}
	// Static fallback
	return []*dominfra.Database{
		{Name: "postgres", Type: "postgres", Status: "unknown"},
		{Name: "redis", Type: "redis", Status: "unknown"},
		{Name: "nats", Type: "nats", Status: "unknown"},
	}, nil
}

// Resources returns current host resource usage.
func (s *InfraService) Resources(ctx context.Context) ([]*dominfra.Resource, error) {
	if s.docker != nil {
		return s.docker.GetResources(ctx)
	}
	// Basic Go runtime stats as fallback
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return []*dominfra.Resource{
		{
			Name:  "memory",
			Type:  "memory",
			Used:  float64(memStats.Alloc) / 1024 / 1024,
			Total: float64(memStats.Sys) / 1024 / 1024,
			Unit:  "MB",
		},
		{
			Name:  "goroutines",
			Type:  "cpu",
			Used:  float64(runtime.NumGoroutine()),
			Total: float64(runtime.NumCPU() * 100),
			Unit:  "count",
		},
	}, nil
}

// Deployments returns a static list of service deployments.
func (s *InfraService) Deployments(_ context.Context) ([]*dominfra.Deployment, error) {
	deps := make([]*dominfra.Deployment, 0, len(dominfra.KnownServices))
	for _, svc := range dominfra.KnownServices {
		deps = append(deps, &dominfra.Deployment{
			Service:   svc,
			Version:   "latest",
			Status:    "running",
			Replicas:  1,
			CreatedAt: time.Now(),
		})
	}
	return deps, nil
}

// staticServiceList returns the hard-coded service list as fallback.
func staticServiceList() []*dominfra.ServiceInfo {
	services := make([]*dominfra.ServiceInfo, 0, len(dominfra.KnownServices))
	for _, name := range dominfra.KnownServices {
		services = append(services, &dominfra.ServiceInfo{
			Name:        name,
			Status:      "unknown",
			LastCheckAt: time.Now(),
		})
	}
	return services
}
