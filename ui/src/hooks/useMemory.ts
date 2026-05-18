import { useQuery } from '@tanstack/react-query';
import { memoryService } from '../services/memory.service';
import { memoryMock } from '../mock/memory.mock';
import { API_CONFIG } from '../config/api.config';

const useMock = API_CONFIG.useMockData;

export function useMemorySearch(query: Record<string, unknown>) {
  return useQuery({
    queryKey: ['memories', 'search', query],
    queryFn: useMock
      ? () => Promise.resolve(memoryMock.searchResult)
      : () => memoryService.search(query),
  });
}

export function useMemoryDetail(id: string) {
  return useQuery({
    queryKey: ['memories', 'detail', id],
    queryFn: useMock
      ? () => Promise.resolve(memoryMock.detail)
      : () => memoryService.getById(id),
    enabled: !!id,
  });
}

export function useMemoryNeighbors(id: string) {
  return useQuery({
    queryKey: ['memories', 'neighbors', id],
    queryFn: useMock
      ? () => Promise.resolve(memoryMock.searchResult)
      : () => memoryService.getNeighbors(id),
    enabled: !!id,
  });
}
