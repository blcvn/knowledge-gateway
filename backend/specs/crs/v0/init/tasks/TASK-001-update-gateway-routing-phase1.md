---
id: TASK-001
title: Update Gateway Routing for Phase 1 Consolidation
service: vnp-gateway
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Update vnp-gateway route configuration to point admin/event/auth services at the new `vnp-platform:9050` target instead of individual services.

## Scope

### In Scope
- Update gateway route table: vnp-admin → vnp-platform:9050
- Update gateway route table: vnp-event → vnp-platform:9050
- Update gateway route table: ov-admin → vnp-platform:9050
- Update gateway route table: zep-admin → vnp-platform:9050
- Update gateway route table: sm-auth → vnp-platform:9050
- Update gateway route table: sm-analytics → vnp-platform:9050
- Update gateway route table: sm-project → vnp-platform:9050
- Register sm-mcp tools in gateway MCP server

### Out of Scope
- Proto definition changes (not needed)
- Client-side changes (not needed)

## Thiết Kế Kỹ Thuật

### Gateway Route Changes

```yaml
# Before
routes:
  - prefix: /vnp.admin.v1.VnpAdminService  → vnp-admin:9041
  - prefix: /vnp.event.v1.VnpEventService  → vnp-event:9043
  - prefix: /ov.admin.v1.OvAdminService    → ov-admin:9056
  - prefix: /zep.admin.v1.ZepAdminService  → zep-admin:9066
  - prefix: /sm.auth.v1.SmAuthService      → sm-auth:9076
  - prefix: /sm.analytics.v1.*             → sm-analytics:9077
  - prefix: /sm.project.v1.*              → sm-project:9078

# After
routes:
  - prefix: /vnp.admin.v1.VnpAdminService  → vnp-platform:9050
  - prefix: /vnp.event.v1.VnpEventService  → vnp-platform:9050
  - prefix: /ov.admin.v1.OvAdminService    → vnp-platform:9050
  - prefix: /zep.admin.v1.ZepAdminService  → vnp-platform:9050
  - prefix: /sm.auth.v1.SmAuthService      → vnp-platform:9050
  - prefix: /sm.analytics.v1.*             → vnp-platform:9050
  - prefix: /sm.project.v1.*              → vnp-platform:9050
```

### Feature Flag

```yaml
# Support gradual rollout
consolidation:
  platform_unified: true  # When false, routes go to original services
```

## Acceptance Criteria

- [ ] AC-1: Given `platform_unified=true`, all admin/event/auth gRPC calls route to vnp-platform:9050
- [ ] AC-2: Given `platform_unified=false`, routes go to original service targets (rollback)
- [ ] AC-3: MCP tools from sm-mcp registered and functional on :8082
- [ ] AC-4: Gateway health check includes vnp-platform cascade check
