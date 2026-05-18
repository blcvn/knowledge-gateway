import { useQuery } from '@tanstack/react-query';
import { graphService } from '../services/graph.service';
import { graphMock } from '../mock/graph.mock';
import { API_CONFIG } from '../config/api.config';

const useMock = API_CONFIG.useMockData;

export function useSubgraph(params: Record<string, string>) {
  return useQuery({
    queryKey: ['graph', 'subgraph', params],
    queryFn: useMock
      ? () => Promise.resolve(graphMock.subgraph)
      : () => graphService.getSubgraph(params),
  });
}

export function useTimeline(params: Record<string, string>) {
  return useQuery({
    queryKey: ['graph', 'timeline', params],
    queryFn: useMock
      ? () => Promise.resolve(graphMock.timeline)
      : () => graphService.getTimeline(params),
  });
}
