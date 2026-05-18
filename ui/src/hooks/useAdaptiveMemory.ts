import { useQuery } from '@tanstack/react-query';
import { adaptiveService } from '../services/adaptive.service';
import { adaptiveMock } from '../mock/adaptive.mock';
import { API_CONFIG } from '../config/api.config';
import type { AdaptiveAnalytics } from '../types/adaptive';

const useMock = API_CONFIG.useMockData;

const defaultAnalyticsMock: AdaptiveAnalytics = {
  creation_rate: 12,
  deletion_rate: 3,
  contradiction_count: 5,
  connector_sync_count: 48,
  storage_usage_bytes: 52_428_800, // 50MB
};

export function useAdaptiveMemories() {
  return useQuery({
    queryKey: ['adaptive', 'memories'],
    queryFn: useMock
      ? () => Promise.resolve(adaptiveMock.memories)
      : () => adaptiveService.getMemories(),
  });
}

export function useMemoryVersions(id: string) {
  return useQuery({
    queryKey: ['adaptive', 'versions', id],
    queryFn: useMock
      ? () => Promise.resolve(adaptiveMock.versions)
      : () => adaptiveService.getMemoryVersions(id),
    enabled: !!id,
  });
}

export function useConnectors() {
  return useQuery({
    queryKey: ['adaptive', 'connectors'],
    queryFn: useMock
      ? () => Promise.resolve(adaptiveMock.connectors)
      : () => adaptiveService.getConnectors(),
  });
}

/** Fixed M1: now returns properly typed mock instead of empty object */
export function useAdaptiveAnalytics() {
  return useQuery({
    queryKey: ['adaptive', 'analytics'],
    queryFn: useMock
      ? () => Promise.resolve(defaultAnalyticsMock)
      : () => adaptiveService.getAnalytics(),
  });
}
