# TASK-UI-002 — Rewrite `services/auth.ts`: Real API calls

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-002 |
| **Layer** | Frontend — TypeScript |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-002 §3.1](../solutions/SOL-002-Auth-Solution.md) |
| **Priority** | 🔴 P0 — Critical (Blocker) |
| **Depends On** | TASK-UI-001 |
| **Estimated** | 1h |

---

## Context

`ui/src/services/auth.ts` hiện dùng fake delay + hardcoded tokens. Tất cả API calls khác sẽ nhận 401 nếu token là fake. Đây là blocker cho toàn bộ migration.

---

## Goal

- Thay thế toàn bộ mock logic bằng real HTTP calls đến backend
- Implement login/logout/refresh/getMe
- Quản lý tokens trong localStorage với đúng key names (`access_token`, `refresh_token`, `tenant_id`)

---

## Target Files

| Action | File Path |
|---|---|
| MODIFY | `ui/src/services/auth.ts` |

---

## Implementation

### File: `ui/src/services/auth.ts`

```typescript
import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';

const BASE = API_CONFIG.auth;

export interface AuthUser {
  id: string;
  name: string;
  email: string;
  role: string;
  tenant_id: string;
  avatar_url?: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type: 'Bearer';
  user: AuthUser;
}

export const authService = {
  async login(email: string, password: string): Promise<LoginResponse> {
    const response = await apiClient.post<LoginResponse>(`${BASE}/login`, {
      email,
      password,
    });
    // Persist tokens
    localStorage.setItem('access_token', response.access_token);
    localStorage.setItem('refresh_token', response.refresh_token);
    localStorage.setItem('tenant_id', response.user.tenant_id);
    return response;
  },

  async logout(): Promise<void> {
    const refreshToken = localStorage.getItem('refresh_token');
    try {
      await apiClient.post<void>(`${BASE}/logout`, { refresh_token: refreshToken });
    } finally {
      // Always clear local storage even if API call fails
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      localStorage.removeItem('tenant_id');
    }
  },

  async getMe(): Promise<AuthUser> {
    return apiClient.get<AuthUser>(`${BASE}/me`);
  },

  async refreshToken(): Promise<{ access_token: string; expires_in: number }> {
    const refreshToken = localStorage.getItem('refresh_token');
    if (!refreshToken) {
      throw new Error('No refresh token available');
    }
    const response = await apiClient.post<{ access_token: string; expires_in: number }>(
      `${BASE}/refresh`,
      { refresh_token: refreshToken }
    );
    localStorage.setItem('access_token', response.access_token);
    return response;
  },

  isAuthenticated(): boolean {
    return !!localStorage.getItem('access_token');
  },

  getAccessToken(): string | null {
    return localStorage.getItem('access_token');
  },

  getTenantId(): string | null {
    return localStorage.getItem('tenant_id');
  },
};
```

---

## Verification

```bash
cd ui
npx tsc --noEmit
```

Manual test: Đăng nhập với credentials thực → localStorage có `access_token`, `refresh_token`, `tenant_id`.
