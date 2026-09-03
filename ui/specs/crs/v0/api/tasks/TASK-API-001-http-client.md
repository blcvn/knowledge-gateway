# TASK-API-001 — HTTP Client Foundation

**Task ID:** TASK-API-001
**Status:** ✅ COMPLETED — 2026-06-17
**Sprint:** 1 — Foundation
**Solution:** [API-SOL-001](../API-SOL-001-http-client.md)
**Depends on:** —
**Ước tính:** 1h
**Priority:** P0 — Critical (Blocker cho toàn bộ API layer)

---

## Mục tiêu

Tạo Axios HTTP client dùng chung cho toàn bộ Data Access Layer:
1. `AppError` class — chuẩn hóa lỗi từ backend
2. `STORAGE_KEYS` — quản lý localStorage keys tập trung
3. Request interceptor — auto-inject `Authorization` + `X-Tenant-ID` headers
4. Response interceptor — 401 auto-refresh + request queue
5. `httpClient` singleton export
6. `queryClient` — TanStack Query config với retry logic

---

## Công việc cụ thể

### 1. Cài đặt dependencies (nếu chưa có)

```bash
cd ui
npm install axios @tanstack/react-query
```

### 2. Tạo `ui/src/api/clients/http.client.ts`

**Nội dung đầy đủ:**

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

// ─── Refresh Queue ────────────────────────────────────────────────────────────

let isRefreshing = false;
let failedQueue: Array<{ resolve: (v: string) => void; reject: (e: unknown) => void }> = [];

function processQueue(error: unknown, token?: string) {
  failedQueue.forEach(({ resolve, reject }) =>
    error ? reject(error) : resolve(token!),
  );
  failedQueue = [];
}

// ─── Factory ──────────────────────────────────────────────────────────────────

export function createHttpClient(baseURL: string): AxiosInstance {
  const client = axios.create({
    baseURL,
    timeout: 30_000,
    headers: { 'Content-Type': 'application/json' },
  });

  // ── Request interceptor: inject auth + tenant headers ────────────────────
  client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
    const token    = localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN);
    const tenantId = localStorage.getItem(STORAGE_KEYS.TENANT_ID);

    if (token)    config.headers.Authorization = `Bearer ${token}`;
    if (tenantId) config.headers['X-Tenant-ID'] = tenantId;

    return config;
  });

  // ── Response interceptor: 401 auto-refresh + request queue ──────────────
  client.interceptors.response.use(
    (response) => response,
    async (error: AxiosError<ApiError>) => {
      const original = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

      if (error.response?.status === 401 && !original._retry) {
        if (isRefreshing) {
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
          // Lazy import để tránh circular dependency với auth.client
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

### 3. Tạo `ui/src/api/queryClient.ts`

```typescript
import { QueryClient } from '@tanstack/react-query';
import { AppError } from './clients/http.client';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: (failureCount, error) => {
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

### 4. Tạo `ui/src/api/index.ts` (barrel export)

```typescript
export { httpClient, createHttpClient, AppError, STORAGE_KEYS } from './clients/http.client';
export { queryClient } from './queryClient';
```

---

## Files tạo ra

```
ui/src/api/
├── clients/
│   └── http.client.ts   ← NEW
├── queryClient.ts        ← NEW
└── index.ts              ← NEW
```

---

## Acceptance Criteria

- [x] `cd ui && npx tsc --noEmit` không có lỗi liên quan đến `http.client.ts`
- [x] `httpClient` có type `AxiosInstance`
- [x] Request gửi ra đi kèm header `Authorization: Bearer <token>` nếu có token
- [x] Request gửi ra đi kèm header `X-Tenant-ID` nếu có tenantId
- [x] Khi nhận 401 và có refreshToken: tự động gọi refresh, retry request gốc
- [x] Khi nhận 401 và không có refreshToken: redirect về `/login`
- [x] Concurrent 401 requests được queue lại, không duplicate refresh call
- [x] `AppError` extend Error, có `code` và `status` fields
- [x] `queryClient` không retry với 401/403 errors

---

## Sau khi hoàn thành

```bash
cd ui
npx tsc --noEmit
# Không có lỗi type → chuyển sang TASK-API-002
```
