/**
 * Dashboard Service — calls real /v1/console/dashboard/* endpoints
 * TASK-API-004
 */

import { apiClient }   from '../lib/api-client';
import { API_CONFIG }  from '../config/api.config';
import type { KPIData, EngineHealth, ThroughputData, HeatmapData } from '../types/dashboard';

const BASE = API_CONFIG.console.dashboard;

export const dashboardService = {
  /** GET /v1/console/dashboard/metrics */
  getMetrics: (): Promise<KPIData> =>
    apiClient.get<KPIData>(`${BASE}/metrics`),

  /** GET /v1/console/dashboard/health */
  getHealth: (): Promise<EngineHealth[]> =>
    apiClient.get<EngineHealth[]>(`${BASE}/health`),

  /** GET /v1/console/dashboard/throughput?window=<w> */
  getThroughput: (window = '1h'): Promise<ThroughputData> =>
    apiClient.get<ThroughputData>(`${BASE}/throughput?window=${window}`),

  /** GET /v1/console/dashboard/heatmap */
  getHeatmap: (): Promise<HeatmapData> =>
    apiClient.get<HeatmapData>(`${BASE}/heatmap`),
};
