# UI API Solutions — VNP Memory Console v0

## Mục đích

Tài liệu này định nghĩa các giải pháp triển khai **Data Access Layer** cho frontend, dựa trên:
- Kiến trúc: [`ui/specs/frontend_architecture.md`](../../frontend_architecture.md)
- Change Requests: [`specs/crs/v0/ui/`](../../../../specs/crs/v0/ui/)

Mỗi giải pháp **API-SOL** mô tả đầy đủ TypeScript code cho `services/` và `hooks/` theo cấu trúc thư mục của dự án.

---

## Cấu trúc thư mục đã triển khai

```text
ui/src/
├── lib/
│   └── api-client.ts               ← HTTP client (fetch wrapper + 401 queue + AbortController)
├── config/
│   └── api.config.ts               ← API base URLs + feature flags
├── types/
│   ├── api.ts                       ← Shared: HealthStatus, PaginatedResponse, etc.
│   ├── auth.ts                      ← AuthUser, LoginRequest, LoginResponse
│   ├── dashboard.ts                 ← KPIData, EngineHealth, ThroughputData
│   ├── session.ts                   ← Session, Conversation, Message
│   ├── memory.ts                    ← MemoryItem, MemorySearchResult, ALL_ENGINES
│   ├── adaptive.ts                  ← AdaptiveMemory, ExternalConnector, ForgetRules
│   ├── profile.ts                   ← UserProfile, BufferZone, ContextAssembly
│   ├── governance.ts                ← Tenant, Policy, AuditLogEntry, GDPRPreviewResponse
│   ├── observability.ts             ← MetricPoint, TraceSpan, ErrorEntry, CostEntry
│   ├── pipeline.ts                  ← PipelineJob, QueueMetrics, PipelineStatus
│   ├── infrastructure.ts            ← ServiceInfo, DatabaseHealth, InfraTopology
│   └── org.ts                       ← OrgSettings, APIKey, CreateKeyResponse, Webhook
├── services/
│   ├── auth.service.ts              ← POST /v1/auth/*
│   ├── dashboard.service.ts         ← GET /v1/console/dashboard/*
│   ├── session.service.ts           ← GET /v1/console/sessions/*
│   ├── memory.service.ts            ← POST/GET /v1/console/memory/*
│   ├── adaptive.service.ts          ← GET/POST /v1/console/adaptive/*
│   ├── profile.service.ts           ← GET/PUT /v1/console/profiles/*
│   ├── governance.service.ts        ← GET/POST /v1/console/governance/*
│   ├── observability.service.ts     ← GET /v1/console/observability/*
│   ├── pipeline.service.ts          ← GET/POST /v1/console/pipelines/*
│   ├── infrastructure.service.ts    ← GET /v1/console/infra/*
│   └── org.service.ts               ← GET/POST/PUT/DELETE /v1/console/org/* + /sdk/*
└── hooks/
    ├── useAuth.ts                   ← login, logout, me, register
    ├── useDashboard.ts              ← metrics (60s), health (30s), throughput, heatmap
    ├── useSessions.ts               ← list (pagination), detail, working-memory (5s poll)
    ├── useMemory.ts                 ← search (disabled when empty), detail, versions
    ├── useAdaptiveMemory.ts         ← memories, connectors, analytics (60s), forgetRules
    ├── useProfiles.ts               ← list, detail, buffers (30s), context, events, config
    ├── useGovernance.ts             ← tenants, policies, audit, GDPR 2-step preview/forget
    ├── useObservability.ts          ← metrics (60s), traces, errors, costs
    ├── usePipelines.ts              ← queues (10s), status, jobs (5s)
    ├── useInfrastructure.ts         ← topology, services (30s), databases (30s), resources
    ├── useOrganizationSettings.ts   ← org settings, members, roles + API keys show-once, webhooks
    ├── useApiSdk.ts                 ← (re-export từ useOrganizationSettings)
    └── useGraph.ts                  ← subgraph, timeline, ontology
```

---

## Solutions Index

| Solution | CR | Status | Nội dung | Implemented files |
|---|---|---|---|---|
| [API-SOL-001](./API-SOL-001-http-client.md) | CR-000 | ✅ IMPLEMENTED | HTTP client: fetch wrapper + 401 queue | `lib/api-client.ts` · `config/api.config.ts` |
| [API-SOL-002](./API-SOL-002-auth.md) | CR-001 | ✅ IMPLEMENTED | Auth: login/logout/me/refresh | `services/auth.service.ts` · `hooks/useAuth.ts` |
| [API-SOL-003](./API-SOL-003-dashboard.md) | CR-002 | ✅ IMPLEMENTED | Dashboard: metrics/health/throughput | `services/dashboard.service.ts` · `hooks/useDashboard.ts` |
| [API-SOL-004](./API-SOL-004-sessions.md) | CR-003 | ✅ IMPLEMENTED | Sessions: list/detail/working-memory | `services/session.service.ts` · `hooks/useSessions.ts` |
| [API-SOL-005](./API-SOL-005-memory.md) | CR-004 | ✅ IMPLEMENTED | Memory: search/detail/neighbors/versions | `services/memory.service.ts` · `hooks/useMemory.ts` |
| [API-SOL-006](./API-SOL-006-adaptive.md) | CR-005 | ✅ IMPLEMENTED | Adaptive: memories/connectors/analytics | `services/adaptive.service.ts` · `hooks/useAdaptiveMemory.ts` |
| [API-SOL-007](./API-SOL-007-profiles.md) | CR-006 | ✅ IMPLEMENTED | Profiles: list/buffers/context/events | `services/profile.service.ts` · `hooks/useProfiles.ts` |
| [API-SOL-008](./API-SOL-008-governance.md) | CR-007 | ✅ IMPLEMENTED | Governance: tenants/policies/audit/GDPR | `services/governance.service.ts` · `hooks/useGovernance.ts` |
| [API-SOL-009](./API-SOL-009-observability.md) | CR-008 | ✅ IMPLEMENTED | Observability: metrics/traces/errors/costs | `services/observability.service.ts` · `hooks/useObservability.ts` |
| [API-SOL-010](./API-SOL-010-pipelines.md) | CR-009 | ✅ IMPLEMENTED | Pipelines: queues/jobs/status | `services/pipeline.service.ts` · `hooks/usePipelines.ts` |
| [API-SOL-011](./API-SOL-011-infra.md) | CR-010 | ✅ IMPLEMENTED | Infrastructure: topology/services/databases | `services/infrastructure.service.ts` · `hooks/useInfrastructure.ts` |
| [API-SOL-012](./API-SOL-012-org-sdk.md) | CR-011 | ✅ IMPLEMENTED | Org: settings/members + SDK: API keys/webhooks | `services/org.service.ts` · `hooks/useOrganizationSettings.ts` |

---

## Implementation Summary

### ✅ Migration hoàn tất — 2026-06-17

**Tất cả 12 giải pháp đã được triển khai đầy đủ.**

| Hạng mục | Chi tiết |
|---|---|
| **HTTP Client** | `fetch` wrapper với `AbortController`, 401 auto-refresh queue, header injection |
| **Type Safety** | 13 type files, không có `any`, sử dụng `unknown` và string literal unions |
| **Polling** | Health 30s · Dashboard 60s · Queues 10s · Working-memory 5s |
| **Mock removal** | 0 mock imports còn lại trong `ui/src/` (verified bằng `validate-no-mock.sh`) |
| **Build** | `vite build` — 0 errors, ~2300+ modules |
| **API Key show-once** | `raw_key` từ `POST /sdk/keys` chỉ trả về 1 lần, captured ngay trong `onSuccess` |
| **GDPR 2-step** | Preview (`/gdpr/forget/preview`) → Confirm → Execute (`/gdpr/forget`) |
| **Compound IDs** | Memory IDs dạng `engine:id` được `encodeURIComponent` trước khi vào URL |

### Validation commands

```bash
# Không còn mock references
grep -r "useMockData\|from.*mock" ui/src --include="*.ts" --include="*.tsx" | grep -v ".mock.ts:"

# TypeScript clean
cd ui && npx tsc --noEmit

# Build production bundle
cd ui && npx vite build
```
