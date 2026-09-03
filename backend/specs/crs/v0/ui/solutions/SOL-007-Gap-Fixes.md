# SOL-007 — Gap Fixes: Service Files, Hook Code & Missing Schemas

| Field | Value |
|---|---|
| **Solution ID** | SOL-007 |
| **CRs** | CR-001, CR-002, CR-003, CR-004, CR-005, CR-006, CR-007, CR-008, CR-009, CR-010, CR-011 |
| **Mục đích** | Bổ sung các gap đã phát hiện trong TRACEABILITY.md — service files đầy đủ, hook code hoàn chỉnh, response schemas còn thiếu |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Implemented** | 2026-06-17 |

---

## 1. Gap CR-001 — `useStore.ts` & `api.config.ts`

Đã bổ sung vào **SOL-002 §4.1 và §4.2**. Xem [SOL-002](./SOL-002-Auth-Solution.md).

---

## 2. Gap CR-002 — KPIData fields còn thiếu

### 2.1 `contextSavingsPct` — Công thức tính

```go
// Nguồn: memobase-context service
// Formula: so sánh tokens của context assembly vs naive context (all messages)
type ContextSavingsCalc struct {
    AvgAssembledTokens float64  // E[tokens_in_assembled_context]
    AvgNaiveTokens     float64  // E[tokens nếu bỏ hết messages vào]
}

// Handler query:
assembled, _ := h.db.QueryRow(`
    SELECT AVG(token_count) FROM context_assemblies
    WHERE tenant_id = $1 AND created_at > NOW() - INTERVAL '1h'
`, tenantID).Scan(&assembled)

naive, _ := h.db.QueryRow(`
    SELECT AVG(message_count * 150)  -- estimate 150 tokens/message
    FROM sessions WHERE tenant_id = $1 AND updated_at > NOW() - INTERVAL '1h'
`, tenantID).Scan(&naive)

savingsPct := (1.0 - assembled/naive) * 100
```

### 2.2 `memoryVersions` — Nguồn dữ liệu

```go
// Nguồn: Supermemory sm-memory — count memories không phải is_latest
var versionCount int64
h.db.QueryRow(`
    SELECT COUNT(*) FROM sm_memories
    WHERE tenant_id = $1
`, tenantID).Scan(&versionCount)

// Hoặc gọi gRPC: sm-analytics.GetStats().TotalMemoryVersions
```

---

## 3. Gap CR-003 — `memory_sources` mapping & `/user-summary`

### 3.1 `memory_sources` mapping trong messages

Khi `ZepMessage` có metadata `memory_sources`, gateway cần map sang array:

```go
// Trong ConsoleSessionsHandler.GetDetail():
memSources := []string{}
if raw, ok := m.Metadata["memory_sources"]; ok {
    // metadata được store dạng JSON string array: '["graphiti:ep_abc", "memobase:prof_xyz"]'
    _ = json.Unmarshal([]byte(fmt.Sprint(raw)), &memSources)
}
msgs[i] = Message{
    // ...
    MemorySources: memSources,
}
```

### 3.2 `/v1/console/sessions/{id}/user-summary` (từ FEAT-014 architecture §4.2)

```go
// GET /v1/console/sessions/{id}/user-summary
// Lấy tóm tắt profile user của session này từ memobase-context
func (h *ConsoleSessionsHandler) GetUserSummary(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")
    session, _ := h.sessionRepo.Get(r.Context(), sessionID, authctx.TenantID(r.Context()))

    // Dùng memobase-context để lấy context assembled cho user của session
    ctx, _ := h.memoCtx.AssembleContext(r.Context(), &memobase.ContextRequest{
        UserID:   session.UserID,
        TenantID: authctx.TenantID(r.Context()),
    })

    httputil.JSON(w, 200, UserSummaryResponse{
        UserID:        session.UserID,
        ContextString: ctx.Summary,
        TokenCount:    ctx.Tokens,
    })
}
```

Route đăng ký:
```go
mux.HandleFunc("GET /v1/console/sessions/{id}/user-summary",
    authMiddleware(sessionsHandler.GetUserSummary))
```

---

## 4. Gap CR-004 — Empty state khi search không có kết quả

Frontend phải xử lý `results: []` gracefully:

```tsx
// ui/src/components/MemoryExplorer/MemorySearchResults.tsx
function MemorySearchResults() {
    const { data, isLoading, isError } = useMemorySearch(query);

    if (isLoading) return <SearchSkeleton />;

    if (isError) return (
        <EmptyState
            icon="alert-circle"
            title="Search failed"
            description="Could not reach the memory search service. Try again."
            action={<Button onClick={() => refetch()}>Retry</Button>}
        />
    );

    if (!data || data.results.length === 0) return (
        <EmptyState
            icon="search"
            title="No memories found"
            description={`No results for "${query.query}". Try different keywords or expand engine filters.`}
        />
    );

    return <MemoryList items={data.results} facets={data.facets} />;
}
```

---

## 5. Gap CR-005 — `AdaptiveAnalytics` schema, Hook & Service đầy đủ

### 5.1 `AdaptiveAnalytics` response schema đầy đủ

Backend `sm-analytics.GetStats()` phải trả về đủ 5 fields:

```go
// Go struct (camelCase JSON để khớp TypeScript AdaptiveAnalytics)
type AdaptiveAnalyticsResponse struct {
    CreationRate        float64 `json:"creation_rate"`           // memories/hour
    DeletionRate        float64 `json:"deletion_rate"`           // deletions/hour
    ContradictionCount  int     `json:"contradiction_count"`     // unresolved contradictions
    ConnectorSyncCount  int     `json:"connector_sync_count"`    // syncs in last 24h
    StorageUsageBytes   int64   `json:"storage_usage_bytes"`     // total bytes in sm_memories
}

// Query PostgreSQL:
SELECT
    COUNT(*)   FILTER (WHERE created_at > NOW() - INTERVAL '1h') AS creation_rate,
    COUNT(*)   FILTER (WHERE deleted_at > NOW() - INTERVAL '1h') AS deletion_rate,
    COUNT(*)   FILTER (WHERE relation_type = 'contradicts' AND resolved = false) AS contradiction_count,
    SUM(LENGTH(content)) AS storage_usage_bytes
FROM sm_memories WHERE tenant_id = $1;
```

### 5.2 `adaptive.service.ts` — đầy đủ methods

```typescript
// ui/src/services/adaptive.service.ts
import { apiClient } from '../lib/api-client';
import type { AdaptiveMemory, MemoryVersion, ExternalConnector, AdaptiveAnalytics, ForgetRules } from '../types/adaptive';

const BASE = '/v1/console/adaptive';

export const adaptiveService = {
    getMemories: () =>
        apiClient.get<AdaptiveMemory[]>(`${BASE}/memories`),

    getMemoryVersions: (id: string) =>
        apiClient.get<MemoryVersion[]>(`${BASE}/memories/${encodeURIComponent(id)}/versions`),

    getConnectors: () =>
        apiClient.get<ExternalConnector[]>(`${BASE}/connectors`),

    createConnector: (config: Partial<ExternalConnector>) =>
        apiClient.post<ExternalConnector>(`${BASE}/connectors`, config),

    syncConnector: (id: string) =>
        apiClient.post<{ job_id: string }>(`${BASE}/connectors/${id}/sync`, {}),

    getAnalytics: () =>
        apiClient.get<AdaptiveAnalytics>(`${BASE}/analytics`),

    getForgetRules: () =>
        apiClient.get<ForgetRules>(`${BASE}/forget-rules`),

    updateForgetRules: (rules: ForgetRules) =>
        apiClient.put<ForgetRules>(`${BASE}/forget-rules`, rules),
};
```

### 5.3 `useAdaptiveMemory.ts` — hook đầy đủ

```typescript
// ui/src/hooks/useAdaptiveMemory.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { adaptiveService } from '../services/adaptive.service';

export function useAdaptiveMemories() {
    return useQuery({
        queryKey: ['adaptive', 'memories'],
        queryFn: () => adaptiveService.getMemories(),
        staleTime: 60_000,
    });
}

export function useMemoryVersions(id: string) {
    return useQuery({
        queryKey: ['adaptive', 'memories', id, 'versions'],
        queryFn: () => adaptiveService.getMemoryVersions(id),
        enabled: !!id,
    });
}

export function useConnectors() {
    return useQuery({
        queryKey: ['adaptive', 'connectors'],
        queryFn: () => adaptiveService.getConnectors(),
    });
}

export function useAdaptiveAnalytics() {
    return useQuery({
        queryKey: ['adaptive', 'analytics'],
        queryFn: () => adaptiveService.getAnalytics(),
        refetchInterval: 60_000,
    });
}

export function useForgetRules() {
    return useQuery({
        queryKey: ['adaptive', 'forget-rules'],
        queryFn: () => adaptiveService.getForgetRules(),
    });
}

export function useSyncConnector() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => adaptiveService.syncConnector(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: ['adaptive', 'connectors'] }),
    });
}

export function useUpdateForgetRules() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: adaptiveService.updateForgetRules,
        onSuccess: () => qc.invalidateQueries({ queryKey: ['adaptive', 'forget-rules'] }),
    });
}

export function useCreateConnector() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: adaptiveService.createConnector,
        onSuccess: () => qc.invalidateQueries({ queryKey: ['adaptive', 'connectors'] }),
    });
}
```

---

## 6. Gap CR-006 — `profile.service.ts` & hooks đầy đủ

### 6.1 `profile.service.ts` — đầy đủ

```typescript
// ui/src/services/profile.service.ts
import { apiClient } from '../lib/api-client';
import type { UserProfile, BufferZone, ContextAssembly, UserEvent, ProfileConfig } from '../types/profile';

const BASE = '/v1/console/profiles';

export const profileService = {
    listProfiles: () =>
        apiClient.get<UserProfile[]>(`${BASE}`),

    getProfile: (userId: string) =>
        apiClient.get<UserProfile>(`${BASE}/${userId}`),

    getBuffers: (userId: string) =>
        apiClient.get<BufferZone>(`${BASE}/${userId}/buffers`),

    getContext: (userId: string) =>
        apiClient.get<ContextAssembly>(`${BASE}/${userId}/context`),

    getEvents: (userId: string) =>
        apiClient.get<UserEvent[]>(`${BASE}/${userId}/events`),

    getConfig: () =>
        apiClient.get<ProfileConfig>(`${BASE}/config`),

    updateConfig: (config: Partial<ProfileConfig>) =>
        apiClient.put<ProfileConfig>(`${BASE}/config`, config),
};
```

### 6.2 `useProfiles.ts` — hooks đầy đủ

```typescript
// ui/src/hooks/useProfiles.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { profileService } from '../services/profile.service';

export function useProfileList() {
    return useQuery({
        queryKey: ['profiles'],
        queryFn: () => profileService.listProfiles(),
    });
}

export function useProfileDetail(userId: string) {
    return useQuery({
        queryKey: ['profiles', userId],
        queryFn: () => profileService.getProfile(userId),
        enabled: !!userId,
    });
}

export function useBufferStatus(userId: string) {
    return useQuery({
        queryKey: ['profiles', userId, 'buffers'],
        queryFn: () => profileService.getBuffers(userId),
        enabled: !!userId,
        refetchInterval: 30_000,
    });
}

export function useContextAssembly(userId: string) {
    return useQuery({
        queryKey: ['profiles', userId, 'context'],
        queryFn: () => profileService.getContext(userId),
        enabled: !!userId,
    });
}

export function useUserEvents(userId: string) {
    return useQuery({
        queryKey: ['profiles', userId, 'events'],
        queryFn: () => profileService.getEvents(userId),
        enabled: !!userId,
    });
}

export function useProfileConfig() {
    return useQuery({
        queryKey: ['profiles', 'config'],
        queryFn: () => profileService.getConfig(),
    });
}

export function useUpdateProfileConfig() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: profileService.updateConfig,
        onSuccess: () => qc.invalidateQueries({ queryKey: ['profiles', 'config'] }),
    });
}
```

---

## 7. Gap CR-007 — `governance.service.ts`, mutation hooks & GDPR response schema

### 7.1 GDPR Preview response schema

```go
// GET /v1/console/governance/gdpr/forget/preview
type GDPRPreviewResponse struct {
    UserID          string           `json:"user_id"`
    EstimatedItems  int              `json:"estimated_items"`
    BreakdownByEngine map[string]int `json:"breakdown_by_engine"`
    // e.g. {"memobase": 42, "graphiti": 15, "zep": 8, "sm": 23}
    Warnings        []string         `json:"warnings"`
}
```

### 7.2 `governance.service.ts` — đầy đủ

```typescript
// ui/src/services/governance.service.ts
import { apiClient } from '../lib/api-client';
import type { Tenant, Policy, AuditLogEntry, GDPRPreviewResponse } from '../types/governance';

const BASE = '/v1/console/governance';

export const governanceService = {
    // Tenants
    getTenants: () =>
        apiClient.get<Tenant[]>(`${BASE}/tenants`),
    createTenant: (data: Partial<Tenant>) =>
        apiClient.post<Tenant>(`${BASE}/tenants`, data),
    updateTenant: (id: string, data: Partial<Tenant>) =>
        apiClient.put<Tenant>(`${BASE}/tenants/${id}`, data),

    // Policies
    getPolicies: () =>
        apiClient.get<Policy[]>(`${BASE}/policies`),
    createPolicy: (data: Partial<Policy>) =>
        apiClient.post<Policy>(`${BASE}/policies`, data),
    updatePolicy: (id: string, data: Partial<Policy>) =>
        apiClient.put<Policy>(`${BASE}/policies/${id}`, data),

    // Audit
    getAuditLogs: (filters: Record<string, string>) => {
        const qs = new URLSearchParams(filters).toString();
        return apiClient.get<AuditLogEntry[]>(`${BASE}/audit?${qs}`);
    },

    // GDPR
    previewForget: (userId: string) =>
        apiClient.post<GDPRPreviewResponse>(`${BASE}/gdpr/forget/preview`, { user_id: userId }),
    executeForget: (userId: string) =>
        apiClient.post<{ success: boolean; deleted_count: number }>(`${BASE}/gdpr/forget`, { user_id: userId }),
};
```

### 7.3 Mutation hooks trong `useGovernance.ts`

```typescript
export function useCreateTenant() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: governanceService.createTenant,
        onSuccess: () => qc.invalidateQueries({ queryKey: ['governance', 'tenants'] }),
    });
}

export function useCreatePolicy() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: governanceService.createPolicy,
        onSuccess: () => qc.invalidateQueries({ queryKey: ['governance', 'policies'] }),
    });
}

export function useGDPRForget() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: governanceService.executeForget,
        onSuccess: () => qc.invalidateQueries({ queryKey: ['governance', 'audit'] }),
    });
}

export function useGDPRPreview() {
    return useMutation({
        mutationFn: governanceService.previewForget,
    });
}
```

---

## 8. Gap CR-008 — `MetricPoint[]` schema, hooks & `observability.service.ts`

### 8.1 `MetricPoint[]` response schema

```go
// GET /v1/console/observability/metrics
type MetricsResponse struct {
    Latency    []MetricPoint `json:"latency"`    // p50, p95 over time
    ErrorRate  []MetricPoint `json:"error_rate"` // errors/s
    Throughput []MetricPoint `json:"throughput"` // requests/s
}

type MetricPoint struct {
    Timestamp string  `json:"timestamp"`
    Value     float64 `json:"value"`
    Label     string  `json:"label"`  // "p50", "p95", "error_rate", etc.
}
```

Prometheus query:
```
# Latency p50
histogram_quantile(0.50, sum(rate(vnp_recall_latency_ms_bucket[5m])) by (le))
# Error rate
rate(vnp_errors_total[5m])
# Throughput
rate(vnp_requests_total[5m])
```

### 8.2 Trace query filters

`GET /v1/console/observability/traces` query params:
| Param | Type | Mô tả |
|---|---|---|
| `service` | string | Filter theo service name |
| `status` | string | `ok`, `slow`, `error` |
| `operation` | string | Filter theo operation name |
| `from` | string | ISO timestamp start |
| `to` | string | ISO timestamp end |
| `limit` | int | Default 50 |

### 8.3 `observability.service.ts` — đầy đủ

```typescript
// ui/src/services/observability.service.ts
import { apiClient } from '../lib/api-client';
import type { MetricsResponse, TraceSpan, ErrorEntry, CostEntry } from '../types/observability';

const BASE = '/v1/console/observability';

export const observabilityService = {
    getMetrics: () =>
        apiClient.get<MetricsResponse>(`${BASE}/metrics`),

    getTraces: (filters: Record<string, string> = {}) => {
        const qs = new URLSearchParams(filters).toString();
        return apiClient.get<TraceSpan[]>(`${BASE}/traces?${qs}`);
    },

    getTraceDetail: (id: string) =>
        apiClient.get<TraceSpan>(`${BASE}/traces/${id}`),

    getErrors: (filters: Record<string, string> = {}) => {
        const qs = new URLSearchParams(filters).toString();
        return apiClient.get<ErrorEntry[]>(`${BASE}/errors?${qs}`);
    },

    getCosts: () =>
        apiClient.get<CostEntry[]>(`${BASE}/costs`),
};
```

### 8.4 `useObservability.ts` — hooks đầy đủ

```typescript
export function useObsMetrics() {
    return useQuery({
        queryKey: ['observability', 'metrics'],
        queryFn: () => observabilityService.getMetrics(),
        refetchInterval: 60_000,
    });
}

export function useTraces(filters: Record<string, string> = {}) {
    return useQuery({
        queryKey: ['observability', 'traces', filters],
        queryFn: () => observabilityService.getTraces(filters),
    });
}

export function useTraceDetail(id: string) {
    return useQuery({
        queryKey: ['observability', 'traces', id],
        queryFn: () => observabilityService.getTraceDetail(id),
        enabled: !!id,
    });
}

export function useErrors(filters: Record<string, string> = {}) {
    return useQuery({
        queryKey: ['observability', 'errors', filters],
        queryFn: () => observabilityService.getErrors(filters),
    });
}

export function useCosts() {
    return useQuery({
        queryKey: ['observability', 'costs'],
        queryFn: () => observabilityService.getCosts(),
    });
}
```

---

## 9. Gap CR-009 — `pipeline.service.ts`, `progress` field & hooks

### 9.1 `progress` field — cách tính

```go
// progress (0-100) tính từ pipeline-service job tracking:
// Cognee Cognify: items_processed / items_total * 100
// Memobase Flush: blobs_flushed / blobs_total * 100
// OpenViking Ingest: chunks_indexed / chunks_total * 100

type PipelineJob struct {
    ID            string    `json:"id"`
    Engine        string    `json:"engine"`
    Type          string    `json:"type"`     // ingest|index|sync|cognify
    Status        string    `json:"status"`   // Running|Idle|Failed|Completed
    Progress      int       `json:"progress"` // 0-100
    ItemsTotal    int       `json:"items_total"`
    ItemsDone     int       `json:"items_done"`
    CreatedAt     string    `json:"created_at"`
    UpdatedAt     string    `json:"updated_at"`
}
```

### 9.2 `pipeline.service.ts` — đầy đủ

```typescript
// ui/src/services/pipeline.service.ts
import { apiClient } from '../lib/api-client';
import type { QueueMetrics, PipelineJob, PipelineWorker, PipelineTemplate, PipelineStatus } from '../types/pipeline';

const BASE = '/v1/console/pipelines';

export const pipelineService = {
    getQueues: () =>
        apiClient.get<QueueMetrics>(`${BASE}/queues`),

    getStatus: () =>
        apiClient.get<PipelineStatus[]>(`${BASE}/status`),

    getWorkers: () =>
        apiClient.get<PipelineWorker[]>(`${BASE}/workers`),

    getTemplates: () =>
        apiClient.get<PipelineTemplate[]>(`${BASE}/templates`),

    getJobs: (engine: string) =>
        apiClient.get<PipelineJob[]>(`${BASE}/${engine}/jobs`),

    getJobDetail: (engine: string, jobId: string) =>
        apiClient.get<PipelineJob>(`${BASE}/${engine}/jobs/${jobId}`),
};
```

### 9.3 `usePipelines.ts` — hooks đầy đủ

```typescript
// ui/src/hooks/usePipelines.ts
export function useQueueMetrics() {
    return useQuery({
        queryKey: ['pipelines', 'queues'],
        queryFn: () => pipelineService.getQueues(),
        refetchInterval: 10_000,
    });
}

export function usePipelineStatus() {
    return useQuery({
        queryKey: ['pipelines', 'status'],
        queryFn: () => pipelineService.getStatus(),
        refetchInterval: 10_000,
    });
}

export function useWorkers() {
    return useQuery({
        queryKey: ['pipelines', 'workers'],
        queryFn: () => pipelineService.getWorkers(),
        refetchInterval: 15_000,
    });
}

export function useTemplates() {
    return useQuery({
        queryKey: ['pipelines', 'templates'],
        queryFn: () => pipelineService.getTemplates(),
    });
}

export function useEngineJobs(engine: string) {
    return useQuery({
        queryKey: ['pipelines', engine, 'jobs'],
        queryFn: () => pipelineService.getJobs(engine),
        enabled: !!engine,
        refetchInterval: 10_000,
    });
}

export function useJobDetail(engine: string, jobId: string) {
    return useQuery({
        queryKey: ['pipelines', engine, 'jobs', jobId],
        queryFn: () => pipelineService.getJobDetail(engine, jobId),
        enabled: !!engine && !!jobId,
        refetchInterval: 5_000,  // Refresh nhanh hơn khi đang xem job
    });
}
```

---

## 10. Gap CR-010 — `infrastructure.service.ts` đầy đủ

```typescript
// ui/src/services/infrastructure.service.ts
import { apiClient } from '../lib/api-client';
import type { ServiceInfo, DatabaseHealth, ResourceMetrics, InfraTopology, DeploymentInfo } from '../types/infrastructure';

const BASE = '/v1/console/infra';

export const infrastructureService = {
    getTopology: () =>
        apiClient.get<InfraTopology>(`${BASE}/topology`),

    getServiceHealth: () =>
        apiClient.get<ServiceInfo[]>(`${BASE}/services`),

    getServiceDetail: (name: string) =>
        apiClient.get<ServiceInfo>(`${BASE}/services/${name}`),

    getDatabaseHealth: () =>
        apiClient.get<DatabaseHealth[]>(`${BASE}/databases`),

    getResourceMetrics: () =>
        apiClient.get<ResourceMetrics[]>(`${BASE}/resources`),

    getDeployments: () =>
        apiClient.get<DeploymentInfo[]>(`${BASE}/deployments`),
};
```

---

## 11. Gap CR-011 — Rate limits schema, refactored hook code

### 11.1 Rate limits response schema

```go
// GET /v1/console/sdk/rate-limits
type RateLimitConfig struct {
    Scope    string `json:"scope"`   // "global" | "per_key" | "per_endpoint"
    RPS      int    `json:"rps"`     // requests per second
    RPM      int    `json:"rpm"`     // requests per minute
    Burst    int    `json:"burst"`   // burst capacity
    TierName string `json:"tier_name"` // "free" | "pro" | "enterprise"
}
```

### 11.2 `useOrganizationSettings.ts` — refactored

```typescript
// ui/src/hooks/useOrganizationSettings.ts
// TRƯỚC: có const mockSettings = {...}; const mockMembers = [...]; ...
// SAU — hoàn toàn sạch:
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { orgService } from '../services/org.service';

export function useOrgSettings() {
    return useQuery({
        queryKey: ['org', 'settings'],
        queryFn: () => orgService.getSettings(),
    });
}

export function useOrgMembers() {
    return useQuery({
        queryKey: ['org', 'members'],
        queryFn: () => orgService.getMembers(),
    });
}

export function useOrgRoles() {
    return useQuery({
        queryKey: ['org', 'roles'],
        queryFn: () => orgService.getRoles(),
    });
}

export function useUpdateOrgSettings() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: orgService.updateSettings,
        onSuccess: () => qc.invalidateQueries({ queryKey: ['org', 'settings'] }),
    });
}
```

### 11.3 `useApiSdk.ts` — refactored

```typescript
// ui/src/hooks/useApiSdk.ts
// TRƯỚC: const mockApiKeys = [...]; const mockRateLimits = [...]; ...
// SAU — hoàn toàn sạch:
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { sdkService } from '../services/sdk.service';

export function useApiKeys() {
    return useQuery({
        queryKey: ['sdk', 'keys'],
        queryFn: () => sdkService.getKeys(),
    });
}

export function useRateLimits() {
    return useQuery({
        queryKey: ['sdk', 'rate-limits'],
        queryFn: () => sdkService.getRateLimits(),
    });
}

export function useWebhooks() {
    return useQuery({
        queryKey: ['sdk', 'webhooks'],
        queryFn: () => sdkService.getWebhooks(),
    });
}

export function useCreateApiKey() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: sdkService.createKey,
        onSuccess: () => qc.invalidateQueries({ queryKey: ['sdk', 'keys'] }),
    });
}

export function useCreateWebhook() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: sdkService.createWebhook,
        onSuccess: () => qc.invalidateQueries({ queryKey: ['sdk', 'webhooks'] }),
    });
}
```

### 11.4 `sdk.service.ts` — đầy đủ với create methods

```typescript
// ui/src/services/sdk.service.ts
import { apiClient } from '../lib/api-client';
import type { APIKey, RateLimitConfig, Webhook } from '../types/sdk';

const BASE = '/v1/console/sdk';

export const sdkService = {
    getKeys: () =>
        apiClient.get<APIKey[]>(`${BASE}/keys`),

    createKey: (data: { name: string; permissions: string[]; expires_in_days?: number }) =>
        apiClient.post<{ key: APIKey; raw_key: string }>(`${BASE}/keys`, data),
    // raw_key chỉ trả về 1 lần duy nhất khi tạo — không lưu plain text

    revokeKey: (id: string) =>
        apiClient.delete<void>(`${BASE}/keys/${id}`),

    getRateLimits: () =>
        apiClient.get<RateLimitConfig[]>(`${BASE}/rate-limits`),

    getWebhooks: () =>
        apiClient.get<Webhook[]>(`${BASE}/webhooks`),

    createWebhook: (data: { url: string; events: string[]; secret?: string }) =>
        apiClient.post<Webhook>(`${BASE}/webhooks`, data),

    deleteWebhook: (id: string) =>
        apiClient.delete<void>(`${BASE}/webhooks/${id}`),
};
```

### 11.5 `org.service.ts` — đầy đủ với updateSettings

```typescript
// ui/src/services/org.service.ts
import { apiClient } from '../lib/api-client';
import type { OrgSettings, OrgMember, OrgRole } from '../types/org';

const BASE = '/v1/console/org';

export const orgService = {
    getSettings: () =>
        apiClient.get<OrgSettings>(`${BASE}/settings`),

    updateSettings: (data: Partial<OrgSettings>) =>
        apiClient.put<OrgSettings>(`${BASE}/settings`, data),

    getMembers: () =>
        apiClient.get<OrgMember[]>(`${BASE}/members`),

    getRoles: () =>
        apiClient.get<OrgRole[]>(`${BASE}/roles`),
};
```

---

## 12. Tóm tắt gaps đã xử lý

| Gap | CR | Đã giải quyết ở |
|---|---|---|
| `useStore.ts` sync `UserProfile` + `tenant_id` | CR-001 | SOL-002 §4.1 |
| `api.config.ts` auth namespace | CR-001 | SOL-002 §4.2 |
| `contextSavingsPct` công thức tính | CR-002 | SOL-007 §2.1 |
| `memoryVersions` nguồn dữ liệu | CR-002 | SOL-007 §2.2 |
| `memory_sources` mapping trong messages | CR-003 | SOL-007 §3.1 |
| `/sessions/{id}/user-summary` endpoint | CR-003 | SOL-007 §3.2 |
| Empty state khi search 0 kết quả | CR-004 | SOL-007 §4 |
| `AdaptiveAnalytics` 5 fields đầy đủ | CR-005 | SOL-007 §5.1 |
| `adaptive.service.ts` đầy đủ | CR-005 | SOL-007 §5.2 |
| `useAdaptiveMemory.ts` hooks đầy đủ | CR-005 | SOL-007 §5.3 |
| `profile.service.ts` đầy đủ | CR-006 | SOL-007 §6.1 |
| `useProfiles.ts` hooks đầy đủ | CR-006 | SOL-007 §6.2 |
| GDPR Preview response schema | CR-007 | SOL-007 §7.1 |
| `governance.service.ts` đầy đủ | CR-007 | SOL-007 §7.2 |
| Mutation hooks create/update tenant, policy | CR-007 | SOL-007 §7.3 |
| `MetricPoint[]` response schema | CR-008 | SOL-007 §8.1 |
| Trace query filter params | CR-008 | SOL-007 §8.2 |
| `observability.service.ts` đầy đủ | CR-008 | SOL-007 §8.3 |
| `useObservability.ts` hooks đầy đủ | CR-008 | SOL-007 §8.4 |
| `progress` field tính như thế nào | CR-009 | SOL-007 §9.1 |
| `pipeline.service.ts` đầy đủ | CR-009 | SOL-007 §9.2 |
| `usePipelines.ts` hooks đầy đủ | CR-009 | SOL-007 §9.3 |
| `infrastructure.service.ts` đầy đủ | CR-010 | SOL-007 §10 |
| Rate limits response schema | CR-011 | SOL-007 §11.1 |
| `useOrganizationSettings.ts` refactored | CR-011 | SOL-007 §11.2 |
| `useApiSdk.ts` refactored | CR-011 | SOL-007 §11.3 |
| `sdk.service.ts` + revokeKey + deleteWebhook | CR-011 | SOL-007 §11.4 |
| `org.service.ts` + updateSettings | CR-011 | SOL-007 §11.5 |
