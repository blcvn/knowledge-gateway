# Proposal: Standardize KG Service Runtime on the go-kratos Platform with Clean Architecture

## Problem

The current runtime is only Kratos-like in spirit. It still relies on a custom bootstrap shell,
manual lifecycle handling, and ad hoc logging/telemetry wiring instead of the official
`go-kratos/kratos` application, transport, logging, middleware, and observability primitives.

That creates three problems:

1. the runtime boundary is not anchored on the actual Kratos app and server types;
2. logs, trace context, and telemetry are not emitted through the official Kratos middleware stack;
3. package boundaries are blurred, which makes clean-code ownership and dependency direction harder
   to maintain.

Without a tracked change, future work in `cmd/` and `internal/` will keep drifting toward local
runtime patterns instead of a single, readable, Kratos-based composition model.

## Proposed Solution

Standardize the `kg-service` runtime around the official `github.com/go-kratos/kratos/v2`
platform and Clean Architecture boundaries:

1. use `kratos.App` as the application shell and `transport/http` as the HTTP server boundary;
2. wire shared cross-cutting behavior through Kratos middleware, including logging, metadata,
   recovery, tracing, and metrics from the official platform packages;
3. use Kratos logging abstractions with a JSON runtime backend and context enrichment for request,
   worker, startup, and shutdown events;
4. initialize OTEL tracing and metrics from one runtime observability module and connect it to the
   Kratos middleware layer;
5. keep composition, transport, use case, and adapter responsibilities separated so bootstrap code
   stays small, explicit, and easy to test.

## Scope

### In scope

- go-kratos-based application bootstrap and lifecycle management using the official platform packages
- runtime code under `cmd/` and `internal/` for startup, request handling, workers, and
  observability
- targeted runtime packages including `internal/bootstrap`, `internal/config`,
  `internal/telemetry`, `internal/observability`, `internal/write`, and `internal/workers`
- clean architecture boundaries for transport, service/use case, and infrastructure wiring
- Clean Architecture refactoring for runtime composition, dependency direction, and cross-cutting concerns
- structured JSON logging for HTTP, worker, and startup/shutdown events
- request and background trace propagation
- metrics and tracing configuration surface implemented in runtime code

### Out of scope

- rewriting every domain package or business rule
- changing public API semantics unless middleware or observability behavior requires documentation
- choosing a vendor-specific telemetry backend
- replacing repository-owned health, integrity, or admin capabilities
- code under `examples/`
- deployment manifests, deployment docs, and example applications except for minimal notes required
  to name runtime config keys

## Success Criteria

- the service starts and stops through the `go-kratos/kratos` application lifecycle
- the highest-impact runtime refactor work is concentrated in `cmd/serve.go`, `internal/bootstrap`,
  `internal/config`, `internal/telemetry`, `internal/observability`, `internal/write`, and
  `internal/workers`
- request, worker, and startup logs are emitted as structured JSON with consistent core fields
- traces propagate across HTTP handlers and background worker execution where correlation is available
- metrics and tracing use one runtime configuration surface implemented in `cmd/` and `internal/`
- runtime responsibilities are reorganized into clearer Clean Architecture boundaries without changing
  domain behavior
