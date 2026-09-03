/**
 * Observability Hooks — real API, no mock
 * TASK-API-010: poll 30s metrics, trace detail, cost tracking, query key factory
 */

import { useQuery } from '@tanstack/react-query';
import { observabilityService } from '../services/observability.service';
import type { TraceFilters } from '../types/observability';

// ─── Query Key Factory ────────────────────────────────────────────────────────

const keys = {
  metrics: () => ['observability', 'metrics'] as const,
  traces:  (f: TraceFilters) => ['observability', 'traces', f] as const,
  trace:   (id: string) => ['observability', 'traces', id] as const,
  errors:  (service?: string) => ['observability', 'errors', service ?? 'all'] as const,
  costs:   () => ['observability', 'costs'] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────────

/**
 * Time-series metrics — poll 30s
 * API: GET /v1/console/observability/metrics
 * Returns: { latency[], error_rate[], throughput[] }
 */
export function useMetricsDashboard() {
  return useQuery({
    queryKey:        keys.metrics(),
    queryFn:         () => observabilityService.getMetrics(),
    refetchInterval: 30_000,
  });
}

/**
 * Trace list with filters
 * API: GET /v1/console/observability/traces[?...filters]
 */
export function useTraces(filters: TraceFilters = {}) {
  return useQuery({
    queryKey: keys.traces(filters),
    queryFn:  () => observabilityService.getTraces(filters),
  });
}

/**
 * Single trace detail — cached 5min
 * API: GET /v1/console/observability/traces/{id}
 */
export function useTraceDetail(id: string) {
  return useQuery({
    queryKey:  keys.trace(id),
    queryFn:   () => observabilityService.getTrace(id),
    enabled:   !!id,
    staleTime: 5 * 60_000,
  });
}

/**
 * Error list filtered by service
 * API: GET /v1/console/observability/errors[?service=<s>]
 */
export function useErrors(filters: { service?: string } = {}) {
  return useQuery({
    queryKey: keys.errors(filters.service),
    queryFn:  () => observabilityService.getErrors(filters),
  });
}

/**
 * LLM cost tracking
 * API: GET /v1/console/observability/costs
 */
export function useCosts() {
  return useQuery({
    queryKey:  keys.costs(),
    queryFn:   () => observabilityService.getCosts(),
    staleTime: 5 * 60_000,
  });
}
