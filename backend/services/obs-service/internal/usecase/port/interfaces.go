// Package port defines output interfaces for obs-service.
package port

import (
	"context"

	domobs "vnp-memory/services/obs-service/internal/domain/observability"
	dominfra "vnp-memory/services/obs-service/internal/domain/infra"
)

// ── Observability Ports ────────────────────────────────────────────────────

// MetricsScraper scrapes metrics from Prometheus or internal tables.
type MetricsScraper interface {
	ScrapeAll(ctx context.Context) ([]*domobs.MetricPoint, error)
}

// TraceClient fetches traces from Jaeger or local store.
type TraceClient interface {
	ListTraces(ctx context.Context, filter domobs.TraceFilter) ([]*domobs.Trace, int, error)
	GetTrace(ctx context.Context, traceID string) (*domobs.Trace, error)
}

// ErrorRepository persists and queries error entries.
type ErrorRepository interface {
	List(ctx context.Context, filter domobs.ErrorFilter) ([]*domobs.ErrorEntry, int, error)
	Record(ctx context.Context, entry *domobs.ErrorEntry) error
}

// CostRepository persists and queries cost entries.
type CostRepository interface {
	GetByPeriod(ctx context.Context, period string) ([]*domobs.CostEntry, error)
	Record(ctx context.Context, entry *domobs.CostEntry) error
}

// MetricsRepository persists metric points.
type MetricsRepository interface {
	Record(ctx context.Context, points []*domobs.MetricPoint) error
	GetSummary(ctx context.Context) (*domobs.MetricSummary, error)
}

// ── Infra Ports ────────────────────────────────────────────────────────────

// DockerClient introspects containers via Docker socket.
type DockerClient interface {
	ListContainers(ctx context.Context) ([]*dominfra.ServiceInfo, error)
	GetResources(ctx context.Context) ([]*dominfra.Resource, error)
}

// ServiceRegistry lists registered gateway upstreams.
type ServiceRegistry interface {
	ListAll() []string
	HealthCheck(ctx context.Context, service string) (bool, error)
}

// DBInspector inspects database connectivity and size.
type DBInspector interface {
	InspectAll(ctx context.Context) ([]*dominfra.Database, error)
}
