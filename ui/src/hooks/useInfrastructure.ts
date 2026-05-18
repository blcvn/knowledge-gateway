import { useQuery } from '@tanstack/react-query';
import { infrastructureService } from '../services/infrastructure.service';
import { infrastructureMock } from '../mock/infrastructure.mock';
import { API_CONFIG } from '../config/api.config';

const useMock = API_CONFIG.useMockData;

export function useServiceHealth() {
  return useQuery({
    queryKey: ['infrastructure', 'services'],
    queryFn: useMock
      ? () => Promise.resolve(infrastructureMock.services)
      : () => infrastructureService.getServiceHealth(),
  });
}

export function useDatabaseHealth() {
  return useQuery({
    queryKey: ['infrastructure', 'databases'],
    queryFn: useMock
      ? () => Promise.resolve(infrastructureMock.databases)
      : () => infrastructureService.getDatabaseHealth(),
  });
}

export function useResourceMetrics() {
  return useQuery({
    queryKey: ['infrastructure', 'resources'],
    queryFn: useMock
      ? () => Promise.resolve(infrastructureMock.metrics)
      : () => infrastructureService.getResourceMetrics(),
  });
}
