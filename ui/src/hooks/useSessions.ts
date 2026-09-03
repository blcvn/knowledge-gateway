/**
 * Session Hooks — real API, no mock
 * TASK-API-005: keepPreviousData for pagination, live poll 10s
 */

import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { sessionService }             from '../services/session.service';
import type { SessionFilters }        from '../types/session';

export const sessionKeys = {
  all:         () => ['sessions'] as const,
  list:        (f: SessionFilters) => [...sessionKeys.all(), 'list', f] as const,
  live:        () => [...sessionKeys.all(), 'live'] as const,
  detail:      (id: string) => [...sessionKeys.all(), id, 'detail'] as const,
  timeline:    (id: string) => [...sessionKeys.all(), id, 'timeline'] as const,
  diff:        (id: string) => [...sessionKeys.all(), id, 'diff'] as const,
  workingMem:  (id: string) => [...sessionKeys.all(), id, 'working-memory'] as const,
  userSummary: (id: string) => [...sessionKeys.all(), id, 'user-summary'] as const,
};

/**
 * Paginated session list — keepPreviousData prevents flash on page change
 * API: GET /v1/console/sessions[?...]
 */
export function useSessionList(filters: SessionFilters = {}) {
  return useQuery({
    queryKey:        sessionKeys.list(filters),
    queryFn:         () => sessionService.getSessions(filters),
    placeholderData: keepPreviousData,
    staleTime:       30_000,
  });
}

/**
 * Live active sessions — poll every 10s
 * API: GET /v1/console/sessions/live
 */
export function useLiveSessions() {
  return useQuery({
    queryKey:                sessionKeys.live(),
    queryFn:                 () => sessionService.getLiveSessions(),
    refetchInterval:         10_000,
    refetchIntervalInBackground: false,
  });
}

/**
 * Session conversation detail
 * API: GET /v1/console/sessions/{id}
 */
export function useSessionDetail(id: string) {
  return useQuery({
    queryKey:  sessionKeys.detail(id),
    queryFn:   () => sessionService.getSessionDetail(id),
    enabled:   !!id,
    staleTime: 60_000,
  });
}

/**
 * Memory operation timeline for a session
 * API: GET /v1/console/sessions/{id}/timeline
 */
export function useSessionTimeline(id: string) {
  return useQuery({
    queryKey: sessionKeys.timeline(id),
    queryFn:  () => sessionService.getTimeline(id),
    enabled:  !!id,
  });
}

/**
 * Memory diff (what changed during session)
 * API: GET /v1/console/sessions/{id}/diff
 */
export function useSessionDiff(id: string) {
  return useQuery({
    queryKey: sessionKeys.diff(id),
    queryFn:  () => sessionService.getDiff(id),
    enabled:  !!id,
  });
}

/**
 * Working memory — poll 5s when session is active
 * API: GET /v1/console/sessions/{id}/working-memory
 */
export function useWorkingMemory(id: string, isActive = false) {
  return useQuery({
    queryKey:        sessionKeys.workingMem(id),
    queryFn:         () => sessionService.getWorkingMemory(id),
    enabled:         !!id,
    refetchInterval: isActive ? 5_000 : false,
  });
}

/**
 * User context summary for session
 * API: GET /v1/console/sessions/{id}/user-summary
 */
export function useUserSummary(id: string) {
  return useQuery({
    queryKey: sessionKeys.userSummary(id),
    queryFn:  () => sessionService.getUserSummary(id),
    enabled:  !!id,
  });
}
