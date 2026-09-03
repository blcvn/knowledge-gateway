# TASK-UI-001 — Cập nhật `api.config.ts`: Auth namespace & Console namespaces

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-001 |
| **Layer** | Frontend — TypeScript |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-002 §4.2](../solutions/SOL-002-Auth-Solution.md) |
| **Priority** | 🔴 P0 — Critical |
| **Depends On** | — |
| **Estimated** | 0.5h |

---

## Context

File `ui/src/config/api.config.ts` hiện tại chưa có namespace `auth` và các namespace console đầy đủ. Tất cả các service files cần import từ đây để đảm bảo consistent base paths.

---

## Goal

Cập nhật `API_CONFIG` object để bao gồm tất cả namespaces cần thiết cho migration.

---

## Target Files

| Action | File Path |
|---|---|
| MODIFY | `ui/src/config/api.config.ts` |

---

## Implementation

### File: `ui/src/config/api.config.ts`

```typescript
export const API_CONFIG = {
  baseUrl: import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080',
  useMockData: import.meta.env.VITE_USE_MOCK_DATA === 'true',

  // Auth
  auth:          '/v1/auth',

  // Console namespaces
  dashboard:     '/v1/console/dashboard',
  sessions:      '/v1/console/sessions',
  memory:        '/v1/console/memory',
  profiles:      '/v1/console/profiles',
  adaptive:      '/v1/console/adaptive',
  governance:    '/v1/console/governance',
  observability: '/v1/console/observability',
  pipelines:     '/v1/console/pipelines',
  infra:         '/v1/console/infra',
  org:           '/v1/console/org',
  sdk:           '/v1/console/sdk',

  // Engine direct APIs (unchanged)
  graphiti:      '/v1/graphiti',
  memobase:      '/v1/memobase',
  zep:           '/v1/zep',
  cognee:        '/v1/cognee',
  supermemory:   '/v1/sm',
  openviking:    '/v1/ov',
  admin:         '/v1/admin',
} as const;

export type APINamespace = keyof typeof API_CONFIG;
```

---

## Verification

```bash
cd ui
npx tsc --noEmit
```

**Expected**: No type errors. All service files that import `API_CONFIG.auth` compile correctly.
