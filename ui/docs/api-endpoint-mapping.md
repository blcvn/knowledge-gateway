# VNP Memory Console UI — API Endpoint Mapping

> Version: 1.0.0 | Created: 2026-05-13 | Source: `docs/product/ux_spec.md` Section 11

Tài liệu này map tất cả API endpoints cần thiết cho UI với các module, service file, và hook tương ứng.

---

## 1. Gateway Admin APIs

| Endpoint | Method | UI Module | Service File | Hook |
|---|---|---|---|---|
| `/v1/admin/metrics` | GET | Dashboard | `dashboard.service.ts` | `useMetrics()` |
| `/v1/admin/health` | GET | Dashboard, Infrastructure | `dashboard.service.ts` | `useEngineHealth()` |
| `/v1/admin/throughput` | GET | Dashboard | `dashboard.service.ts` | `useThroughput()` |
| `/v1/admin/tenants` | GET | Governance | `governance.service.ts` | `useTenants()` |
| `/v1/admin/policies` | POST | Governance | `governance.service.ts` | `useCreatePolicy()` |
| `/v1/admin/audit` | GET | Governance | `governance.service.ts` | `useAuditLogs()` |

## 2. Memory APIs (Gateway)

| Endpoint | Method | UI Module | Service File | Hook |
|---|---|---|---|---|
| `/v1/memory/search` | GET | Memory Explorer | `memory.service.ts` | `useMemorySearch()` |
| `/v1/memory/{id}` | GET | Memory Explorer | `memory.service.ts` | `useMemoryDetail()` |
| `/v1/memory/{id}/neighbors` | GET | Memory Explorer | `memory.service.ts` | `useMemoryNeighbors()` |

## 3. Graph APIs (Gateway)

| Endpoint | Method | UI Module | Service File | Hook |
|---|---|---|---|---|
| `/v1/graph/subgraph` | GET | Graph Studio | `graph.service.ts` | `useSubgraph()` |
| `/v1/graph/timeline` | GET | Graph Studio | `graph.service.ts` | `useTimeline()` |
| `/v1/graph/query` | POST | Graph Studio | `graph.service.ts` | `useGraphQuery()` |

## 4. Memobase (Profile) APIs

| Endpoint | Method | UI Module | Service File | Hook |
|---|---|---|---|---|
| `/api/v1/users` | POST | User Profiles | `profile.service.ts` | `useCreateUser()` |
| `/api/v1/users/{user_id}` | GET | User Profiles | `profile.service.ts` | `useUserDetail()` |
| `/api/v1/blobs/insert/{user_id}` | POST | User Profiles | `profile.service.ts` | `useInsertBlob()` |
| `/api/v1/users/profile/{user_id}` | GET | User Profiles | `profile.service.ts` | `useProfileDetail()` |
| `/api/v1/users/profile/{user_id}` | POST | User Profiles | `profile.service.ts` | `useUpdateProfile()` |
| `/api/v1/users/context/{user_id}` | GET | User Profiles | `profile.service.ts` | `useContextAssembly()` |
| `/api/v1/users/event/{user_id}` | GET | User Profiles | `profile.service.ts` | `useUserEvents()` |
| `/api/v1/users/event/search/{user_id}` | GET | User Profiles | `profile.service.ts` | `useEventSearch()` |
| `/api/v1/users/buffer/{user_id}/{buffer_type}` | POST | User Profiles | `profile.service.ts` | `useFlushBuffer()` |
| `/api/v1/users/buffer/capacity/{user_id}/{buffer_type}` | GET | User Profiles | `profile.service.ts` | `useBufferCapacity()` |
| `/api/v1/project/profile_config` | GET | User Profiles | `profile.service.ts` | `useProfileConfig()` |
| `/api/v1/project/profile_config` | POST | User Profiles | `profile.service.ts` | `useUpdateProfileConfig()` |
| `/api/v1/project/billing` | GET | User Profiles | `profile.service.ts` | `useProjectBilling()` |
| `/api/v1/project/usage` | GET | User Profiles | `profile.service.ts` | `useProjectUsage()` |

## 5. Supermemory (Adaptive) APIs

| Endpoint | Method | UI Module | Service File | Hook |
|---|---|---|---|---|
| `/api/v1/documents` | POST | Adaptive Memory | `adaptive.service.ts` | `useCreateDocument()` |
| `/api/v1/memories` | GET | Adaptive Memory | `adaptive.service.ts` | `useAdaptiveMemories()` |
| `/api/v1/memories/{id}/versions` | GET | Adaptive Memory | `adaptive.service.ts` | `useMemoryVersions()` |
| `/api/v1/search` | GET | Adaptive Memory | `adaptive.service.ts` | `useAdaptiveSearch()` |
| `/api/v1/profiles` | GET | Adaptive Memory | `adaptive.service.ts` | `useAdaptiveProfiles()` |
| `/api/v1/connectors` | GET | Adaptive Memory | `adaptive.service.ts` | `useConnectors()` |
| `/api/v1/connectors` | POST | Adaptive Memory | `adaptive.service.ts` | `useCreateConnector()` |
| `/api/v1/analytics` | GET | Adaptive Memory | `adaptive.service.ts` | `useAdaptiveAnalytics()` |
| `/api/v1/projects` | GET | Adaptive Memory | `adaptive.service.ts` | `useAdaptiveProjects()` |

## 6. Cognee APIs

| Endpoint | Method | UI Module | Service File | Hook |
|---|---|---|---|---|
| `/api/v1/cognee/add` | POST | Pipelines | `cognee.service.ts` | `useCogneeAdd()` |
| `/api/v1/cognee/datasets` | GET | Memory Explorer | `cognee.service.ts` | `useCogneeDatasets()` |
| `/api/v1/cognee/cognify` | POST | Pipelines | `cognee.service.ts` | `useCogneeCognify()` |
| `/api/v1/cognee/cognify/{id}/status` | GET | Pipelines | `cognee.service.ts` | `useCognifyStatus()` |
| `/api/v1/cognee/search` | POST | Memory Explorer | `cognee.service.ts` | `useCogneeSearch()` |
| `/api/v1/cognee/search/explore` | GET | Graph Studio | `cognee.service.ts` | `useCogneeExplore()` |
| `/api/v1/cognee/search/rag` | POST | Context Debugger | `cognee.service.ts` | `useCogneeRAG()` |

## 7. Zep APIs

| Endpoint | Method | UI Module | Service File | Hook |
|---|---|---|---|---|
| `/api/v1/users` | POST | Sessions | `zep.service.ts` | `useZepCreateUser()` |
| `/api/v1/threads` | GET | Sessions | `zep.service.ts` | `useZepThreads()` |
| `/api/v1/memories` | POST | Memory Explorer | `zep.service.ts` | `useZepAddMemory()` |
| `/api/v1/graph` | GET | Graph Studio | `zep.service.ts` | `useZepGraph()` |
| `/api/v1/search` | GET | Memory Explorer | `zep.service.ts` | `useZepSearch()` |

---

## Summary Statistics

| Category | Count |
|---|---|
| **Total API Endpoints** | 44 |
| **Service Files** | 12 |
| **Custom Hooks** | ~40+ |
| **UI Modules Affected** | 13 |
| **New Modules** | 2 (User Profiles, Adaptive Memory) |
| **Memobase Endpoints** | 14 |
| **Supermemory Endpoints** | 9 |
| **Cognee Endpoints** | 7 |
| **Zep Endpoints** | 5 |
| **Gateway Endpoints** | 9 |
