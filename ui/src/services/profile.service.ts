/**
 * Profile Service — calls real /v1/console/profiles/* endpoints
 * TASK-API-008: typed ProfileConfig (partial update support)
 */

import { apiClient }  from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type {
  UserProfile, ProfileConfig, BufferZone, UserEvent, ContextAssembly,
} from '../types/profile';

const BASE = API_CONFIG.console.profiles;

export const profileService = {
  /** GET /v1/console/profiles */
  listProfiles: (): Promise<UserProfile[]> =>
    apiClient.get<UserProfile[]>(`${BASE}`),

  /** GET /v1/console/profiles/{user_id} */
  getProfile: (userId: string): Promise<UserProfile> =>
    apiClient.get<UserProfile>(`${BASE}/${userId}`),

  /** GET /v1/console/profiles/{user_id}/buffers */
  getBuffers: (userId: string): Promise<BufferZone> =>
    apiClient.get<BufferZone>(`${BASE}/${userId}/buffers`),

  /** GET /v1/console/profiles/{user_id}/context */
  getContext: (userId: string): Promise<ContextAssembly> =>
    apiClient.get<ContextAssembly>(`${BASE}/${userId}/context`),

  /** GET /v1/console/profiles/{user_id}/events */
  getEvents: (userId: string): Promise<UserEvent[]> =>
    apiClient.get<UserEvent[]>(`${BASE}/${userId}/events`),

  /** GET /v1/console/profiles/config */
  getProfileConfig: (): Promise<ProfileConfig> =>
    apiClient.get<ProfileConfig>(`${BASE}/config`),

  /** PUT /v1/console/profiles/config */
  updateProfileConfig: (config: Partial<ProfileConfig>): Promise<ProfileConfig> =>
    apiClient.put<ProfileConfig>(`${BASE}/config`, config),
};
