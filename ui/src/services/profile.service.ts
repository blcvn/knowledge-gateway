import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type {
  UserProfile, ProfileConfig, BufferZone,
  UserEvent, ContextAssembly,
} from '../types/profile';

const BASE = API_CONFIG.console.profiles;

export const profileService = {
  /** GET /v1/console/profiles */
  listProfiles: () =>
    apiClient.get<UserProfile[]>(`${BASE}`),

  /** GET /v1/console/profiles/config */
  getProfileConfig: () =>
    apiClient.get<ProfileConfig>(`${BASE}/config`),

  /** PUT /v1/console/profiles/config */
  updateProfileConfig: (config: ProfileConfig) =>
    apiClient.put<ProfileConfig>(`${BASE}/config`, config),

  /** GET /v1/console/profiles/{user_id} */
  getProfile: (userId: string) =>
    apiClient.get<UserProfile>(`${BASE}/${userId}`),

  /** GET /v1/console/profiles/{user_id}/events */
  getEvents: (userId: string) =>
    apiClient.get<UserEvent[]>(`${BASE}/${userId}/events`),

  /** GET /v1/console/profiles/{user_id}/context */
  getContext: (userId: string) =>
    apiClient.get<ContextAssembly>(`${BASE}/${userId}/context`),

  /** GET /v1/console/profiles/{user_id}/buffers */
  getBuffers: (userId: string) =>
    apiClient.get<BufferZone>(`${BASE}/${userId}/buffers`),
};
