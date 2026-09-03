// Package observability implements ObservabilityService.
//
// Absorbed from: vnp-observability, sm-engine
// (MERGE-P3-T2)
package observability

import (
	"context"
	"fmt"
	"time"

	domobs "vnp-memory/services/obs-service/internal/domain/observability"
	"vnp-memory/services/obs-service/internal/usecase/port"
)

// ObservabilityService aggregates metrics, traces, errors, and costs.
type ObservabilityService struct {
	scraper port.MetricsScraper
	traces  port.TraceClient
	errors  port.ErrorRepository
	costs   port.CostRepository
	metrics port.MetricsRepository
}

// NewObservabilityService creates an ObservabilityService.
func NewObservabilityService(
	scraper port.MetricsScraper,
	traces port.TraceClient,
	errors port.ErrorRepository,
	costs port.CostRepository,
	metrics port.MetricsRepository,
) *ObservabilityService {
	return &ObservabilityService{
		scraper: scraper,
		traces:  traces,
		errors:  errors,
		costs:   costs,
		metrics: metrics,
	}
}

// Metrics returns an aggregated MetricSummary.
// Falls back to PostgreSQL if Prometheus is unavailable.
func (s *ObservabilityService) Metrics(ctx context.Context) (*domobs.MetricSummary, error) {
	if s.scraper != nil {
		points, err := s.scraper.ScrapeAll(ctx)
		if err == nil && len(points) > 0 {
			return aggregateMetrics(points), nil
		}
	}
	// Fallback: aggregate from PostgreSQL
	if s.metrics != nil {
		return s.metrics.GetSummary(ctx)
	}
	// Static fallback if no DB either
	return &domobs.MetricSummary{
		TotalRequests: 0,
		ErrorRate:     0,
		Timestamp:     time.Now(),
		Services:      []domobs.ServiceMetric{},
	}, nil
}

// ListTraces returns traces filtered by TraceFilter.
func (s *ObservabilityService) ListTraces(ctx context.Context, filter domobs.TraceFilter) ([]*domobs.Trace, int, error) {
	if s.traces == nil {
		return []*domobs.Trace{}, 0, nil
	}
	return s.traces.ListTraces(ctx, filter)
}

// GetTrace retrieves a single trace by ID.
func (s *ObservabilityService) GetTrace(ctx context.Context, traceID string) (*domobs.Trace, error) {
	if s.traces == nil {
		return nil, fmt.Errorf("trace backend: not configured")
	}
	return s.traces.GetTrace(ctx, traceID)
}

// Errors returns grouped error entries.
func (s *ObservabilityService) Errors(ctx context.Context, filter domobs.ErrorFilter) ([]*domobs.ErrorEntry, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	return s.errors.List(ctx, filter)
}

// Costs returns cost entries for a given period.
func (s *ObservabilityService) Costs(ctx context.Context, period string) ([]*domobs.CostEntry, error) {
	if period == "" {
		period = "day"
	}
	return s.costs.GetByPeriod(ctx, period)
}

// EngineStatus returns status for all memory engines (sm-engine absorption).
func (s *ObservabilityService) EngineStatus(ctx context.Context) (map[string]any, error) {
	// Aggregate engine health from metrics
	summary, err := s.Metrics(ctx)
	if err != nil {
		return nil, err
	}
	engines := map[string]any{}
	for _, svc := range summary.Services {
		engines[svc.Name] = map[string]any{
			"healthy":       svc.Healthy,
			"request_count": svc.RequestCount,
			"error_count":   svc.ErrorCount,
			"avg_latency_ms": svc.AvgLatencyMs,
		}
	}
	return map[string]any{
		"engines":   engines,
		"timestamp": summary.Timestamp,
	}, nil
}

// aggregateMetrics aggregates raw metric points into a MetricSummary.
func aggregateMetrics(points []*domobs.MetricPoint) *domobs.MetricSummary {
	summary := &domobs.MetricSummary{Timestamp: time.Now()}
	serviceMap := make(map[string]*domobs.ServiceMetric)

	for _, p := range points {
		svc, ok := serviceMap[p.Labels["service"]]
		if !ok {
			svc = &domobs.ServiceMetric{Name: p.Labels["service"], Healthy: true}
			serviceMap[p.Labels["service"]] = svc
		}
		switch p.Name {
		case "http_requests_total":
			svc.RequestCount += int64(p.Value)
			summary.TotalRequests += int64(p.Value)
		case "http_errors_total":
			svc.ErrorCount += int64(p.Value)
		case "http_request_duration_ms_p50":
			summary.P50LatencyMs = p.Value
		case "http_request_duration_ms_p95":
			summary.P95LatencyMs = p.Value
		case "http_request_duration_ms_p99":
			summary.P99LatencyMs = p.Value
		}
	}

	for _, svc := range serviceMap {
		if svc.RequestCount > 0 {
			summary.ErrorRate = float64(svc.ErrorCount) / float64(svc.RequestCount)
		}
		summary.Services = append(summary.Services, *svc)
	}
	return summary
}
