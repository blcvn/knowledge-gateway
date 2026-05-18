import { useQuery } from '@tanstack/react-query';
import { sessionService } from '../services/session.service';
import { sessionMock } from '../mock/session.mock';
import { API_CONFIG } from '../config/api.config';

const useMock = API_CONFIG.useMockData;

export function useSessionList() {
  return useQuery({
    queryKey: ['sessions'],
    queryFn: useMock
      ? () => Promise.resolve(sessionMock.sessions)
      : () => sessionService.getSessions(),
  });
}

export function useSessionDetail(id: string) {
  return useQuery({
    queryKey: ['sessions', id],
    queryFn: useMock
      ? () => Promise.resolve(sessionMock.conversation)
      : () => sessionService.getSessionDetail(id),
    enabled: !!id,
  });
}
