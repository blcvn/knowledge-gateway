import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type { KPIData, EngineHealth, ThroughputData } from '../types/dashboard';

const BASE = API_CONFIG.console.dashboard;

export const dashboardService = {
  /** GET /v1/console/dashboard/health */
  getHealth: () =>
    apiClient.get<EngineHealth[]>(`${BASE}/health`),

  /** GET /v1/console/dashboard/metrics */
  getMetrics: () =>
    apiClient.get<KPIData>(`${BASE}/metrics`),

  /** GET /v1/console/dashboard/throughput */
  getThroughput: () =>
    apiClient.get<ThroughputData>(`${BASE}/throughput`),

  /** GET /v1/console/dashboard/heatmap */
  getHeatmap: () =>
    apiClient.get<Record<string, unknown>>(`${BASE}/heatmap`),
};
