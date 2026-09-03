# TASK-UI-003 — Cập nhật `api-client.ts`: 401 Auto-Refresh Interceptor

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-003 |
| **Layer** | Frontend — TypeScript |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-002 §3.2](../solutions/SOL-002-Auth-Solution.md) |
| **Priority** | 🔴 P0 — Critical |
| **Depends On** | TASK-UI-002 |
| **Estimated** | 1h |

---

## Context

`ui/src/lib/api-client.ts` đã inject `Authorization` header và `x-tenant-id`, nhưng chưa có logic auto-refresh khi nhận 401. Khi access token hết hạn, mọi request sẽ fail thay vì tự động lấy token mới.

---

## Goal

- Thêm 401 interceptor: khi nhận 401, gọi `authService.refreshToken()` và retry request gốc 1 lần
- Nếu refresh cũng fail → redirect về `/login` và throw AppError

---

## Target Files

| Action | File Path |
|---|---|
| MODIFY | `ui/src/lib/api-client.ts` |

---

## Implementation

Tìm hàm `fetchWrapper` (hoặc tương đương) trong file và bổ sung logic sau **sau khi nhận response**:

```typescript
// Thêm import ở đầu file (tránh circular dependency bằng lazy import)
// KHÔNG import authService ở top-level — dùng lazy import trong function

async function fetchWithRefresh<T>(
  url: string,
  options: RequestInit,
  isRetry = false
): Promise<T> {
  const token = localStorage.getItem('access_token');
  const tenantId = localStorage.getItem('tenant_id');

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(tenantId ? { 'x-tenant-id': tenantId } : {}),
    ...(options.headers as Record<string, string> ?? {}),
  };

  const response = await fetch(url, { ...options, headers });

  if (response.status === 401 && !isRetry) {
    // Lazy import to avoid circular dependency
    const { authService } = await import('../services/auth');
    try {
      await authService.refreshToken();
      // Retry original request with new token
      return fetchWithRefresh<T>(url, options, true);
    } catch {
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      localStorage.removeItem('tenant_id');
      window.location.href = '/login';
      throw new AppError('Session expired. Please login again.', 'AUTH_EXPIRED', 401);
    }
  }

  if (!response.ok) {
    const errBody = await response.json().catch(() => ({}));
    throw new AppError(
      errBody.message ?? `HTTP ${response.status}`,
      errBody.code ?? 'UNKNOWN',
      response.status
    );
  }

  // 204 No Content
  if (response.status === 204) return undefined as T;

  return response.json() as Promise<T>;
}
```

**Quan trọng**: Đảm bảo `apiClient.get`, `apiClient.post`, `apiClient.put`, `apiClient.delete` đều gọi `fetchWithRefresh` thay vì fetch trực tiếp.

---

## Verification

```bash
cd ui
npx tsc --noEmit
```

Manual test:
1. Login thành công → access_token lưu
2. Expire token thủ công (xóa trong DevTools, hoặc set exp cũ)
3. Gọi một API protected → tự động refresh và request thành công
4. Nếu refresh cũng fail → redirect `/login`
