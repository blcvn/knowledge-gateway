/**
 * Auth Service — VNP Memory Console
 *
 * TASK-UI-002: Rewrite từ mock sang real HTTP calls đến /v1/auth/*.
 * Quản lý tokens trong localStorage: access_token, refresh_token, tenant_id.
 *
 * Backward compatibility: vẫn export interface AuthUser và LoginResponse
 * để Login.tsx + Register.tsx không cần sửa imports.
 */

import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';

const BASE = API_CONFIG.auth;

// ─── Types ────────────────────────────────────────────────────────────────────

export interface AuthUser {
  id:         string;
  name:       string;
  email:      string;
  role:       string;
  tenant_id:  string;
  avatar_url?: string;
  /** @deprecated use avatar_url */
  avatar?:    string;
}

export interface LoginResponse {
  access_token:  string;
  refresh_token: string;
  expires_in:    number;
  token_type:    'Bearer';
  user:          AuthUser;
}

// ─── Storage helpers ─────────────────────────────────────────────────────────

function persistTokens(response: LoginResponse): void {
  localStorage.setItem('access_token',  response.access_token);
  localStorage.setItem('refresh_token', response.refresh_token);
  localStorage.setItem('tenant_id',     response.user.tenant_id);
}

function clearTokens(): void {
  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');
  localStorage.removeItem('tenant_id');
}

// ─── Auth Service ─────────────────────────────────────────────────────────────

export const authService = {
  /**
   * POST /v1/auth/login
   * Returns { access_token, refresh_token, user } and persists tokens.
   * Also returns legacy `token` field for backward compatibility with Login.tsx.
   */
  async login(
    email: string,
    password: string,
  ): Promise<LoginResponse & { token: string; user: AuthUser }> {
    const response = await apiClient.post<LoginResponse>(`${BASE}/login`, {
      email,
      password,
    });
    persistTokens(response);
    return {
      ...response,
      token: response.access_token, // backward compat for setAuth(token, user)
    };
  },

  /**
   * POST /v1/auth/logout
   * Sends refresh_token to invalidate server-side, then clears localStorage.
   */
  async logout(): Promise<void> {
    const refreshToken = localStorage.getItem('refresh_token');
    try {
      await apiClient.post<void>(`${BASE}/logout`, {
        refresh_token: refreshToken,
      });
    } finally {
      clearTokens();
    }
  },

  /**
   * GET /v1/auth/me
   * Returns current authenticated user from JWT context.
   */
  async getMe(): Promise<AuthUser> {
    return apiClient.get<AuthUser>(`${BASE}/me`);
  },

  /**
   * POST /v1/auth/refresh
   * Uses stored refresh_token to obtain a new access_token.
   * Called automatically by api-client when 401 is received.
   */
  async refreshToken(): Promise<{ access_token: string; expires_in: number }> {
    const refreshToken = localStorage.getItem('refresh_token');
    if (!refreshToken) {
      throw new Error('No refresh token available');
    }
    const response = await apiClient.post<{ access_token: string; expires_in: number }>(
      `${BASE}/refresh`,
      { refresh_token: refreshToken },
    );
    localStorage.setItem('access_token', response.access_token);
    return response;
  },

  // ── Helpers ────────────────────────────────────────────────────────

  isAuthenticated(): boolean {
    return !!localStorage.getItem('access_token');
  },

  getAccessToken(): string | null {
    return localStorage.getItem('access_token');
  },

  getTenantId(): string | null {
    return localStorage.getItem('tenant_id');
  },

  // ── Stubs for SSO (no backend yet — kept for UI compatibility) ─────

  /**
   * @deprecated Google SSO not implemented yet — shows toast in UI.
   */
  async loginWithGoogle(): Promise<LoginResponse & { token: string; user: AuthUser }> {
    throw new Error('Google SSO is not configured. Please use email/password login.');
  },

  /**
   * @deprecated Self-registration not implemented yet.
   */
  async register(
    _name: string,
    _email: string,
    _password: string,
  ): Promise<LoginResponse & { token: string; user: AuthUser }> {
    throw new Error('Self-registration is not supported. Contact your administrator.');
  },
};
