# Tasks — VNP Memory Console UI v0

## Tổ chức

Các tác vụ được chia theo **layer** và **module**, mỗi tác vụ đủ nhỏ để AI thực thi độc lập trong 1 lần chạy.

---

## Danh sách Tasks

### Layer: Frontend (TypeScript / React)

| Task ID | Module | Nội dung | Priority | Depends On |
|---|---|---|---|---|
| [TASK-UI-001](./TASK-UI-001-api-config-auth-namespace.md) | Config | Cập nhật `api.config.ts` — thêm auth namespace và tất cả console namespaces | P0 | — |
| [TASK-UI-002](./TASK-UI-002-auth-service.md) | Auth | Rewrite `services/auth.ts` — real API calls, localStorage management | P0 | TASK-UI-001 |
| [TASK-UI-003](./TASK-UI-003-api-client-refresh.md) | Auth | Cập nhật `lib/api-client.ts` — 401 auto-refresh interceptor | P0 | TASK-UI-002 |
| [TASK-UI-004](./TASK-UI-004-usestore-tenant.md) | Auth | Cập nhật `store/useStore.ts` — thêm `tenant_id` vào UserProfile | P0 | TASK-UI-002 |
| [TASK-UI-005](./TASK-UI-005-dashboard-hooks.md) | Dashboard | Refactor `hooks/useDashboard.ts` — xóa mock, thêm refetchInterval | P0 | TASK-UI-001 |
| [TASK-UI-006](./TASK-UI-006-sessions-hooks.md) | Sessions | Refactor `hooks/useSessions.ts` — xóa mock, thêm pagination params | P0 | TASK-UI-001 |
| [TASK-UI-007](./TASK-UI-007-memory-hooks.md) | Memory | Refactor `hooks/useMemory.ts` — xóa mock, thêm `useMemoryNeighbors`, `useMemoryVersions` | P0 | TASK-UI-001 |
| [TASK-UI-008](./TASK-UI-008-memory-empty-state.md) | Memory | Tạo `MemoryEmptyState` component cho search 0 kết quả | P0 | TASK-UI-007 |
| [TASK-UI-009](./TASK-UI-009-adaptive-service-hooks.md) | Adaptive | Tạo `services/adaptive.service.ts` + refactor `hooks/useAdaptiveMemory.ts` | P1 | TASK-UI-001 |
| [TASK-UI-010](./TASK-UI-010-profile-service-hooks.md) | Profiles | Tạo `services/profile.service.ts` + refactor `hooks/useProfiles.ts` | P0 | TASK-UI-001 |
| [TASK-UI-011](./TASK-UI-011-governance-service-hooks.md) | Governance | Tạo `services/governance.service.ts` + refactor `hooks/useGovernance.ts` | P1 | TASK-UI-001 |
| [TASK-UI-012](./TASK-UI-012-observability-service-hooks.md) | Observability | Tạo `services/observability.service.ts` + refactor `hooks/useObservability.ts` | P1 | TASK-UI-001 |
| [TASK-UI-013](./TASK-UI-013-pipeline-service-hooks.md) | Pipelines | Tạo `services/pipeline.service.ts` + refactor `hooks/usePipelines.ts` | P1 | TASK-UI-001 |
| [TASK-UI-014](./TASK-UI-014-infra-service.md) | Infrastructure | Tạo `services/infrastructure.service.ts` + refactor `hooks/useInfrastructure.ts` | P2 | TASK-UI-001 |
| [TASK-UI-015](./TASK-UI-015-org-sdk-service-hooks.md) | Org & SDK | Tạo `services/org.service.ts` + `services/sdk.service.ts` + refactor hooks | P2 | TASK-UI-001 |
| [TASK-UI-016](./TASK-UI-016-remove-mock-files.md) | All | Xóa tất cả file `src/mock/*.ts` + xóa `useMock` ternary khỏi hooks | P0 | TASK-UI-005→015 |

### Layer: Backend (Go)

| Task ID | Module | Nội dung | Priority | Depends On |
|---|---|---|---|---|
| [TASK-BE-001](./TASK-BE-001-auth-db-schema.md) | Auth | PostgreSQL migrations: `users`, `refresh_tokens` tables | P0 | — |
| [TASK-BE-002](./TASK-BE-002-auth-handler.md) | Auth | Gateway: `auth_handler.go` — login/logout/refresh/me endpoints | P0 | TASK-BE-001 |
| [TASK-BE-003](./TASK-BE-003-dashboard-handler.md) | Dashboard | Gateway: `console_dashboard_handler.go` — metrics/health/throughput/heatmap | P0 | — |
| [TASK-BE-004](./TASK-BE-004-sessions-db-schema.md) | Sessions | PostgreSQL: `sessions` + `messages` tables + FTS index | P0 | — |
| [TASK-BE-005](./TASK-BE-005-sessions-handler.md) | Sessions | Gateway: `console_sessions_handler.go` — list/detail/timeline/diff/user-summary | P0 | TASK-BE-004 |
| [TASK-BE-006](./TASK-BE-006-memory-handler.md) | Memory | Gateway: `console_memory_handler.go` — search/get/neighbors/versions | P0 | — |
| [TASK-BE-007](./TASK-BE-007-adaptive-handler.md) | Adaptive | Gateway: `console_adaptive_handler.go` — memories/connectors/analytics/rules | P1 | — |
| [TASK-BE-008](./TASK-BE-008-profiles-handler.md) | Profiles | Gateway: `console_profiles_handler.go` — list/detail/buffers/context/events | P0 | — |
| [TASK-BE-009](./TASK-BE-009-governance-handler.md) | Governance | Gateway: `console_governance_handler.go` — tenants/policies/audit/gdpr + audit_logs migration | P1 | — |
| [TASK-BE-010](./TASK-BE-010-observability-handler.md) | Observability | Gateway: `console_observability_handler.go` — metrics/traces/errors/costs | P1 | — |
| [TASK-BE-011](./TASK-BE-011-pipelines-handler.md) | Pipelines | Gateway: `console_pipelines_handler.go` — queues/status/workers/jobs | P1 | — |
| [TASK-BE-012](./TASK-BE-012-infra-handler.md) | Infrastructure | Gateway: `console_infra_handler.go` — services/databases/resources/topology | P2 | — |
| [TASK-BE-013](./TASK-BE-013-org-sdk-handler.md) | Org & SDK | Gateway: `console_org_handler.go` + `console_sdk_handler.go` + webhooks migration | P2 | TASK-BE-001 |

---

## Thứ tự thực hiện

```
Sprint 1 — Auth foundation (unblock mọi thứ):
  TASK-BE-001 → TASK-BE-002
  TASK-UI-001 → TASK-UI-002 → TASK-UI-003 → TASK-UI-004

Sprint 2 — Core P0 modules:
  BE: TASK-BE-003, TASK-BE-004 → TASK-BE-005, TASK-BE-006, TASK-BE-008
  UI: TASK-UI-005, TASK-UI-006, TASK-UI-007 → TASK-UI-008, TASK-UI-010

Sprint 3 — P1 modules:
  BE: TASK-BE-007, TASK-BE-009, TASK-BE-010, TASK-BE-011
  UI: TASK-UI-009, TASK-UI-011, TASK-UI-012, TASK-UI-013

Sprint 4 — P2 + Cleanup:
  BE: TASK-BE-012, TASK-BE-013
  UI: TASK-UI-014, TASK-UI-015 → TASK-UI-016
```
