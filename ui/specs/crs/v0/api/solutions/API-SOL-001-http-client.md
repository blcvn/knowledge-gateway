# API-SOL-001 — HTTP Client Foundation

| Field | Value |
|---|---|
| **Solution ID** | API-SOL-001 |
| **Status** | ✅ IMPLEMENTED — 2026-06-17 |
| **CR** | CR-000 — Migration Overview |
| **Kiến trúc ref** | `frontend_architecture.md §3.3` Data Access Layer |
| **Target** | `ui/src/lib/api-client.ts` |
| **Implemented files** | `ui/src/lib/api-client.ts` · `ui/src/config/api.config.ts` |

---

## Context

Theo kiến trúc, Data Access Layer dùng Axios (hoặc fetch) với:
- Auto-inject `Authorization: Bearer <token>` header
- Auto-inject `X-Tenant-ID` header từ `TenantContext`
- 401 interceptor → auto-refresh → retry
- Chuẩn hóa error response

---

## Implementation

### `ui/src/api/clients/http.client.ts`

```typescript
import axios, { AxiosError, AxiosInstance, InternalAxiosRequestConfig } from 'axios';

// ─── Types ────────────────────────────────────────────────────────────────────

export interface ApiError {
  message: string;
  code: string;
  status: number;
}

export class AppError extends Error {
  constructor(
    message: string,
    public code: string,
    public status: number,
  ) {
    super(message);
    this.name = 'AppError';
  }
}

// ─── Storage Keys ─────────────────────────────────────────────────────────────

export const STORAGE_KEYS = {
  ACCESS_TOKEN:  'access_token',
  REFRESH_TOKEN: 'refresh_token',
  TENANT_ID:     'tenant_id',
} as const;

// ─── Factory ──────────────────────────────────────────────────────────────────

let isRefreshing = false;
let failedQueue: Array<{ resolve: (v: string) => void; reject: (e: unknown) => void }> = [];

function processQueue(error: unknown, token?: string) {
  failedQueue.forEach(({ resolve, reject }) =>
    error ? reject(error) : resolve(token!),
  );
  failedQueue = [];
}

export function createHttpClient(baseURL: string): AxiosInstance {
  const client = axios.create({
    baseURL,
    timeout: 30_000,
    headers: { 'Content-Type': 'application/json' },
  });

  // ── Request interceptor: inject auth + tenant ────────────────────────────
  client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
    const token    = localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN);
    const tenantId = localStorage.getItem(STORAGE_KEYS.TENANT_ID);

    if (token)    config.headers.Authorization = `Bearer ${token}`;
    if (tenantId) config.headers['X-Tenant-ID'] = tenantId;

    return config;
  });

  // ── Response interceptor: 401 auto-refresh ───────────────────────────────
  client.interceptors.response.use(
    (response) => response,
    async (error: AxiosError<ApiError>) => {
      const original = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

      if (error.response?.status === 401 && !original._retry) {
        if (isRefreshing) {
          // Queue requests while refresh is in flight
          return new Promise((resolve, reject) => {
            failedQueue.push({
              resolve: (token) => {
                original.headers.Authorization = `Bearer ${token}`;
                resolve(client(original));
              },
              reject,
            });
          });
        }

        original._retry = true;
        isRefreshing = true;

        const refreshToken = localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN);
        if (!refreshToken) {
          clearAuthAndRedirect();
          return Promise.reject(new AppError('No refresh token', 'AUTH_NO_TOKEN', 401));
        }

        try {
          // Lazy import to avoid circular dependency
          const { authClient } = await import('./auth.client');
          const { access_token } = await authClient.refresh(refreshToken);

          localStorage.setItem(STORAGE_KEYS.ACCESS_TOKEN, access_token);
          original.headers.Authorization = `Bearer ${access_token}`;
          processQueue(null, access_token);
          return client(original);
        } catch (refreshError) {
          processQueue(refreshError);
          clearAuthAndRedirect();
          return Promise.reject(new AppError('Session expired', 'AUTH_EXPIRED', 401));
        } finally {
          isRefreshing = false;
        }
      }

      // Map axios error → AppError
      const data = error.response?.data;
      throw new AppError(
        data?.message ?? error.message,
        data?.code ?? 'UNKNOWN',
        error.response?.status ?? 0,
      );
    },
  );

  return client;
}

function clearAuthAndRedirect() {
  localStorage.removeItem(STORAGE_KEYS.ACCESS_TOKEN);
  localStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
  localStorage.removeItem(STORAGE_KEYS.TENANT_ID);
  window.location.replace('/login');
}

// ─── Singleton instance ───────────────────────────────────────────────────────

export const httpClient = createHttpClient(
  import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080',
);
```

---

## QueryClient Configuration

### `ui/src/api/queryClient.ts`

```typescript
import { QueryClient } from '@tanstack/react-query';
import { AppError } from './clients/http.client';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,          // 30s mặc định
      retry: (failureCount, error) => {
        // Không retry với auth errors
        if (error instanceof AppError && error.status === 401) return false;
        if (error instanceof AppError && error.status === 403) return false;
        return failureCount < 2;
      },
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: false,
    },
  },
});
```

---

## Verification

```bash
cd ui
npx tsc --noEmit

# Kiểm tra interceptors hoạt động:
# 1. Login → access_token lưu
# 2. Fake expire token → next request tự refresh
# 3. Refresh fail → redirect /login
```
