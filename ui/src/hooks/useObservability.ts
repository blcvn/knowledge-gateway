import { useQuery } from '@tanstack/react-query';
import { observabilityService } from '../services/observability.service';
import { observabilityMock } from '../mock/observability.mock';
import { API_CONFIG } from '../config/api.config';

const useMock = API_CONFIG.useMockData;

export function useMetricsDashboard() {
  return useQuery({
    queryKey: ['observability', 'metrics'],
    queryFn: useMock
      ? () => Promise.resolve(observabilityMock.metrics)
      : () => observabilityService.getMetrics(),
  });
}

export function useTraces(filters: Record<string, string>) {
  return useQuery({
    queryKey: ['observability', 'traces', filters],
    queryFn: useMock
      ? () => Promise.resolve(observabilityMock.traces)
      : () => observabilityService.getTraces(filters),
  });
}

export function useErrors(filters: Record<string, string>) {
  return useQuery({
    queryKey: ['observability', 'errors', filters],
    queryFn: useMock
      ? () => Promise.resolve(observabilityMock.errors)
      : () => observabilityService.getErrors(filters),
  });
}
