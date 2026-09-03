# v3/platform Task Planning — README

**Domain:** Platform Infrastructure (Auth, Rate Limit, SSO, OTel, Webhooks, WebSocket)
**Solutions ref:** `backend/specs/crs/v3/platform/solutions/`
**TDD ref:** `backend/specs/tdd/architecture/01-gateway.md` · `08-platform-services.md` · `09-shared-packages.md`
**Date:** 2026-09-03

---

## Wave Map

| Wave | Scope | Tasks |
|---|---|---|
| **Wave 1** (Foundation) | API Key domain, JWT RS256, DB migrations | TASK-PLAT-001 → TASK-PLAT-004 |
| **Wave 2** (Auth Flows) | Auth use cases, SSO Google OAuth, rate limit middleware | TASK-PLAT-005 → TASK-PLAT-009 |
| **Wave 3** (Observability) | OTel tracing, LLM spans, console observability API | TASK-PLAT-010 → TASK-PLAT-013 |
| **Wave 4** (Events) | Webhook delivery system, WebSocket hub, SSE fallback | TASK-PLAT-014 → TASK-PLAT-020 |

---

## Task List

| Task | Wave | Solution | Component | Est |
|---|---|---|---|---|
| [TASK-PLAT-001](./TASK-PLAT-001-apikey-domain.md) | 1 | SOL-PLAT-001 | `services/vnp-platform/internal/domain/` | 2h |
| [TASK-PLAT-002](./TASK-PLAT-002-apikey-db-migration.md) | 1 | SOL-PLAT-001 | `deployment/dev/migrations/` | 1h |
| [TASK-PLAT-003](./TASK-PLAT-003-jwt-rsa256.md) | 1 | SOL-PLAT-001 | `gateway/internal/infra/middleware/` | 3h |
| [TASK-PLAT-004](./TASK-PLAT-004-devmode-guard.md) | 1 | SOL-PLAT-001 | `gateway/internal/infra/middleware/` | 1h |
| [TASK-PLAT-005](./TASK-PLAT-005-apikey-usecase.md) | 2 | SOL-PLAT-001 | `services/vnp-platform/internal/usecase/` | 4h |
| [TASK-PLAT-006](./TASK-PLAT-006-apikey-handler.md) | 2 | SOL-PLAT-001 | `gateway/adapter/handler/` | 3h |
| [TASK-PLAT-007](./TASK-PLAT-007-ratelimiter-redis.md) | 2 | SOL-PLAT-002 | `shared/pkg/resilience/` | 4h |
| [TASK-PLAT-008](./TASK-PLAT-008-ratelimit-middleware.md) | 2 | SOL-PLAT-002 | `gateway/internal/infra/middleware/` | 2h |
| [TASK-PLAT-009](./TASK-PLAT-009-sso-google-oauth.md) | 2 | SOL-PLAT-003 | `services/vnp-platform/internal/usecase/` | 5h |
| [TASK-PLAT-010](./TASK-PLAT-010-otel-tracer.md) | 3 | SOL-PLAT-004 | `shared/pkg/telemetry/` | 3h |
| [TASK-PLAT-011](./TASK-PLAT-011-tracing-middleware.md) | 3 | SOL-PLAT-004 | `gateway/internal/infra/middleware/` | 2h |
| [TASK-PLAT-012](./TASK-PLAT-012-llm-span-redaction.md) | 3 | SOL-PLAT-004 | `shared/pkg/telemetry/` | 2h |
| [TASK-PLAT-013](./TASK-PLAT-013-observability-handler.md) | 3 | SOL-PLAT-004 | `gateway/adapter/handler/` | 3h |
| [TASK-PLAT-014](./TASK-PLAT-014-webhook-domain-db.md) | 4 | SOL-PLAT-005 | `services/vnp-platform/internal/domain/` | 2h |
| [TASK-PLAT-015](./TASK-PLAT-015-webhook-delivery-service.md) | 4 | SOL-PLAT-005 | `services/vnp-platform/internal/usecase/` | 5h |
| [TASK-PLAT-016](./TASK-PLAT-016-webhook-handler.md) | 4 | SOL-PLAT-005 | `gateway/adapter/handler/` | 3h |
| [TASK-PLAT-017](./TASK-PLAT-017-websocket-hub.md) | 4 | SOL-PLAT-006 | `gateway/adapter/handler/` | 4h |
| [TASK-PLAT-018](./TASK-PLAT-018-websocket-handler.md) | 4 | SOL-PLAT-006 | `gateway/adapter/handler/` | 4h |
| [TASK-PLAT-019](./TASK-PLAT-019-ws-event-buffer.md) | 4 | SOL-PLAT-006 | `gateway/adapter/handler/` | 2h |
| [TASK-PLAT-020](./TASK-PLAT-020-sse-fallback.md) | 4 | SOL-PLAT-006 | `gateway/adapter/handler/` | 2h |

**Total estimate:** ~57h (~7 working days)
