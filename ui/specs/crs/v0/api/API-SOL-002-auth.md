# API-SOL-002 — Auth API Client + Hooks

| Field | Value |
|---|---|
| **Solution ID** | API-SOL-002 |
| **Status** | ✅ IMPLEMENTED — 2026-06-17 |
| **CR** | [CR-001 — Authentication](../../../../specs/crs/v0/ui/CR-001-AUTH.md) |
| **Kiến trúc ref** | `frontend_architecture.md §3.3, §5.3` |
| **Target files** | `ui/src/services/auth.service.ts` · `ui/src/hooks/useAuth.ts` |
| **Implemented files** | `ui/src/services/auth.service.ts` · `ui/src/hooks/useAuth.ts` · `ui/src/contexts/AuthContext.tsx` |

---

## API Endpoints Được Implement

| Method | Endpoint | Mô tả |
|---|---|---|
| `POST` | `/v1/auth/login` | Email/password login |
| `POST` | `/v1/auth/logout` | Invalidate session |
| `POST` | `/v1/auth/refresh` | Refresh access token |
| `GET` | `/v1/auth/me` | Lấy thông tin user hiện tại |

---

## Implementation

### `ui/src/api/clients/auth.client.ts`

```typescript
import { httpClient, STORAGE_KEYS } from './http.client';

// ─── Types ────────────────────────────────────────────────────────────────────

export interface AuthUser {
  id:         string;
  name:       string;
  email:      string;
  role:       string;
  tenant_id:  string;
  avatar_url?: string;
}

export interface LoginRequest {
  email:    string;
  password: string;
}

export interface LoginResponse {
  access_token:  string;
  refresh_token: string;
  expires_in:    number;
  token_type:    'Bearer';
  user:          AuthUser;
}

export interface RefreshResponse {
  access_token: string;
  expires_in:   number;
}

// ─── Client ───────────────────────────────────────────────────────────────────

const BASE = '/v1/auth';

export const authClient = {
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const { data } = await httpClient.post<LoginResponse>(`${BASE}/login`, credentials);

    // Persist tokens — single responsibility in client layer
    localStorage.setItem(STORAGE_KEYS.ACCESS_TOKEN,  data.access_token);
    localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, data.refresh_token);
    localStorage.setItem(STORAGE_KEYS.TENANT_ID,     data.user.tenant_id);

    return data;
  },

  async logout(): Promise<void> {
    const refreshToken = localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN);
    try {
      await httpClient.post(`${BASE}/logout`, { refresh_token: refreshToken });
    } finally {
      // Always clear storage even if API fails
      localStorage.removeItem(STORAGE_KEYS.ACCESS_TOKEN);
      localStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
      localStorage.removeItem(STORAGE_KEYS.TENANT_ID);
    }
  },

  async getMe(): Promise<AuthUser> {
    const { data } = await httpClient.get<AuthUser>(`${BASE}/me`);
    return data;
  },

  async refresh(refreshToken: string): Promise<RefreshResponse> {
    // Note: gọi trực tiếp httpClient để tránh interceptor loop
    const { data } = await httpClient.post<RefreshResponse>(
      `${BASE}/refresh`,
      { refresh_token: refreshToken },
    );
    return data;
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
```

### `ui/src/api/hooks/useAuth.ts`

```typescript
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { authClient, type LoginRequest } from '../clients/auth.client';

// ─── Hooks ────────────────────────────────────────────────────────────────────

/**
 * Query: Lấy thông tin user đang đăng nhập.
 * - Chỉ fetch khi đã có access_token
 * - Dùng để restore session khi reload trang
 */
export function useCurrentUser() {
  return useQuery({
    queryKey: ['auth', 'me'],
    queryFn:  () => authClient.getMe(),
    enabled:  authClient.isAuthenticated(),
    staleTime: 5 * 60_000,   // Cache 5 phút
    retry: false,             // Không retry auth errors
  });
}

/**
 * Mutation: Đăng nhập
 * - Sau khi thành công: lưu tokens, invalidate cache, navigate về dashboard
 */
export function useLogin() {
  const qc       = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    mutationFn: (credentials: LoginRequest) => authClient.login(credentials),
    onSuccess: (data) => {
      // Warm up cache với user data
      qc.setQueryData(['auth', 'me'], data.user);
      navigate('/');
    },
  });
}

/**
 * Mutation: Đăng xuất
 * - Clear toàn bộ TanStack Query cache
 * - Navigate về trang login
 */
export function useLogout() {
  const qc       = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    mutationFn: () => authClient.logout(),
    onSettled: () => {
      // Clear cache bất kể thành công hay thất bại
      qc.clear();
      navigate('/login');
    },
  });
}
```

---

## Types (đồng bộ với backend)

### `ui/src/types/auth.ts`

```typescript
// Re-export từ client để dùng trong components
export type { AuthUser, LoginRequest, LoginResponse } from '../api/clients/auth.client';
```

---

## Context Integration

Auth user được expose qua `AuthContext` để component không cần gọi hook trực tiếp:

```typescript
// ui/src/contexts/AuthContext.tsx
import { createContext, useContext } from 'react';
import { useCurrentUser } from '../api/hooks/useAuth';

interface AuthContextValue {
  user: AuthUser | undefined;
  isLoading: boolean;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const { data: user, isLoading } = useCurrentUser();

  return (
    <AuthContext.Provider value={{
      user,
      isLoading,
      isAuthenticated: !!user,
    }}>
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be inside AuthProvider');
  return ctx;
};
```

---

## Verification

```bash
# Type check
cd ui && npx tsc --noEmit

# Manual test sequence:
# 1. POST /v1/auth/login với credentials thực → 200 + tokens
# 2. GET /v1/auth/me với Bearer token → 200 + user
# 3. POST /v1/auth/refresh → new access_token
# 4. POST /v1/auth/logout → 204, tokens bị xóa khỏi localStorage
```
