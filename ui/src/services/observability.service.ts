import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type { MetricPoint, TraceSpan, ErrorEntry } from '../types/observability';

const BASE = API_CONFIG.console.observability;

export const observabilityService = {
  /** GET /v1/console/observability/metrics */
  getMetrics: () =>
    apiClient.get<MetricPoint[]>(`${BASE}/metrics`),

  /** GET /v1/console/observability/traces */
  getTraces: (filters: Record<string, string>) => {
    const qs = new URLSearchParams(filters).toString();
    return apiClient.get<TraceSpan[]>(`${BASE}/traces?${qs}`);
  },

  /** GET /v1/console/observability/traces/{id} */
  getTrace: (id: string) =>
    apiClient.get<TraceSpan>(`${BASE}/traces/${id}`),

  /** GET /v1/console/observability/errors */
  getErrors: (filters: Record<string, string>) => {
    const qs = new URLSearchParams(filters).toString();
    return apiClient.get<ErrorEntry[]>(`${BASE}/errors?${qs}`);
  },

  /** GET /v1/console/observability/costs */
  getCosts: () =>
    apiClient.get<unknown>(`${BASE}/costs`),
};
