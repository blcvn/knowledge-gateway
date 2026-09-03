# TASK-API-003 — Auth API Client + Hooks + Context

**Task ID:** TASK-API-003
**Status:** ✅ COMPLETED — 2026-06-17
**Sprint:** 2 — P0 Modules
**Solution:** [API-SOL-002](../API-SOL-002-auth.md)
**Depends on:** TASK-API-001, TASK-API-002
**Ước tính:** 2h
**Priority:** P0 — Critical (Blocker cho tất cả CRs khác)

---

## Mục tiêu

Implement auth flow hoàn chỉnh:
1. `auth.client.ts` — gọi API thực `/v1/auth/*`
2. `useAuth.ts` — TanStack Query hooks cho login/logout/me
3. `AuthContext.tsx` — React Context expose user state cho toàn bộ app

---

## Công việc cụ thể

### 1. Tạo `ui/src/api/clients/auth.client.ts`

```typescript
import { httpClient, STORAGE_KEYS } from './http.client';
import type { AuthUser, LoginRequest, LoginResponse, RefreshResponse } from '../../types/auth';

const BASE = '/v1/auth';

export const authClient = {
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const { data } = await httpClient.post<LoginResponse>(`${BASE}/login`, credentials);
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

### 2. Tạo `ui/src/api/hooks/useAuth.ts`

```typescript
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { authClient } from '../clients/auth.client';
import type { LoginRequest } from '../../types/auth';

export function useCurrentUser() {
  return useQuery({
    queryKey:  ['auth', 'me'],
    queryFn:   () => authClient.getMe(),
    enabled:   authClient.isAuthenticated(),
    staleTime: 5 * 60_000,
    retry:     false,
  });
}

export function useLogin() {
  const qc       = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    mutationFn: (credentials: LoginRequest) => authClient.login(credentials),
    onSuccess: (data) => {
      qc.setQueryData(['auth', 'me'], data.user);
      navigate('/');
    },
  });
}

export function useLogout() {
  const qc       = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    mutationFn: () => authClient.logout(),
    onSettled: () => {
      qc.clear();
      navigate('/login');
    },
  });
}
```

### 3. Tạo `ui/src/contexts/AuthContext.tsx`

```typescript
import { createContext, useContext, type ReactNode } from 'react';
import { useCurrentUser } from '../api/hooks/useAuth';
import type { AuthUser } from '../types/auth';

interface AuthContextValue {
  user:            AuthUser | undefined;
  isLoading:       boolean;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
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

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used inside <AuthProvider>');
  return ctx;
}
```

### 4. Cập nhật `ui/src/main.tsx` (hoặc `App.tsx`)

Bọc app trong `QueryClientProvider` và `AuthProvider`:

```typescript
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from './api/queryClient';
import { AuthProvider } from './contexts/AuthContext';

// Trong render:
<QueryClientProvider client={queryClient}>
  <AuthProvider>
    <RouterProvider router={router} />
  </AuthProvider>
</QueryClientProvider>
```

### 5. Xóa mock trong auth

Tìm và xóa toàn bộ references đến mock auth trong `ui/src/services/auth.ts`:

```bash
# Tìm file auth service hiện tại
find ui/src -name "auth.ts" -o -name "auth.service.ts"

# Thay thế nội dung bằng re-export từ client mới
# hoặc xóa nếu không còn được dùng
```

---

## Files tạo ra / chỉnh sửa

```
ui/src/
├── api/
│   ├── clients/
│   │   └── auth.client.ts     ← NEW
│   └── hooks/
│       └── useAuth.ts         ← NEW
├── contexts/
│   └── AuthContext.tsx        ← NEW
└── main.tsx                   ← MODIFY (add providers)
```

---

## Acceptance Criteria

- [x] `POST /v1/auth/login` với credentials thực → nhận JWT, lưu vào localStorage
- [x] `GET /v1/auth/me` với token → trả về user object đúng type `AuthUser`
- [x] `POST /v1/auth/refresh` → cập nhật `access_token` trong localStorage
- [x] `POST /v1/auth/logout` → xóa hết token khỏi localStorage, redirect login
- [x] `useCurrentUser()` chỉ gọi API khi có access_token
- [x] `useLogin()` sau thành công: cache `['auth','me']`, navigate về `/`
- [x] `useLogout()` clear toàn bộ QueryClient cache
- [x] `useAuth()` throw khi dùng ngoài `AuthProvider`
- [x] `npx tsc --noEmit` không lỗi

---

## Sau khi hoàn thành

```bash
cd ui && npx tsc --noEmit
# Test login flow thủ công trên browser
# → Chuyển sang TASK-API-004 (dashboard)
```
