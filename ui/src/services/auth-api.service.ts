/**
 * Auth API Client — calls real /v1/auth/* endpoints
 * Replaces the mock authService in services/auth.ts
 */

import { apiClient, STORAGE_KEYS } from '../lib/api-client';
import type { AuthUser, LoginRequest, LoginResponse, RefreshResponse } from '../types/auth';

const BASE = '/v1/auth';

export const authApiClient = {
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const data = await apiClient.post<LoginResponse>(`${BASE}/login`, credentials);
    // Persist tokens immediately
    localStorage.setItem(STORAGE_KEYS.ACCESS_TOKEN,  data.access_token);
    localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, data.refresh_token);
    localStorage.setItem(STORAGE_KEYS.TENANT_ID,     data.user.tenant_id);
    return data;
  },

  async logout(): Promise<void> {
    const refreshToken = localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN);
    try {
      await apiClient.post<void>(`${BASE}/logout`, { refresh_token: refreshToken });
    } finally {
      // Always clear tokens even if API call fails
      localStorage.removeItem(STORAGE_KEYS.ACCESS_TOKEN);
      localStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
      localStorage.removeItem(STORAGE_KEYS.TENANT_ID);
    }
  },

  async getMe(): Promise<AuthUser> {
    return apiClient.get<AuthUser>(`${BASE}/me`);
  },

  async refresh(refreshToken: string): Promise<RefreshResponse> {
    return apiClient.post<RefreshResponse>(`${BASE}/refresh`, {
      refresh_token: refreshToken,
    });
  },

  isAuthenticated(): boolean {
    return !!localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN);
  },

  getAccessToken(): string | null {
    return localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN);
  },

  getTenantId(): string | null {
    return localStorage.getItem(STORAGE_KEYS.TENANT_ID);
  },
};
