# Tasks — UI API Data Access Layer v0

## Mục đích

Danh sách tác vụ AI-executable để implement **Data Access Layer** (`ui/src/api/`) theo kiến trúc frontend.

---

## Danh sách tác vụ

| Task ID | Solution | Nội dung | Priority | Phụ thuộc |
|---|---|---|---|---|
| [TASK-API-001](./TASK-API-001-http-client.md) | API-SOL-001 | Tạo HTTP client (`http.client.ts` + `queryClient.ts`) | P0 | — |
| [TASK-API-002](./TASK-API-002-types-all.md) | All | Tạo toàn bộ TypeScript type files (`types/*.ts`) | P0 | — |
| [TASK-API-003](./TASK-API-003-auth-client.md) | API-SOL-002 | Tạo `auth.client.ts` + `useAuth.ts` + `AuthContext.tsx` | P0 | TASK-API-001 |
| [TASK-API-004](./TASK-API-004-dashboard-client.md) | API-SOL-003 | Tạo `dashboard.client.ts` + `useDashboard.ts` | P0 | TASK-API-001 |
| [TASK-API-005](./TASK-API-005-sessions-client.md) | API-SOL-004 | Tạo `sessions.client.ts` + `useSessions.ts` | P0 | TASK-API-001 |
| [TASK-API-006](./TASK-API-006-memory-client.md) | API-SOL-005 | Tạo `memory.client.ts` + `useMemory.ts` | P0 | TASK-API-001 |
| [TASK-API-007](./TASK-API-007-adaptive-client.md) | API-SOL-006 | Tạo `adaptive.client.ts` + `useAdaptive.ts` | P1 | TASK-API-001 |
| [TASK-API-008](./TASK-API-008-profiles-client.md) | API-SOL-007 | Tạo `profiles.client.ts` + `useProfiles.ts` | P0 | TASK-API-001 |
| [TASK-API-009](./TASK-API-009-governance-client.md) | API-SOL-008 | Tạo `governance.client.ts` + `useGovernance.ts` | P1 | TASK-API-001 |
| [TASK-API-010](./TASK-API-010-observability-client.md) | API-SOL-009 | Tạo `observability.client.ts` + `useObservability.ts` | P1 | TASK-API-001 |
| [TASK-API-011](./TASK-API-011-pipelines-client.md) | API-SOL-010 | Tạo `pipelines.client.ts` + `usePipelines.ts` | P1 | TASK-API-001 |
| [TASK-API-012](./TASK-API-012-infra-client.md) | API-SOL-011 | Tạo `infra.client.ts` + `useInfrastructure.ts` | P2 | TASK-API-001 |
| [TASK-API-013](./TASK-API-013-org-sdk-client.md) | API-SOL-012 | Tạo `org.client.ts` + `useOrg.ts` | P2 | TASK-API-001 |
| [TASK-API-014](./TASK-API-014-wiring.md) | All | Wiring: app provider setup, migrate old services, xóa mock | P0 | TASK-API-001→013 |

---

## Thứ tự thực hiện

```
Sprint 1 — Foundation:
  TASK-API-001 (http client)
  TASK-API-002 (all types)

Sprint 2 — P0 Auth + Core modules:
  TASK-API-003 (auth)    ← blocker
  TASK-API-004 (dashboard)
  TASK-API-005 (sessions)
  TASK-API-006 (memory)
  TASK-API-008 (profiles)

Sprint 3 — P1 modules:
  TASK-API-007 (adaptive)
  TASK-API-009 (governance)
  TASK-API-010 (observability)
  TASK-API-011 (pipelines)

Sprint 4 — P2 + Wiring:
  TASK-API-012 (infra)
  TASK-API-013 (org/sdk)
  TASK-API-014 (wiring + cleanup)
```

---

## Cấu trúc thư mục sau khi hoàn thành

```text
ui/src/
├── api/
│   ├── clients/
│   │   ├── http.client.ts          ← TASK-API-001
│   │   ├── auth.client.ts          ← TASK-API-003
│   │   ├── dashboard.client.ts     ← TASK-API-004
│   │   ├── sessions.client.ts      ← TASK-API-005
│   │   ├── memory.client.ts        ← TASK-API-006
│   │   ├── adaptive.client.ts      ← TASK-API-007
│   │   ├── profiles.client.ts      ← TASK-API-008
│   │   ├── governance.client.ts    ← TASK-API-009
│   │   ├── observability.client.ts ← TASK-API-010
│   │   ├── pipelines.client.ts     ← TASK-API-011
│   │   ├── infra.client.ts         ← TASK-API-012
│   │   └── org.client.ts           ← TASK-API-013
│   ├── hooks/
│   │   ├── useAuth.ts              ← TASK-API-003
│   │   ├── useDashboard.ts         ← TASK-API-004
│   │   ├── useSessions.ts          ← TASK-API-005
│   │   ├── useMemory.ts            ← TASK-API-006
│   │   ├── useAdaptive.ts          ← TASK-API-007
│   │   ├── useProfiles.ts          ← TASK-API-008
│   │   ├── useGovernance.ts        ← TASK-API-009
│   │   ├── useObservability.ts     ← TASK-API-010
│   │   ├── usePipelines.ts         ← TASK-API-011
│   │   ├── useInfrastructure.ts    ← TASK-API-012
│   │   └── useOrg.ts               ← TASK-API-013
│   └── queryClient.ts              ← TASK-API-001
├── contexts/
│   └── AuthContext.tsx             ← TASK-API-003
└── types/
    ├── auth.ts                     ← TASK-API-002
    ├── dashboard.ts                ← TASK-API-002
    ├── session.ts                  ← TASK-API-002
    ├── memory.ts                   ← TASK-API-002
    ├── adaptive.ts                 ← TASK-API-002
    ├── profile.ts                  ← TASK-API-002
    ├── governance.ts               ← TASK-API-002
    ├── observability.ts            ← TASK-API-002
    ├── pipeline.ts                 ← TASK-API-002
    ├── infrastructure.ts           ← TASK-API-002
    └── org.ts                      ← TASK-API-002
```
