# UI Solution: UI-SOL-005 — Response Schema Contracts Alignment

**Solution ID:** UI-SOL-005  
**CR References:** [CR-004-response-schema-contracts](../../../../docs/crs/v2/api-update/CR-004-response-schema-contracts.md)  
**Backend Solution:** [SOL-004-response-schema-contracts.md](../../../../backend/specs/crs/v2/api-update/solutions/SOL-004-response-schema-contracts.md)  
**Feature:** Response Schema — Type Safety & Null Handling  
**Priority:** 🔴 Critical  
**Frontend Component:** `ui/src/types/*.ts`, `ui/src/lib/api-client.ts`

---

## 1. Mục Đích

Align tất cả TypeScript types với backend response schemas:
- Chuẩn hóa PaginatedResponse (snake_case + camelCase aliases)
- Fix nullable fields (`temporalValidity`, `versionChain`, `last_used`)
- Chuẩn hóa datetime format (ISO 8601 string)
- Fix error response handling

---

## 2. Schema Contracts

### 2.1 PaginatedResponse — Dual Field Names

```typescript
// Backend trả về CẢ HAI: snake_case và camelCase aliases
// Frontend phải accept cả 2 formats

interface PaginatedResponse<T> {
  data:      T[];
  total:     number;
  page:      number;
  page_size: number;   // snake_case
  pageSize:  number;   // camelCase alias
  has_more:  boolean;
  hasMore:   boolean;  // camelCase alias
}

// Frontend sử dụng: response.page_size ?? response.pageSize
// Helper function:
export function getPaginationMeta(r: PaginatedResponse<unknown>) {
  return {
    pageSize: r.page_size ?? r.pageSize,
    hasMore:  r.has_more ?? r.hasMore,
    total:    r.total,
    page:     r.page,
  };
}
```

### 2.2 MemoryItem — Nullable Fields

```typescript
interface MemoryItem {
  id:               string;
  engine:           EngineType;
  memoryType:       MemoryType;
  title:            string;
  summary:          string;
  content:          string;
  score:            number;
  entities:         string[];
  sourceSessions:   string[];
  temporalValidity: {
    from: string | null;   // NULLABLE — must handle null
    to:   string | null;   // NULLABLE
  };
  policyTags:       string[];
  versionChain:     string | null;   // NULLABLE — SM only
  metadata:         Record<string, unknown>;
}

// Display helper:
function formatTemporalValidity(tv: MemoryItem['temporalValidity']): string {
  if (!tv.from && !tv.to) return 'No temporal bounds';
  if (!tv.to) return `From ${formatDate(tv.from!)}`;
  return `${formatDate(tv.from!)} → ${formatDate(tv.to)}`;
}
```

### 2.3 APIKey — last_used Optional

```typescript
interface APIKey {
  id:          string;
  name:        string;
  prefix:      string;
  scopes:      string[];
  created_at:  string;
  last_used?:  string | null;   // OPTIONAL + NULLABLE
  expires_at?: string | null;   // OPTIONAL + NULLABLE
  status:      'active' | 'revoked' | 'expired';
}

// Display: last_used ?? "Never used"
// Display: expires_at ?? "Never expires"
```

### 2.4 Error Response — Standard Format

```typescript
// ui/src/lib/api-client.ts

interface ApiErrorResponse {
  message: string;
  code:    string;         // "INVALID_ARGUMENT", "NOT_FOUND", etc.
  status:  number;
  details?: Record<string, unknown>;
}

// Error handling in api-client
async function handleResponse(res: Response): Promise<unknown> {
  if (!res.ok) {
    const error: ApiErrorResponse = await res.json().catch(() => ({
      message: 'Unknown error',
      code:    'INTERNAL_ERROR',
      status:  res.status,
    }));
    throw new ApiError(error);
  }
  return res.json();
}

class ApiError extends Error {
  constructor(public readonly response: ApiErrorResponse) {
    super(response.message);
    this.name = 'ApiError';
  }
}
```

---

## 3. Error UI Mapping

```typescript
// ui/src/hooks/useErrorHandler.ts

export function getErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    switch (error.response.code) {
      case 'INVALID_ARGUMENT': return `Validation error: ${error.response.message}`;
      case 'NOT_FOUND':        return 'Resource not found';
      case 'PERMISSION_DENIED': return 'Access denied — insufficient permissions';
      case 'RESOURCE_EXHAUSTED': return `Rate limited — retry in ${getRetryAfter()}s`;
      default:                 return error.response.message;
    }
  }
  return 'An unexpected error occurred';
}

// HTTP Status → UI behavior mapping
// 400 → show field validation errors
// 401 → auto-refresh token (api-client.ts handles)
// 403 → show AccessDenied component
// 404 → show NotFound component  
// 429 → show RateLimitToast with retry-after
// 500 → show GenericError with retry button
// 503 → show ServiceUnavailable
// 504 → show TimeoutError with retry
```

---

## 4. Runtime Validation

```typescript
// Use Zod for runtime schema validation in development
// ui/src/lib/validators.ts

import { z } from 'zod';

const MemoryItemSchema = z.object({
  id:           z.string(),
  engine:       z.enum(['cognee', 'graphiti', 'zep', 'openviking', 'memobase', 'supermemory', 'kgs']),
  memoryType:   z.enum(['episodic', 'semantic', 'conversational', 'procedural', 'profile', 'adaptive']),
  score:        z.number().min(0).max(1),
  temporalValidity: z.object({
    from: z.string().nullable(),
    to:   z.string().nullable(),
  }),
  versionChain: z.string().nullable(),
  // ... etc.
});

// In development: validate API responses, log schema mismatches
// In production: skip validation for performance
```

---

## 5. Acceptance Criteria (Frontend)

- [ ] `PaginatedResponse`: cả `page_size` và `pageSize` đều accessible
- [ ] `MemoryItem.temporalValidity`: null values handled gracefully (không crash)
- [ ] `MemoryItem.versionChain`: null → "No versions" label
- [ ] `APIKey.last_used`: null/undefined → "Never used"
- [ ] `ApiError` class wraps tất cả API errors với `code` field
- [ ] HTTP 429 → RateLimitToast với `Retry-After` countdown
- [ ] HTTP 503 → ServiceUnavailable component (không generic error)
- [ ] Zod validation: schema mismatches logged in console (dev only)
