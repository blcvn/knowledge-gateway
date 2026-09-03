---
id: TASK-047
title: React Query Custom Hooks Layer
service: ui
version: 1.0.0
status: Done
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
depends_on: TASK-045, TASK-046
---

## Mục Tiêu
Tạo custom React Query hooks tại `src/hooks/` — bridge giữa service layer và UI components. Mỗi hook file cung cấp `useQuery`/`useMutation` wrappers typed cho một domain.

## Scope
### In Scope
- 10 hook files covering tất cả modules
- Smart caching với queryKey patterns
- Mock data fallback integration
- Error handling via AppError

### Out of Scope
- Component UI changes (separate tasks T05-T16)

## Thiết Kế Kỹ Thuật

### Hook Files

| File | Key Hooks |
|---|---|
| `useDashboard.ts` | `useMetrics()`, `useEngineHealth()`, `useMemoryFlow()`, `useThroughput()` |
| `useMemory.ts` | `useMemorySearch(query, filters)`, `useMemoryDetail(id)`, `useMemoryNeighbors(id)` |
| `useProfiles.ts` | `useProfileList()`, `useProfileDetail(userId)`, `useProfileConfig()`, `useBufferStatus(userId)`, `useUserEvents(userId)`, `useContextAssembly(userId)`, `useProjectUsage()` |
| `useAdaptiveMemory.ts` | `useAdaptiveMemories()`, `useMemoryVersions(id)`, `useConnectors()`, `useForgetRules()`, `useAdaptiveAnalytics()` |
| `useGraph.ts` | `useSubgraph(params)`, `useTimeline(range)`, `useGraphQuery(cypher)` |
| `useSessions.ts` | `useSessionList()`, `useSessionDetail(id)` |
| `useGovernance.ts` | `useTenants()`, `usePolicies()`, `useAuditLogs(filters)` |
| `usePipelines.ts` | `usePipelineJobs(engine)`, `useQueueMetrics()` |
| `useInfrastructure.ts` | `useServiceHealth()`, `useDatabaseHealth()`, `useResourceMetrics()` |
| `useObservability.ts` | `useMetricsDashboard()`, `useTraces(filters)`, `useErrors(filters)` |

### Hook Pattern

```typescript
// src/hooks/useProfiles.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { profileService } from '@/services/profile.service';
import { profileMock } from '@/mock/profile.mock';
import { API_CONFIG } from '@/config/api.config';

const useMock = API_CONFIG.useMockData;

export function useProfileList() {
  return useQuery({
    queryKey: ['profiles'],
    queryFn: useMock
      ? () => Promise.resolve(profileMock.users)
      : () => profileService.listUsers(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useProfileDetail(userId: string) {
  return useQuery({
    queryKey: ['profiles', userId],
    queryFn: useMock
      ? () => Promise.resolve(profileMock.userDetail)
      : () => profileService.getProfile(userId),
    enabled: !!userId,
  });
}

export function useUpdateProfileConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: profileService.updateProfileConfig,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['profileConfig'] }),
  });
}
```

### QueryKey Convention
- `['metrics']` — dashboard metrics
- `['engineHealth']` — engine health grid
- `['memories', { query, type, engine }]` — memory search
- `['profiles']` — profile list
- `['profiles', userId]` — profile detail
- `['adaptive', 'memories']` — adaptive memory list
- `['adaptive', 'versions', memoryId]` — version chain

## Acceptance Criteria
- [x] AC-1: 10 hook files trong `src/hooks/`
- [x] AC-2: Tất cả hooks return `UseQueryResult<T>` hoặc `UseMutationResult<T>`
- [x] AC-3: Mock fallback hoạt động khi `VITE_USE_MOCK_DATA=true`
- [x] AC-4: QueryKey patterns nhất quán, hỗ trợ cache invalidation
- [x] AC-5: Profile hooks cover CRUD + context assembly + buffer
- [x] AC-6: Adaptive hooks cover versions + connectors + analytics

## Definition of Done
- [x] TypeScript compile pass
- [x] ESLint pass
