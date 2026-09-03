/**
 * Observability Service — calls real /v1/console/observability/* endpoints
 * TASK-API-010: typed MetricsResponse, CostEntry, TraceFilters
 */

import { apiClient }  from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type {
  MetricsResponse, TraceSpan, ErrorEntry, CostEntry, TraceFilters,
} from '../types/observability';

const BASE = API_CONFIG.console.observability;

function buildQuery(filters: Record<string, string | undefined>): string {
  const params = new URLSearchParams();
  Object.entries(filters).forEach(([k, v]) => { if (v) params.set(k, v); });
  const qs = params.toString();
  return qs ? `?${qs}` : '';
}

export const observabilityService = {
  /**
   * GET /v1/console/observability/metrics
   * Returns time-series: { latency[], error_rate[], throughput[] }
   */
  getMetrics: (): Promise<MetricsResponse> =>
    apiClient.get<MetricsResponse>(`${BASE}/metrics`),

  /** GET /v1/console/observability/traces[?...filters] */
  getTraces: (filters: TraceFilters = {}): Promise<TraceSpan[]> =>
    apiClient.get<TraceSpan[]>(`${BASE}/traces${buildQuery(filters as Record<string, string>)}`),

  /** GET /v1/console/observability/traces/{id} */
  getTrace: (id: string): Promise<TraceSpan> =>
    apiClient.get<TraceSpan>(`${BASE}/traces/${id}`),

  /** GET /v1/console/observability/errors[?service=<s>] */
  getErrors: (filters: { service?: string } = {}): Promise<ErrorEntry[]> =>
    apiClient.get<ErrorEntry[]>(`${BASE}/errors${buildQuery(filters as Record<string, string>)}`),

  /** GET /v1/console/observability/costs */
  getCosts: (): Promise<CostEntry[]> =>
    apiClient.get<CostEntry[]>(`${BASE}/costs`),
};
