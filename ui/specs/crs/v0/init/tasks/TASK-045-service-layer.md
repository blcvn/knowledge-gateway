---
id: TASK-045
title: API Service Layer — 12 Service Adapter Files
service: ui
version: 1.0.0
status: Done
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
depends_on: TASK-044
---

## Mục Tiêu
Tạo service layer tại `src/services/` — adapter giữa `api-client.ts` và React Query hooks. Mỗi service file chứa các function gọi API endpoint cụ thể, trả về typed responses.

## Scope

### In Scope
- 12 service files mapping toàn bộ API endpoints trong UX Spec Section 11
- Sử dụng `apiClient` từ `src/lib/api-client.ts`
- Type-safe request/response dùng types từ `src/types/`

### Out of Scope
- React Query hooks (TASK-047)
- Mock data fallback logic (TASK-046)

## Thiết Kế Kỹ Thuật

### Service File Pattern

Mỗi service file tuân theo pattern chuẩn:

```typescript
// src/services/[module].service.ts
import { apiClient } from '@/lib/api-client';
import { API_CONFIG } from '@/config/api.config';
import type { SomeType } from '@/types/[module]';

const BASE = API_CONFIG.gateway.admin; // hoặc engine-specific

export const dashboardService = {
  getMetrics: () => 
    apiClient.get<MetricsResponse>(`${BASE}/metrics`),
  
  getHealth: () => 
    apiClient.get<EngineHealth[]>(`${BASE}/health`),
  
  getThroughput: () => 
    apiClient.get<ThroughputData>(`${BASE}/throughput`),
};
```

### 12 Service Files (Mapping UX Spec Section 11)

| # | File | API Endpoints | Engine |
|---|---|---|---|
| 1 | `dashboard.service.ts` | `GET /v1/admin/metrics`, `GET /v1/admin/health`, `GET /v1/admin/throughput` | Gateway |
| 2 | `memory.service.ts` | `GET /v1/memory/search`, `GET /v1/memory/{id}`, `GET /v1/memory/{id}/neighbors` | Gateway |
| 3 | `profile.service.ts` | `POST /api/v1/users`, `GET /api/v1/users/{uid}`, `GET /api/v1/users/profile/{uid}`, `POST /api/v1/users/profile/{uid}`, `GET /api/v1/users/context/{uid}`, `GET /api/v1/users/event/{uid}`, `GET /api/v1/users/event/search/{uid}`, `POST /api/v1/users/buffer/{uid}/{type}`, `GET /api/v1/users/buffer/capacity/{uid}/{type}`, `GET /api/v1/project/profile_config`, `POST /api/v1/project/profile_config`, `GET /api/v1/project/billing`, `GET /api/v1/project/usage` | Memobase |
| 4 | `adaptive.service.ts` | `POST /api/v1/documents`, `GET /api/v1/memories`, `GET /api/v1/memories/{id}/versions`, `GET /api/v1/search`, `GET /api/v1/profiles`, `GET /api/v1/connectors`, `POST /api/v1/connectors`, `GET /api/v1/analytics`, `GET /api/v1/projects` | Supermemory |
| 5 | `graph.service.ts` | `GET /v1/graph/subgraph`, `GET /v1/graph/timeline`, `POST /v1/graph/query` | Gateway |
| 6 | `session.service.ts` | Session/Thread/User APIs via Gateway + Zep | Zep |
| 7 | `governance.service.ts` | `GET /v1/admin/tenants`, `POST /v1/admin/policies`, `GET /v1/admin/audit` | Gateway |
| 8 | `pipeline.service.ts` | Pipeline status, job queue APIs per engine | Gateway |
| 9 | `infrastructure.service.ts` | Service topology, DB health, resource metrics | Gateway |
| 10 | `observability.service.ts` | Metrics, traces, logs, errors, cost APIs | Gateway |
| 11 | `cognee.service.ts` | `POST /api/v1/cognee/add`, `GET /api/v1/cognee/datasets`, `POST /api/v1/cognee/cognify`, `GET /api/v1/cognee/cognify/{id}/status`, `POST /api/v1/cognee/search`, `GET /api/v1/cognee/search/explore`, `POST /api/v1/cognee/search/rag` | Cognee |
| 12 | `zep.service.ts` | `POST /api/v1/users`, `GET /api/v1/threads`, `POST /api/v1/memories`, `GET /api/v1/graph`, `GET /api/v1/search` | Zep |

### Profile Service Detail (`src/services/profile.service.ts`)

```typescript
import { apiClient } from '@/lib/api-client';
import { API_CONFIG } from '@/config/api.config';
import type {
  UserProfile, ProfileConfig, BufferZone,
  UserEvent, ContextAssembly, ProjectBilling, ProjectUsage
} from '@/types/profile';

const BASE = API_CONFIG.engines.memobase.baseUrl;

export const profileService = {
  // User CRUD
  createUser: (data: { user_id: string }) =>
    apiClient.post<{ user_id: string }>(`${BASE}/users`, data),

  getUser: (userId: string) =>
    apiClient.get<UserProfile>(`${BASE}/users/${userId}`),

  // Profile Management
  getProfile: (userId: string) =>
    apiClient.get<UserProfile>(`${BASE}/users/profile/${userId}`),

  updateProfile: (userId: string, data: Partial<UserProfile>) =>
    apiClient.post<UserProfile>(`${BASE}/users/profile/${userId}`, data),

  // Context Assembly
  getContext: (userId: string) =>
    apiClient.get<ContextAssembly>(`${BASE}/users/context/${userId}`),

  // Events
  getEvents: (userId: string) =>
    apiClient.get<UserEvent[]>(`${BASE}/users/event/${userId}`),

  searchEvents: (userId: string, query: string) =>
    apiClient.get<UserEvent[]>(`${BASE}/users/event/search/${userId}?q=${query}`),

  // Buffer Zone
  flushBuffer: (userId: string, bufferType: string) =>
    apiClient.post<void>(`${BASE}/users/buffer/${userId}/${bufferType}`, {}),

  getBufferCapacity: (userId: string, bufferType: string) =>
    apiClient.get<BufferZone>(`${BASE}/users/buffer/capacity/${userId}/${bufferType}`),

  // Project Config
  getProfileConfig: () =>
    apiClient.get<ProfileConfig>(`${BASE}/project/profile_config`),

  updateProfileConfig: (config: ProfileConfig) =>
    apiClient.post<ProfileConfig>(`${BASE}/project/profile_config`, config),

  // Billing & Usage
  getBilling: () =>
    apiClient.get<ProjectBilling>(`${BASE}/project/billing`),

  getUsage: () =>
    apiClient.get<ProjectUsage>(`${BASE}/project/usage`),
};
```

## Acceptance Criteria
- [x] AC-1: 12 service files tồn tại trong `src/services/`
- [x] AC-2: Tất cả function return typed Promise<T> (không `any`)
- [x] AC-3: Tất cả sử dụng `apiClient` từ `src/lib/api-client.ts`
- [x] AC-4: Profile service cover 100% Memobase API endpoints từ UX Spec
- [x] AC-5: Adaptive service cover 100% Supermemory API endpoints từ UX Spec
- [x] AC-6: TSC compile pass

## Definition of Done
- [x] Files tạo đúng thư mục
- [x] TypeScript compile pass
- [x] ESLint pass
