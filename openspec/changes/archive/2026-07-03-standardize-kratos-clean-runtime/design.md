# Design: Standardize KG Service Runtime on the go-kratos Platform with Clean Architecture

## Overview

This change replaces the current custom runtime shell with the official `github.com/go-kratos/kratos/v2`
runtime model. The goal is not a product rewrite. The goal is to make `cmd/` and `internal/`
follow one explicit composition style with clear ownership, small packages, and official Kratos
middleware for logging, metadata propagation, tracing, recovery, and metrics.

## Current Behavior

Today the runtime is assembled through a custom bootstrap layer that:

- opens infrastructure clients directly during app construction;
- registers HTTP routes on `http.ServeMux`;
- starts background workers with ad hoc goroutines;
- logs startup and worker activity through package-local helpers;
- records metrics in a local in-memory telemetry registry;
- applies write-request tracing separately from broader HTTP middleware concerns.

The service works, but the runtime shape is not aligned with the actual Kratos app/server boundary
and it spreads cross-cutting concerns across multiple packages.

## Scope Boundary

This change is limited to `kg-service` runtime code, primarily:

- `cmd/`
- `internal/`

The primary target packages are:

- `cmd` with emphasis on `cmd/serve.go`
- `internal/bootstrap`
- `internal/config`
- `internal/telemetry`
- `internal/observability`
- `internal/write` with emphasis on request logging/tracing middleware
- `internal/workers`

The change explicitly excludes:

- `examples/`
- standalone example wiring or bridge code
- broad deployment/documentation rewrites outside the minimum needed to name runtime config

## Goals

- standardize application lifecycle, dependency wiring, and graceful shutdown on `kratos.App`
- define cleaner clean-code boundaries between transport, application/use case, and adapters
- reduce ad hoc bootstrap logic and package-local lifecycle ownership
- use Kratos middleware for logging, metadata propagation, recovery, tracing, and metrics
- emit structured JSON logs consistently across HTTP, workers, startup, and shutdown events
- propagate trace context through request handling and background processing
- align metrics, tracing, and operator config under one runtime contract

## Non-Goals

- migrating every feature to generated protobuf transport in one change
- changing business semantics for access, ontology, read, write, search, or integrity flows
- introducing a mandatory external collector for local development
- removing repository-specific operational endpoints that are already part of the service contract
- refactoring `examples/` to match the new runtime shape
- broad changes in `deploy/` or docs as a primary deliverable of this change

## Proposed Architecture

### 1. Kratos application boundary

Build the runtime around `kratos.App` and `transport/http.NewServer`.

The app constructor should own:

- config loading and validation handoff
- logger construction
- OTEL provider and propagator setup
- HTTP server construction
- background worker lifecycle hooks
- graceful shutdown ordering

The `cmd/serve` entrypoint should call into this boundary rather than managing an `http.Server` and
OS signals directly.

The primary implementation surface for this part is:

- `cmd/serve.go`
- `internal/bootstrap/app.go`
- supporting wiring files under `internal/bootstrap/`

### 2. Kratos middleware and observability

Use the official Kratos middleware stack where it fits the HTTP boundary:

- `middleware/logging`
- `middleware/metadata`
- `middleware/recovery`
- `middleware/tracing`
- `middleware/metrics`

The runtime should configure these once in the composition root and let handlers remain focused on
request handling. Logging should flow through Kratos logging interfaces, with a JSON backend adapter
kept behind the runtime boundary so the rest of the service does not care about the concrete logger.
No custom transport or logger facade should replace the Kratos packages unless the change explicitly
documents why the official package cannot express the needed behavior.
Temporary shim packages that only mirror Kratos APIs should be treated as migration scaffolding and
removed before the change is considered complete.

Tracing and metrics should be initialized from one observability module that configures the OTEL
provider, propagator, and sampling options, then wires Kratos middleware to that shared runtime
state.

### 3. Clean architecture package boundaries

The runtime should distinguish three responsibility layers:

- transport: HTTP handlers, middleware, request metadata extraction, error translation
- application/use case: orchestration services and domain-facing workflows
- infrastructure/adapters: Postgres, Redis, graph, vector, FTS, telemetry exporters, and external
  clients

This change does not require every package to be renamed at once, but it does require the new
bootstrap path in `cmd/` and `internal/` to encode these boundaries clearly so future refactors have
a stable target.

Clean code expectations for the targeted runtime packages:

- `internal/bootstrap` should be the composition root, not a place where transport, infra setup, and
  long-running process control are blended together
- `internal/config` should remain the single runtime config boundary for Kratos and observability
  settings
- `internal/write` should not own the only structured request tracing pattern if that concern belongs
  to shared runtime middleware
- `internal/workers` should participate in lifecycle management through explicit runtime hooks
  rather than detached goroutine ownership
- `internal/telemetry` and `internal/observability` should expose clearer seams between metric
  recording, aggregation, and transport-facing reporting

### 4. Structured JSON logging

All runtime logs should flow through one logger abstraction backed by Kratos logging contracts.

Required log behavior:

- JSON output by default in all non-test runtime paths
- shared base fields such as `ts`, `level`, `service`, `version`, `trace_id`, `span_id`,
  `component`
- request logs with method, route, status, latency, tenant/app identity when available
- worker logs with batch size, processed count, failure count, and correlation identifiers when
  available
- startup and shutdown logs for infrastructure dependencies and lifecycle state changes

Plaintext `log.Printf` output should not be the normal runtime path.

### 5. Unified tracing and telemetry

Tracing and metrics should be initialized from one observability module that can:

- enable or disable trace export by config
- wire OTEL-compatible propagators and tracer providers
- expose metrics via the current service-facing capability or a documented equivalent
- preserve existing domain metrics while moving emission behind a more standard runtime contract

The existing in-memory telemetry registry may remain as an internal aggregation source during the
transition, but the runtime contract should no longer be tied to package-level globals.

### 6. Middleware standardization

The runtime should move cross-cutting HTTP concerns into a consistent middleware chain, including:

- request ID / trace propagation
- structured access logging
- panic recovery
- authentication and identity enrichment
- timeout or cancellation boundaries where appropriate

Write-path-only logging middleware should be folded into the shared runtime approach unless there is a
documented reason to keep write-specific fields separate.

## Configuration Surface

The change should introduce or standardize environment/config entries for:

- log level
- log format with JSON as the supported runtime default
- service name and version fields
- OTEL exporter endpoint and protocol where tracing is enabled
- trace sampling configuration
- metrics enablement and exposure settings if required by the chosen Kratos wiring

If config keys change, the implementation should keep that contract localized to runtime code first.
Any broader deployment or docs updates should be treated as follow-up work, not as part of this
standardization scope.

## Migration Strategy

Implement the runtime standardization in phases:

1. establish the Kratos app shell and logger/telemetry providers;
2. migrate `internal/bootstrap` and `cmd/serve.go` to that shell;
3. standardize shared logging and middleware behavior, including `internal/write/middleware.go`;
4. move `internal/workers` under managed lifecycle hooks;
5. refactor `internal/telemetry`, `internal/observability`, and `internal/config` around the new
   runtime contract;
6. verify the runtime contract inside the targeted packages.

This phased approach reduces risk by preserving existing handlers and core services while changing
the cross-cutting runtime model first.

## Risks And Mitigations

- Refactor scope could expand into a full package rewrite.
  - Mitigation: keep this change focused on runtime boundaries and observability seams, not feature
    redesign.
- Observability changes could break local developer workflows.
  - Mitigation: keep local defaults simple and allow no-op or stdout-based exporters.
- Trace context may not exist for some background jobs.
  - Mitigation: require correlation fields when available and document fallback behavior.
- JSON logs may affect existing log parsing scripts.
  - Mitigation: document the schema in change notes and keep rollout of downstream parsing updates as
    separate follow-up work.

## Validation Strategy

- verify the service still boots and shuts down cleanly through one application boundary
- verify HTTP requests emit structured JSON logs with stable fields
- verify worker paths emit structured logs instead of raw formatted strings
- verify trace headers propagate into request-scoped logs and spans
- verify existing metrics and health capabilities remain available after the runtime refactor
