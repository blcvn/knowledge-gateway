/**
 * Enterprise-grade API client for VNP Memory Console
 * Features: CRUD operations, auth injection, retry + backoff, timeout, error enrichment
 */

export class AppError extends Error {
  code: string;
  status: number;
  requestId?: string;

  constructor(message: string, code: string, status: number, requestId?: string) {
    super(message);
    this.name = 'AppError';
    this.code = code;
    this.status = status;
    this.requestId = requestId;
  }
}

interface FetchOptions extends RequestInit {
  timeout?: number;        // ms, default 30_000
  retries?: number;        // default 3 for GET, 0 for mutations
  retryDelay?: number;     // base ms for exponential backoff, default 500
}

const DEFAULT_TIMEOUT = 30_000;
const DEFAULT_RETRIES = 3;
const BASE_RETRY_DELAY = 500;

function getAuthHeaders(): Record<string, string> {
  const tenantId = localStorage.getItem('tenant_id') ?? 'default-tenant';
  const token = localStorage.getItem('access_token');
  const headers: Record<string, string> = {
    'x-tenant-id': tenantId,
    'x-client': 'vnp-memory-console/1.0',
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  return headers;
}

async function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function fetchWrapper<T>(url: string, options: FetchOptions): Promise<T> {
  const {
    timeout = DEFAULT_TIMEOUT,
    retries = 0,
    retryDelay = BASE_RETRY_DELAY,
    ...fetchInit
  } = options;

  const headers = {
    ...getAuthHeaders(),
    ...fetchInit.headers,
  };

  let attempt = 0;
  while (true) {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeout);

    try {
      const response = await fetch(url, {
        ...fetchInit,
        headers,
        signal: controller.signal,
      });
      clearTimeout(timeoutId);

      const requestId = response.headers.get('x-request-id') ?? undefined;

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new AppError(
          errorData.message ?? `HTTP ${response.status}: ${response.statusText}`,
          errorData.code ?? `HTTP_${response.status}`,
          response.status,
          requestId,
        );
      }

      // Handle 204 No Content
      if (response.status === 204) return undefined as T;
      return (await response.json()) as T;
    } catch (error) {
      clearTimeout(timeoutId);

      const isAbort = error instanceof DOMException && error.name === 'AbortError';
      const isNetworkError = error instanceof TypeError;
      const shouldRetry = (isAbort || isNetworkError) && attempt < retries;

      if (shouldRetry) {
        const delay = retryDelay * Math.pow(2, attempt); // exponential backoff
        console.warn(`[api-client] Retry ${attempt + 1}/${retries} after ${delay}ms for ${url}`);
        await sleep(delay);
        attempt++;
        continue;
      }

      // Enrich timeout error
      if (isAbort) {
        throw new AppError('Request timed out', 'REQUEST_TIMEOUT', 408);
      }

      // Log to error tracking (Sentry hook point)
      if (process.env.NODE_ENV !== 'test') {
        console.error('[api-client] Error:', { url, error });
      }

      throw error;
    }
  }
}

export const apiClient = {
  get: <T>(url: string, options?: FetchOptions): Promise<T> =>
    fetchWrapper<T>(url, { ...options, method: 'GET', retries: DEFAULT_RETRIES }),

  post: <T>(url: string, body: unknown, options?: FetchOptions): Promise<T> =>
    fetchWrapper<T>(url, {
      ...options,
      method: 'POST',
      body: JSON.stringify(body),
      headers: { 'Content-Type': 'application/json', ...options?.headers },
    }),

  put: <T>(url: string, body: unknown, options?: FetchOptions): Promise<T> =>
    fetchWrapper<T>(url, {
      ...options,
      method: 'PUT',
      body: JSON.stringify(body),
      headers: { 'Content-Type': 'application/json', ...options?.headers },
    }),

  patch: <T>(url: string, body: unknown, options?: FetchOptions): Promise<T> =>
    fetchWrapper<T>(url, {
      ...options,
      method: 'PATCH',
      body: JSON.stringify(body),
      headers: { 'Content-Type': 'application/json', ...options?.headers },
    }),

  delete: <T>(url: string, options?: FetchOptions): Promise<T> =>
    fetchWrapper<T>(url, { ...options, method: 'DELETE' }),
};
