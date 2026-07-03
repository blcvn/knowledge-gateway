## MODIFIED Requirements

### Requirement: Publish operational metrics for service objectives

The KG Service MUST expose metrics needed to validate latency, sync lag, revoke propagation, and
reconciliation objectives through a standardized observability runtime.

#### Scenario: Metrics remain available after runtime standardization

- **GIVEN** the service runtime is migrated to go-kratos-managed observability components from the official platform
- **WHEN** operators inspect service metrics
- **THEN** they can still observe latency, sync lag, backlog, and failure indicators
- **AND** the runtime does not silently drop previously documented service-level metrics during the migration

#### Scenario: Trace context correlates operational signals

- **GIVEN** a request or managed background workflow emits logs, metrics, and spans
- **WHEN** observability data is inspected
- **THEN** trace or correlation identifiers are included where available so operators can join related signals

### Requirement: Emit structured operational logs for runtime and protected traffic

The KG Service MUST emit structured JSON logs for startup, shutdown, HTTP traffic, and managed worker activity.

#### Scenario: HTTP request logs use structured JSON fields

- **GIVEN** an authenticated or unauthenticated HTTP request is handled by the service
- **WHEN** the request completes
- **THEN** the service emits a JSON log entry with route, method, status, latency, and correlation fields
- **AND** tenant or app identity is included only when that metadata is available through the request context

#### Scenario: Worker logs are structured and correlated

- **GIVEN** projection or cleanup workers process background work
- **WHEN** they emit progress or failure logs
- **THEN** the logs are emitted as JSON records with component-specific fields
- **AND** the records use the same base runtime schema as request and startup logs

### Requirement: Support trace propagation for service and worker execution

The KG Service MUST support trace propagation across request handling and managed worker execution where correlation is available.

#### Scenario: Incoming trace headers flow into request-scoped observability

- **GIVEN** an HTTP client sends supported trace propagation headers
- **WHEN** the request is handled by the service
- **THEN** request-scoped logs and spans preserve the propagated trace context

#### Scenario: Worker activity records correlation when spawned from traced work

- **GIVEN** a background workflow processes work derived from a traced request or managed job context
- **WHEN** the runtime emits logs or spans for that workflow
- **THEN** correlation identifiers are preserved where the runtime can safely carry them forward

#### Scenario: Observability standardization is limited to service runtime code

- **GIVEN** the service observability stack is being standardized on the official go-kratos runtime patterns
- **WHEN** implementation scope is evaluated
- **THEN** the primary code changes are limited to `cmd/` and `internal/`
- **AND** example applications or bridges under `examples/` are excluded from this change

#### Scenario: Observability refactor starts from the current runtime observability packages

- **GIVEN** observability code is being standardized toward Clean Architecture and go-kratos-managed runtime
  concerns
- **WHEN** the implementation is planned in `internal/`
- **THEN** the primary observability targets include `internal/telemetry`, `internal/observability`,
  `internal/bootstrap`, `internal/write`, and `internal/workers`
- **AND** the refactor focuses on shared logging, tracing propagation, and lifecycle-aware signal
  emission before expanding to unrelated packages

#### Scenario: No custom observability facade survives the migration

- **GIVEN** the service has adopted the official Kratos observability and middleware stack
- **WHEN** the runtime composition is finalized
- **THEN** temporary logger or middleware shim packages that only exist to imitate Kratos are removed
- **AND** the remaining implementation depends on official Kratos packages or clearly named domain/runtime packages with single responsibilities
