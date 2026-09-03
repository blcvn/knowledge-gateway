# VNP Memory Console UI — Change Requests v0

## Mục tiêu

Loại bỏ toàn bộ **mock data cứng** khỏi frontend và kết nối hoàn toàn với **backend API**, đảm bảo mọi dữ liệu hiển thị trên Console đều được lấy từ database thực.

---

## Danh sách Change Requests

| CR | Tên | Module | Ưu tiên | Trạng thái |
|---|---|---|---|---|
| [CR-000](./CR-000-OVERVIEW.md) | Tổng quan Migration | All | P0 | ✅ Implemented |
| [CR-001](./CR-001-AUTH.md) | Authentication API | Auth | P0 🔴 | ✅ Implemented |
| [CR-002](./CR-002-DASHBOARD.md) | Dashboard KPIs & Health | Dashboard | P0 🔴 | ✅ Implemented |
| [CR-003](./CR-003-SESSIONS.md) | Sessions Explorer | Sessions | P0 🔴 | ✅ Implemented |
| [CR-004](./CR-004-MEMORY.md) | Memory Explorer | Memory | P0 🔴 | ✅ Implemented |
| [CR-005](./CR-005-ADAPTIVE.md) | Adaptive Memory | Adaptive | P1 🟠 | ✅ Implemented |
| [CR-006](./CR-006-PROFILES.md) | User Profiles | Profiles | P0 🔴 | ✅ Implemented |
| [CR-007](./CR-007-GOVERNANCE.md) | Governance Center | Governance | P1 🟠 | ✅ Implemented |
| [CR-008](./CR-008-OBSERVABILITY.md) | Observability | Observability | P1 🟠 | ✅ Implemented |
| [CR-009](./CR-009-PIPELINES.md) | Pipelines Monitor | Pipelines | P1 🟠 | ✅ Implemented |
| [CR-010](./CR-010-INFRASTRUCTURE.md) | Infrastructure Health | Infrastructure | P2 🟡 | ✅ Implemented |
| [CR-011](./CR-011-ORG-SDK.md) | Org Settings & API SDK | Org/SDK | P2 🟡 | ✅ Implemented |

---

## Mock Data — ✅ Đã xóa hoàn toàn (TASK-UI-016)

> **Tất cả file mock đã bị xóa** vào ngày 2026-06-17. Thư mục `ui/src/mock/` không còn tồn tại.

| File mock | Được dùng bởi | CR liên quan | Trạng thái |
|---|---|---|---|
| `ui/src/mock/dashboard.mock.ts` | `useDashboard.ts` | CR-002 | ✅ Đã xóa |
| `ui/src/mock/session.mock.ts` | `useSessions.ts` | CR-003 | ✅ Đã xóa |
| `ui/src/mock/memory.mock.ts` | `useMemory.ts` | CR-004 | ✅ Đã xóa |
| `ui/src/mock/adaptive.mock.ts` | `useAdaptiveMemory.ts` | CR-005 | ✅ Đã xóa |
| `ui/src/mock/profile.mock.ts` | `useProfiles.ts` | CR-006 | ✅ Đã xóa |
| `ui/src/mock/governance.mock.ts` | `useGovernance.ts` | CR-007 | ✅ Đã xóa |
| `ui/src/mock/observability.mock.ts` | `useObservability.ts` | CR-008 | ✅ Đã xóa |
| `ui/src/mock/pipeline.mock.ts` | `usePipelines.ts` | CR-009 | ✅ Đã xóa |
| `ui/src/mock/infrastructure.mock.ts` | `useInfrastructure.ts` | CR-010 | ✅ Đã xóa |
| `ui/src/services/auth.ts` (mock logic) | Login page | CR-001 | ✅ Rewritten → real API |
| Inline mock trong `useOrganizationSettings.ts` | Org UI | CR-011 | ✅ Đã xóa |
| Inline mock trong `useApiSdk.ts` | SDK UI | CR-011 | ✅ Đã xóa |

---

## Backend Endpoints cần implement

### Auth (CR-001)
```
POST /v1/auth/login
POST /v1/auth/logout
POST /v1/auth/refresh
GET  /v1/auth/me
```

### Dashboard (CR-002)
```
GET /v1/console/dashboard/metrics
GET /v1/console/dashboard/health
GET /v1/console/dashboard/throughput
GET /v1/console/dashboard/heatmap
```

### Sessions (CR-003)
```
GET /v1/console/sessions
GET /v1/console/sessions/live
GET /v1/console/sessions/{id}
GET /v1/console/sessions/{id}/timeline
GET /v1/console/sessions/{id}/diff
GET /v1/console/sessions/{id}/working-memory
```

### Memory (CR-004)
```
POST /v1/console/memory/search
GET  /v1/console/memory/{id}
GET  /v1/console/memory/{id}/neighbors
GET  /v1/console/memory/{id}/versions
```

### Adaptive (CR-005)
```
GET  /v1/console/adaptive/memories
GET  /v1/console/adaptive/memories/{id}/versions
GET  /v1/console/adaptive/connectors
POST /v1/console/adaptive/connectors
POST /v1/console/adaptive/connectors/{id}/sync
GET  /v1/console/adaptive/analytics
GET  /v1/console/adaptive/forget-rules
PUT  /v1/console/adaptive/forget-rules
```

### Profiles (CR-006)
```
GET /v1/console/profiles
GET /v1/console/profiles/config
PUT /v1/console/profiles/config
GET /v1/console/profiles/{user_id}
GET /v1/console/profiles/{user_id}/events
GET /v1/console/profiles/{user_id}/context
GET /v1/console/profiles/{user_id}/buffers
```

### Governance (CR-007)
```
GET  /v1/console/governance/tenants
POST /v1/console/governance/tenants
PUT  /v1/console/governance/tenants/{id}
GET  /v1/console/governance/policies
POST /v1/console/governance/policies
PUT  /v1/console/governance/policies/{id}
GET  /v1/console/governance/audit
POST /v1/console/governance/gdpr/forget
POST /v1/console/governance/gdpr/forget/preview
```

### Observability (CR-008)
```
GET /v1/console/observability/metrics
GET /v1/console/observability/traces
GET /v1/console/observability/traces/{id}
GET /v1/console/observability/errors
GET /v1/console/observability/costs
```

### Pipelines (CR-009)
```
GET /v1/console/pipelines/status
GET /v1/console/pipelines/queues
GET /v1/console/pipelines/workers
GET /v1/console/pipelines/templates
GET /v1/console/pipelines/{engine}/jobs
GET /v1/console/pipelines/{engine}/jobs/{id}
```

### Infrastructure (CR-010)
```
GET /v1/console/infra/topology
GET /v1/console/infra/services
GET /v1/console/infra/services/{name}
GET /v1/console/infra/databases
GET /v1/console/infra/resources
GET /v1/console/infra/deployments
```

### Org & SDK (CR-011)
```
GET /v1/console/org/settings
PUT /v1/console/org/settings
GET /v1/console/org/members
GET /v1/console/org/roles
GET /v1/console/sdk/keys
POST /v1/console/sdk/keys
GET /v1/console/sdk/rate-limits
GET /v1/console/sdk/webhooks
POST /v1/console/sdk/webhooks
```

---

## Thứ tự thực hiện đề xuất

```
Sprint 1 (P0 - Blocker):
  CR-001 → CR-002 → CR-003 → CR-004 → CR-006

Sprint 2 (P1 - Core):
  CR-005 → CR-007 → CR-008 → CR-009

Sprint 3 (P2 - Ops):
  CR-010 → CR-011
```

---

## Môi trường

| Env var | Giá trị đích (Production) | Trạng thái |
|---|---|---|
| `VITE_API_BASE_URL` | `https://api.vnp-memory.io` | ✅ Configured |
| `VITE_USE_MOCK_DATA` | Đã xóa / comment out | ✅ Removed (TASK-UI-016) |

> **VITE_USE_MOCK_DATA** không còn được đọc từ code — field `useMockData` đã bị xóa khỏi `API_CONFIG`. Tất cả hooks gọi API thực trực tiếp, không còn flag mock nào.
