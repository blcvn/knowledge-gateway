import { useQuery } from '@tanstack/react-query';
import { dashboardService } from '../services/dashboard.service';
import { dashboardMock } from '../mock/dashboard.mock';
import { API_CONFIG } from '../config/api.config';

const useMock = API_CONFIG.useMockData;

export function useMetrics() {
  return useQuery({
    queryKey: ['metrics'],
    queryFn: useMock
      ? () => Promise.resolve(dashboardMock.kpis)
      : () => dashboardService.getMetrics(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useEngineHealth() {
  return useQuery({
    queryKey: ['engineHealth'],
    queryFn: useMock
      ? () => Promise.resolve(dashboardMock.engineHealth)
      : () => dashboardService.getHealth(),
    staleTime: 60 * 1000,
  });
}

export function useThroughput() {
  return useQuery({
    queryKey: ['throughput'],
    queryFn: useMock
      ? () => Promise.resolve(dashboardMock.throughput)
      : () => dashboardService.getThroughput(),
    staleTime: 60 * 1000,
  });
}
