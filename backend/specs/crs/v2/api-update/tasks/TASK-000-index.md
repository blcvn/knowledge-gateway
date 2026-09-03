# TASK-000: Task Index — API Update v2

**Directory**: `specs/crs/v2/api-update/tasks/`  
**Created**: 2026-06-18  
**Solutions Reference**: [`solutions/`](../solutions/)

---

## Execution Order

Tasks must be executed in the following order due to dependencies:

```
Phase 1 — Fix Compile Errors (Day 1)
  TASK-001 → Create auth.go handler
  TASK-003 → Create console_org.go handler  
  TASK-004 → Create console_sdk.go handler
  TASK-002 → Register all new routes in router.go + auth middleware exemption
  ↑ These 4 tasks fix the current compile errors in gateway.go (lines 40, 59–60, 63, 67)

Phase 2 — Backend Proto (Day 1–2)
  TASK-010 → Add Logout/RefreshToken/GetCurrentUser to sm-auth proto
  ↑ Enables proper auth flow end-to-end

Phase 3 — Domain Model (Day 2)
  TASK-008 → Add Webhook + OrgSettings entities to vnp-platform
  TASK-009 → Add webhooks database migration
  ↑ Required before vnp-admin can serve org/sdk endpoints

Phase 4 — Response Quality (Day 2–3)
  TASK-005 → Add pagination normaliser + update ListSessions
  TASK-006 → Fix error response format (WriteError → flat { message, code, status })
  TASK-007 → Fix HealthStatus enum casing in vnp-platform
```

---

## Task Checklist

| Task | Priority | Estimate | File | Status |
|------|----------|----------|------|--------|
| [TASK-001](./TASK-001-create-auth-handler.md) | 🔴 Critical | 2h | `gateway/.../handler/auth.go` (CREATE) | ❌ Not Started |
| [TASK-002](./TASK-002-register-auth-routes.md) | 🔴 Critical | 1h | `gateway/.../handler/router.go` + `middleware/auth.go` (MODIFY) | ❌ Not Started |
| [TASK-003](./TASK-003-create-console-org-handler.md) | 🔴 Critical | 1h | `gateway/.../handler/console_org.go` (CREATE) | ❌ Not Started |
| [TASK-004](./TASK-004-create-console-sdk-handler.md) | 🔴 Critical | 1h | `gateway/.../handler/console_sdk.go` (CREATE) | ❌ Not Started |
| [TASK-005](./TASK-005-pagination-normaliser.md) | 🟡 High | 2h | `gateway/.../handler/handler.go` + `console.go` (MODIFY) | ❌ Not Started |
| [TASK-006](./TASK-006-error-response-normaliser.md) | 🟠 Medium | 2h | `gateway/.../handler/handler.go` + `router.go` (MODIFY) | 🔄 Partial |
| [TASK-007](./TASK-007-fix-healthstatus-casing.md) | 🟠 Medium | 0.5h | `services/vnp-platform/.../admin/entity.go` (MODIFY) | ❌ Not Started |
| [TASK-008](./TASK-008-add-webhook-orgsettings-entities.md) | 🟡 High | 2h | `services/vnp-platform/.../admin/entity.go` (MODIFY) | ❌ Not Started |
| [TASK-009](./TASK-009-webhooks-migration.md) | 🟡 High | 0.5h | `deployment/dev/migrations/XXXX_add_webhooks.sql` (CREATE) | ❌ Not Started |
| [TASK-010](./TASK-010-smauth-proto-refresh-logout-me.md) | 🔴 Critical | 2h | `services/sm-auth/api/proto/v1/auth.proto` + handler (MODIFY) | ❌ Not Started |

**Total Estimate**: ~14 hours

---

## Build Verification Commands

After completing all tasks, verify the full build:

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory

# Build gateway package
go build ./gateway/...

# Build monolith (includes all bootstrap)
go build ./apps/memory/...

# Build vnp-platform service
go build ./services/vnp-platform/...

# Build sm-auth service
go build ./services/sm-auth/...

# Run all tests
go test ./gateway/... ./services/vnp-platform/... ./services/sm-auth/...
```

---

## Files Changed Summary

| File | Task | Action |
|------|------|--------|
| `gateway/internal/adapter/handler/auth.go` | TASK-001 | CREATE |
| `gateway/internal/adapter/handler/console_org.go` | TASK-003 | CREATE |
| `gateway/internal/adapter/handler/console_sdk.go` | TASK-004 | CREATE |
| `gateway/internal/adapter/handler/router.go` | TASK-002 | MODIFY (+auth/org/sdk params + 15 routes) |
| `gateway/internal/infra/middleware/auth.go` | TASK-002 | MODIFY (skip login/refresh paths) |
| `gateway/internal/adapter/handler/handler.go` | TASK-005, TASK-006 | MODIFY (pagination norm + WriteError fix) |
| `gateway/internal/adapter/handler/console.go` | TASK-005 | MODIFY (ListSessions uses ForwardWithNorm) |
| `services/vnp-platform/internal/domain/admin/entity.go` | TASK-007, TASK-008 | MODIFY |
| `deployment/dev/migrations/XXXX_add_webhooks.sql` | TASK-009 | CREATE |
| `services/sm-auth/api/proto/v1/auth.proto` | TASK-010 | MODIFY |
| `services/sm-auth/api/proto/v1/auth.pb.go` | TASK-010 | REGENERATE |
| `services/sm-auth/api/proto/v1/auth_grpc.pb.go` | TASK-010 | REGENERATE |
| `services/sm-auth/internal/adapter/grpc/auth_handler.go` | TASK-010 | MODIFY |
