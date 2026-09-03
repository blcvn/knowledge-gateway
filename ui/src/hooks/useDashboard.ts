/**
 * Dashboard Hooks — uses real API via dashboardService
 * TASK-API-004: Remove mock dependency, add polling intervals
 */

import { useQuery }         from '@tanstack/react-query';
import { dashboardService } from '../services/dashboard.service';
import type { ThroughputData } from '../types/dashboard';

export type ThroughputWindow = '5m' | '15m' | '1h' | '6h' | '24h';

export const dashboardKeys = {
  all:        () => ['dashboard'] as const,
  metrics:    () => [...dashboardKeys.all(), 'metrics'] as const,
  health:     () => [...dashboardKeys.all(), 'health'] as const,
  throughput: (w: ThroughputWindow) => [...dashboardKeys.all(), 'throughput', w] as const,
  heatmap:    () => [...dashboardKeys.all(), 'heatmap'] as const,
};

/**
 * KPI metrics — poll 60s
 * API: GET /v1/console/dashboard/metrics
 */
export function useMetrics() {
  return useQuery({
    queryKey:                dashboardKeys.metrics(),
    queryFn:                 () => dashboardService.getMetrics(),
    staleTime:               30_000,
    refetchInterval:         60_000,
    refetchIntervalInBackground: false,
  });
}

/**
 * 7-engine health checks — poll 30s
 * API: GET /v1/console/dashboard/health
 */
export function useEngineHealth() {
  return useQuery({
    queryKey:                dashboardKeys.health(),
    queryFn:                 () => dashboardService.getHealth(),
    staleTime:               15_000,
    refetchInterval:         30_000,
    refetchIntervalInBackground: false,
  });
}

/**
 * Prometheus throughput metrics — poll 30s
 * API: GET /v1/console/dashboard/throughput?window=<w>
 */
export function useThroughput(window: ThroughputWindow = '1h') {
  return useQuery({
    queryKey:        dashboardKeys.throughput(window),
    queryFn:         () => dashboardService.getThroughput(window),
    staleTime:       15_000,
    refetchInterval: 30_000,
  });
}

/**
 * Activity heatmap — staleTime 5min (slow-changing)
 * API: GET /v1/console/dashboard/heatmap
 */
export function useDashboardHeatmap() {
  return useQuery({
    queryKey:  dashboardKeys.heatmap(),
    queryFn:   () => dashboardService.getHeatmap(),
    staleTime: 5 * 60_000,
  });
}
