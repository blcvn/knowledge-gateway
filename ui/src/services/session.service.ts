/**
 * Session Service — calls real /v1/console/sessions/* endpoints
 * TASK-API-005
 */

import { apiClient }  from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type {
  Session, Conversation, WorkingMemory, UserSummary,
  SessionFilters, SessionTimeline, SessionDiff,
} from '../types/session';
import type { PaginatedResponse } from '../types/api';

const BASE = API_CONFIG.console.sessions;

function buildQuery(filters: SessionFilters): string {
  const params = new URLSearchParams();
  if (filters.status)    params.set('status',    filters.status);
  if (filters.user_id)   params.set('user_id',   filters.user_id);
  if (filters.agent_id)  params.set('agent_id',  filters.agent_id);
  if (filters.search)    params.set('search',    filters.search);
  if (filters.sort)      params.set('sort',      filters.sort);
  if (filters.page)      params.set('page',      String(filters.page));
  if (filters.page_size) params.set('page_size', String(filters.page_size));
  const qs = params.toString();
  return qs ? `?${qs}` : '';
}

export const sessionService = {
  /** GET /v1/console/sessions[?...filters] */
  getSessions: (filters: SessionFilters = {}): Promise<PaginatedResponse<Session>> =>
    apiClient.get<PaginatedResponse<Session>>(`${BASE}${buildQuery(filters)}`),

  /** GET /v1/console/sessions/live */
  getLiveSessions: (): Promise<Session[]> =>
    apiClient.get<Session[]>(`${BASE}/live`),

  /** GET /v1/console/sessions/{id} */
  getSessionDetail: (id: string): Promise<Conversation> =>
    apiClient.get<Conversation>(`${BASE}/${id}`),

  /** GET /v1/console/sessions/{id}/timeline */
  getTimeline: (id: string): Promise<SessionTimeline[]> =>
    apiClient.get<SessionTimeline[]>(`${BASE}/${id}/timeline`),

  /** GET /v1/console/sessions/{id}/diff */
  getDiff: (id: string): Promise<SessionDiff> =>
    apiClient.get<SessionDiff>(`${BASE}/${id}/diff`),

  /** GET /v1/console/sessions/{id}/working-memory */
  getWorkingMemory: (id: string): Promise<WorkingMemory> =>
    apiClient.get<WorkingMemory>(`${BASE}/${id}/working-memory`),

  /** GET /v1/console/sessions/{id}/user-summary */
  getUserSummary: (id: string): Promise<UserSummary> =>
    apiClient.get<UserSummary>(`${BASE}/${id}/user-summary`),
};
