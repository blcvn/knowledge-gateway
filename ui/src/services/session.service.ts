import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type { Session, Conversation, WorkingMemory } from '../types/session';

const BASE = API_CONFIG.console.sessions;

export const sessionService = {
  /** GET /v1/console/sessions */
  getSessions: () =>
    apiClient.get<Session[]>(`${BASE}`),

  /** GET /v1/console/sessions/live */
  getLiveSessions: () =>
    apiClient.get<Session[]>(`${BASE}/live`),

  /** GET /v1/console/sessions/{id} */
  getSessionDetail: (id: string) =>
    apiClient.get<Conversation>(`${BASE}/${id}`),

  /** GET /v1/console/sessions/{id}/timeline */
  getTimeline: (id: string) =>
    apiClient.get<unknown[]>(`${BASE}/${id}/timeline`),

  /** GET /v1/console/sessions/{id}/diff */
  getDiff: (id: string) =>
    apiClient.get<unknown>(`${BASE}/${id}/diff`),

  /** GET /v1/console/sessions/{id}/working-memory */
  getWorkingMemory: (id: string) =>
    apiClient.get<WorkingMemory>(`${BASE}/${id}/working-memory`),

  /** GET /v1/console/sessions/{id}/user-summary */
  getUserSummary: (id: string) =>
    apiClient.get<unknown>(`${BASE}/${id}/user-summary`),
};
