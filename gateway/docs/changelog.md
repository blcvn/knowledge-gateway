---
id: DOC-S07
service: vnp-gateway
version: 0.7.0
status: Active
created: 2026-05-09
updated: 2026-05-14
format: Keep a Changelog (keepachangelog.com)
---

# Changelog — vnp-gateway

All notable changes to this service will be documented in this file.

## [Unreleased] — v0.7.0 (SOL-003: UI/Gateway Hardening)

### Changed
- **api.md**: Upgraded to v2.0.0 (Active status), verified 116 HandleFunc routes vs documentation parity
- **api.md**: Added SOL-003 cross-reference, UI client notes (`api.config.ts → console.*` routes)
- **Console namespace verified**: All 70 console endpoints match between `router.go` and `api.md` Section 13

### Verified (SOL-003 Phase 1 — UI)
- UI service layer: 10/10 services aligned to `/v1/console/*` namespace
- Hook contracts: `useProfileList`, `usePolicies`, `useAdaptiveAnalytics` fixed
- Type safety: Session/Observability types extended, mock `any` assertions removed
- ErrorBoundary: Global error boundary wrapping lazy-loaded modules
- Dead code: `Placeholder.tsx` removed
- Build: `vite build` passes (2.15s, 22 chunks, 0 errors)

### Planned (SOL-003 Phase 3 — Downstream)
- TASK-062: vnp-admin audit log + policy CRUD handlers
- TASK-063: vnp-platform pipeline status + infra probe handlers
- TASK-064: vnp-search-hub unified search orchestrator
- TASK-065: vnp-event GDPR cascade forget + timeline handlers

## [Unreleased] — v0.6.0 (SOL-002: UX Console API Upgrade)

### Added
- **Console API namespace** (`/v1/console/*`): 11 handler groups, ~70 new REST endpoints
  for VNP Memory Console UI. All require `admin` role.
  - Dashboard API (FEAT-006): health, metrics, throughput, heatmap
  - Memory Explorer API (FEAT-007): unified search, detail, neighbors, versions
  - Graph Studio API (FEAT-013): subgraph, entity, timeline, ontology CRUD, query
  - User Profiles API (FEAT-008): profile list, events, context, buffers, config
  - Adaptive Memory API (FEAT-009): memories, versions, connectors, analytics, forget-rules
  - Context Debugger API (FEAT-010): trace simulation, saved traces
  - Sessions API (FEAT-014): session list, timeline, diff, working-memory, user-summary
  - Governance API (FEAT-011): tenant/policy CRUD, audit logs, GDPR forget/preview
  - Pipelines API (FEAT-015): status, jobs, queues, workers, templates
  - Infrastructure API (FEAT-016): topology, services, databases, resources
  - Observability API (FEAT-017): metrics, traces, errors, costs

- **WebSocket realtime** (FEAT-012): `WS /v1/console/ws`
  - Channels: engine.health, memory.flow, pipeline.progress, alerts
  - JWT-authenticated, per-connection channel subscription

- **Governance data model**: `audit_logs` + `policies` PostgreSQL tables
  - Indexes for tenant-scoped time-range queries
  - Migration v2.0.0

- **Console configuration**: 11 new env vars
  - `CONSOLE_ADMIN_ROLE`, `WS_MAX_CONNECTIONS`, `DASHBOARD_*_CACHE_TTL`, etc.

- **Console usecases**: DashboardUseCase, SessionUseCase, DebuggerUseCase, GraphUseCase

### Changed
- **api.md**: Added Section 13 — Console APIs (~70 endpoints across 11 namespaces)
- **architecture.md**: Added 11 console handler files + 4 console usecases to layer structure
- **data-model.md**: Added audit_logs, policies tables + dashboard/WS Redis keys
- **configuration.md**: Added Console UI section with WebSocket + dashboard config
- **SOL-002**: Upgraded to v2.0.0 — added 5 missing FEAT specs (013-017), expanded to 21 tasks

### Specs Added
- `FEAT-013-graph-studio-api.md` — UX §6.3 Graph Studio
- `FEAT-014-sessions-api.md` — UX §6.7 Sessions & Conversations
- `FEAT-015-pipelines-console-api.md` — UX §6.9 Pipelines Console
- `FEAT-016-infrastructure-health-api.md` — UX §6.10 Infrastructure View
- `FEAT-017-observability-api.md` — UX §6.11 Observability

## [Unreleased] — v0.5.0

### Changed
- **Merged `services/vnp-gateway` into `gateway/`**: Removed the legacy stub scaffold
  that only contained docs and a basic main.go (86 LOC). All documentation and specs
  now live alongside the production implementation in `gateway/`. Updated all external
  references across the monorepo. Moved `QA-001-align-docs-with-implementation.md`
  to `gateway/specs/quality/`.

## [Unreleased] — v0.4.0

### Added
- **gRPC Proto Definitions**: 8 `.proto` files covering all 6 cognitive engines + unified memory + admin
  - 52 RPCs, 114 message types, 915 lines of proto definitions
  - Services: memory, cognee, graphiti, memobase, openviking, zep, supermemory, admin
  - Buf module config for Go stub generation
  - Proto design: request/response pairs, well-known types (Timestamp, Struct)

### Planned
- Wire code generation (`go generate` with google/wire) — TASK-015
- OTel distributed tracing with Jaeger integration — TASK-016
- E2E tests with Docker Compose downstream services — TASK-017
- `buf generate` → typed gRPC clients (replace raw `[]byte` forwarding)
- LLM-based content classifier via Bifrost (replace keyword heuristic)

## [0.3.0] - 2026-05-09

### Added — Productionization

- **JWT RS256 Authentication**: Full `golang-jwt/jwt/v5` integration
  - VNPClaims with tenant_id, user_id, roles, scopes, rate_tier
  - RS256 signature validation with issuer/audience/leeway checks
  - Auth failure events published to NATS `gateway.auth.failed`
  - 7 unit tests (valid, expired, wrong issuer, dev mode, missing claim, API key)

- **PostgreSQL Store**: `persistence/pg_store.go`
  - `tenants` table with rate_tier, enabled, metadata JSONB
  - `api_keys` table with key_hash, expiry, revocation, last_used tracking
  - Auto-migration via `MigrateSchema()`
  - Async `last_used_at` updates (non-blocking)
  - pgxpool with keepalive, health checks, connection limits

- **NATS JetStream Publisher**: `persistence/nats_publisher.go`
  - Auto-creates `GATEWAY` stream with 7-day retention
  - Subjects: `gateway.>` (request, auth, ratelimit, circuit events)
  - Reconnect with backoff (2s interval, 60 max retries)
  - Drain on shutdown

- **Auth HTTP Middleware**: `middleware/auth.go`
  - Extracts JWT from `Authorization: Bearer` or API Key from `X-API-Key`
  - Injects AuthContext into request context
  - Returns 401 with `WWW-Authenticate: Bearer` on failure
  - Skips auth for `/healthz`, `/readyz`, `/metrics`

- **Rate Limit HTTP Middleware**: `middleware/auth.go`
  - Per-tenant enforcement using AuthContext.RateTier
  - Sets `X-RateLimit-Remaining` response header
  - Returns 429 with `Retry-After: 60` when exceeded
  - Fail-open on Redis errors

- **Production cmd/main.go**: Full infrastructure wiring
  - Graceful degradation: noop fallbacks for unavailable infra
  - 3-server lifecycle: REST (:8080) + MCP (:8082) + Observability (:11080)
  - Background gRPC health checker (30s interval)
  - Ordered cleanup on shutdown

## [0.2.0] - 2026-05-09

### Added — Infrastructure (TASK-007 → TASK-013)

- **gRPC Client Registry** (TASK-007): `client/registry.go`
  - Connection pool with keepalive (10s ping, 3s timeout)
  - Resolve + Forward with tenant metadata propagation
  - Background health checker for all 35 services
  - ServiceStatus() for observability dashboard
  - Graceful connection cleanup on shutdown

- **Rate Limiter** (TASK-008): `persistence/ratelimit_redis.go`
  - Redis sliding window via atomic Lua script
  - 3 tiers: Free (60 RPM), Pro (600 RPM), Enterprise (6000 RPM)
  - Fail-open when Redis unavailable
  - X-RateLimit-* response headers

- **Circuit Breaker** (TASK-009): `client/circuit.go`
  - sony/gobreaker v2 per-service isolation
  - 5 consecutive failures → open → 503 UNAVAILABLE
  - 60s timeout → half-open → 3 probe requests
  - State change logging (WARN level)
  - AllCircuitStates() for monitoring

- **MCP Server** (TASK-010): `mcp/server.go`
  - 16 AI agent tools with JSON Schema input definitions
  - SSE transport (text/event-stream) for long-lived connections
  - HTTP Streamable transport (JSON-RPC 2.0 over POST)
  - Session management with unique IDs
  - initialize / tools/list / tools/call / ping handlers

- **WebDAV Proxy** (TASK-011): `webdav/proxy.go`
  - Reverse proxy to ov-fs for all WebDAV methods
  - Header propagation: Depth, Lock-Token, If, Destination
  - Auth context (X-Tenant-ID) forwarding
  - macOS Finder / Windows Explorer compatible

- **Observability** (TASK-012): `middleware/metrics.go`, `server/observability.go`
  - Prometheus: requests_total, request_duration, active_connections, circuit_breaker_state
  - Health: `/healthz` (liveness), `/readyz` (readiness), `/healthz/deep` (cascade)
  - Metrics: `/metrics` (Prometheus scrape endpoint)
  - Path normalization to prevent label explosion

- **Integration Tests** (TASK-013): `tests/integration/gateway_test.go`
  - 15 test cases: routing (8), auto-routing (3), error format (1), CORS (1), RequestID (2)
  - Mock ServiceRegistry + EventPublisher
  - All tests pass in < 1s (no real network calls)

### Stats
- **24 Go source files** across 13 packages
- **3172 lines** of production Go code + tests
- `go build ./internal/...` ✅
- `go test ./internal/domain/...` ✅ (22 sub-tests)
- `go test ./tests/integration/...` ✅ (15 tests)
- Domain layer: zero external imports ✅

## [0.1.0] - 2026-05-09

### Added — Foundation (TASK-001 → TASK-006)

- **Domain Layer** (TASK-001): `entity.go`, `errors.go`, `event.go`
  - AuthContext, RouteTarget, ProtocolType, StoreRequest, RouteResult
  - 7 sentinel errors mapped to gRPC/HTTP codes
  - 5 NATS event types with JSON tags
  - Zero external dependencies — Go stdlib only

- **Usecase Ports** (TASK-002): `port/input.go`, `port/output.go`
  - 4 input interfaces: Router, Authenticator, MCPHandler, RateLimiter
  - 5 output interfaces: ServiceRegistry, TenantStore, KeyStore, EventPublisher, RateLimitStore

- **Config + Wire** (TASK-003): `config/config.go`
  - Viper-compatible Config struct with 40+ env vars
  - DefaultConfig() with all 35 service addresses
  - Config.Validate() for required fields

- **Auth Middleware** (TASK-004): `usecase/auth.go`, `middleware/middleware.go`
  - JWT RS256 validation skeleton (AuthenticateJWT)
  - API Key resolution (SHA-256 hash → KeyStore)
  - Dev mode bypass (AUTH_DEV_MODE=true)
  - AuthContext injection into request context

- **HTTP Router** (TASK-005): `handler/router.go`, `server/http.go`
  - Go 1.22 stdlib ServeMux with method-based routing
  - Middleware chain: Recovery → RequestID → Logger → CORS
  - Graceful shutdown (30s timeout on SIGTERM)
  - Health server on separate port

- **REST Handlers** (TASK-006): `handler/handler.go`, `handler/services.go`
  - 8 namespace handlers: Memory, Cognee, Graphiti, Memobase, OV, Zep, SM, Admin
  - 50+ route registrations
  - ForwardToService generic handler pattern
  - Auto-routing via RouteUseCase for /v1/memory/store
