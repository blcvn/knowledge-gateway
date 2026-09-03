/**
 * Enterprise-grade API client for VNP Memory Console
 *
 * Features:
 * - Auto-inject Authorization + X-Tenant-ID headers
 * - 401 auto-refresh with concurrent request queue (no duplicate refresh)
 * - Exponential backoff retry for network errors
 * - Timeout via AbortController
 * - Standardized AppError
 */

// ─── Error Types ──────────────────────────────────────────────────────────────

export interface ApiErrorPayload {
  message: string;
  code:    string;
}

export class AppError extends Error {
  code:       string;
  status:     number;
  requestId?: string;

  constructor(message: string, code: string, status: number, requestId?: string) {
    super(message);
    this.name      = 'AppError';
    this.code      = code;
    this.status    = status;
    this.requestId = requestId;
  }
}

// ─── Storage Keys ────────────────────────────────────────────────────────────

export const STORAGE_KEYS = {
  ACCESS_TOKEN:  'access_token',
  REFRESH_TOKEN: 'refresh_token',
  TENANT_ID:     'tenant_id',
} as const;

// ─── 401 Refresh Queue ───────────────────────────────────────────────────────

let isRefreshing = false;
let failedQueue: Array<{ resolve: (token: string) => void; reject: (e: unknown) => void }> = [];

function processQueue(error: unknown, token?: string) {
  failedQueue.forEach(({ resolve, reject }) =>
    error ? reject(error) : resolve(token!),
  );
  failedQueue = [];
}

function clearAuthAndRedirect() {
  localStorage.removeItem(STORAGE_KEYS.ACCESS_TOKEN);
  localStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
  localStorage.removeItem(STORAGE_KEYS.TENANT_ID);
  window.location.replace('/login');
}

// ─── Auth Header Helper ───────────────────────────────────────────────────────

function getAuthHeaders(): Record<string, string> {
  const token    = localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN);
  const tenantId = localStorage.getItem(STORAGE_KEYS.TENANT_ID) ?? 'default-tenant';
  const headers: Record<string, string> = {
    'X-Tenant-ID': tenantId,
    'X-Client':    'vnp-memory-console/1.0',
  };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  return headers;
}

// ─── Core Options ─────────────────────────────────────────────────────────────

interface FetchOptions extends RequestInit {
  timeout?:    number;   // ms, default 30_000
  retries?:    number;   // default 3 for GET, 0 for mutations
  retryDelay?: number;   // base ms for exponential backoff, default 500
  _retry?:     boolean;  // internal: marks a retry after 401 refresh
}

const DEFAULT_TIMEOUT     = 30_000;
const DEFAULT_GET_RETRIES = 3;
const BASE_RETRY_DELAY    = 500;

async function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// ─── Core fetchWrapper ────────────────────────────────────────────────────────

async function fetchWrapper<T>(url: string, options: FetchOptions): Promise<T> {
  const {
    timeout    = DEFAULT_TIMEOUT,
    retries    = 0,
    retryDelay = BASE_RETRY_DELAY,
    _retry     = false,
    ...fetchInit
  } = options;

  const headers: Record<string, string> = {
    ...getAuthHeaders(),
    ...(fetchInit.headers as Record<string, string> | undefined),
  };

  let attempt = 0;
  while (true) {
    const controller = new AbortController();
    const timeoutId  = setTimeout(() => controller.abort(), timeout);

    try {
      const response = await fetch(url, {
        ...fetchInit,
        headers,
        signal: controller.signal,
      });
      clearTimeout(timeoutId);

      const requestId = response.headers.get('x-request-id') ?? undefined;

      // ── 401: attempt refresh ────────────────────────────────────────────────
      if (response.status === 401 && !_retry) {
        const refreshToken = localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN);
        if (!refreshToken) {
          clearAuthAndRedirect();
          return Promise.reject(new AppError('No refresh token', 'AUTH_NO_TOKEN', 401));
        }

        if (isRefreshing) {
          // Queue this request until refresh completes
          return new Promise<T>((resolve, reject) => {
            failedQueue.push({
              resolve: (newToken) => {
                headers['Authorization'] = `Bearer ${newToken}`;
                resolve(fetchWrapper<T>(url, { ...options, _retry: true }));
              },
              reject,
            });
          });
        }

        isRefreshing = true;
        try {
          const refreshRes = await fetch(
            `${import.meta.env.VITE_API_BASE_URL ?? ''}/v1/auth/refresh`,
            {
              method:  'POST',
              headers: { 'Content-Type': 'application/json' },
              body:    JSON.stringify({ refresh_token: refreshToken }),
            },
          );

          if (!refreshRes.ok) throw new Error('Refresh failed');

          const { access_token } = await refreshRes.json() as { access_token: string };
          localStorage.setItem(STORAGE_KEYS.ACCESS_TOKEN, access_token);
          headers['Authorization'] = `Bearer ${access_token}`;
          processQueue(null, access_token);
          return fetchWrapper<T>(url, { ...options, _retry: true });
        } catch (refreshErr) {
          processQueue(refreshErr);
          clearAuthAndRedirect();
          return Promise.reject(new AppError('Session expired', 'AUTH_EXPIRED', 401));
        } finally {
          isRefreshing = false;
        }
      }

      // ── Non-OK responses ────────────────────────────────────────────────────
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({} as ApiErrorPayload));
        throw new AppError(
          errorData.message ?? `HTTP ${response.status}: ${response.statusText}`,
          errorData.code    ?? `HTTP_${response.status}`,
          response.status,
          requestId,
        );
      }

      // 204 No Content
      if (response.status === 204) return undefined as T;
      return (await response.json()) as T;

    } catch (error) {
      clearTimeout(timeoutId);

      const isAbort        = error instanceof DOMException && error.name === 'AbortError';
      const isNetworkError = error instanceof TypeError;
      const shouldRetry    = (isAbort || isNetworkError) && attempt < retries;

      if (shouldRetry) {
        const delay = retryDelay * Math.pow(2, attempt);
        console.warn(`[api-client] Retry ${attempt + 1}/${retries} after ${delay}ms`);
        await sleep(delay);
        attempt++;
        continue;
      }

      if (isAbort) throw new AppError('Request timed out', 'REQUEST_TIMEOUT', 408);

      if (error instanceof AppError) throw error;

      console.error('[api-client] Unexpected error:', { url, error });
      throw error;
    }
  }
}

// ─── Public API ───────────────────────────────────────────────────────────────

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '';

export const apiClient = {
  get: <T>(path: string, options?: FetchOptions): Promise<T> =>
    fetchWrapper<T>(`${BASE_URL}${path}`, {
      ...options,
      method:  'GET',
      retries: DEFAULT_GET_RETRIES,
    }),

  post: <T>(path: string, body: unknown, options?: FetchOptions): Promise<T> =>
    fetchWrapper<T>(`${BASE_URL}${path}`, {
      ...options,
      method:  'POST',
      body:    JSON.stringify(body),
      headers: { 'Content-Type': 'application/json', ...options?.headers },
    }),

  put: <T>(path: string, body: unknown, options?: FetchOptions): Promise<T> =>
    fetchWrapper<T>(`${BASE_URL}${path}`, {
      ...options,
      method:  'PUT',
      body:    JSON.stringify(body),
      headers: { 'Content-Type': 'application/json', ...options?.headers },
    }),

  patch: <T>(path: string, body: unknown, options?: FetchOptions): Promise<T> =>
    fetchWrapper<T>(`${BASE_URL}${path}`, {
      ...options,
      method:  'PATCH',
      body:    JSON.stringify(body),
      headers: { 'Content-Type': 'application/json', ...options?.headers },
    }),

  delete: <T = void>(path: string, options?: FetchOptions): Promise<T> =>
    fetchWrapper<T>(`${BASE_URL}${path}`, {
      ...options,
      method: 'DELETE',
    }),
};
