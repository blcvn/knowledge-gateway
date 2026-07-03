## ADDED Requirements

### Requirement: Standardize runtime lifecycle on a go-kratos application boundary

The KG Service MUST start, run, and stop through a `go-kratos/kratos` application boundary in
`cmd/` and `internal/` that owns transport startup, dependency wiring, and graceful shutdown.

#### Scenario: Service startup is managed through one application shell

- **GIVEN** the service process is launched through the repository-owned serve entrypoint
- **WHEN** runtime dependencies are initialized
- **THEN** a `kratos.App` instance from the official Kratos platform manages server startup,
  lifecycle hooks, and shutdown coordination
- **AND** transport serving is not orchestrated through ad hoc goroutines and signal handling spread across multiple packages

#### Scenario: Runtime standardization excludes example code

- **GIVEN** the runtime architecture is being standardized
- **WHEN** implementation work is planned for this change
- **THEN** the primary write scope is limited to `cmd/` and `internal/`
- **AND** code under `examples/` is not required to be migrated as part of the same change

#### Scenario: Runtime standardization targets the current composition and worker packages first

- **GIVEN** the service runtime is being refactored toward Clean Architecture and go-kratos-managed boundaries
- **WHEN** implementation scope is selected inside `internal/`
- **THEN** the primary targets include `internal/bootstrap`, `internal/config`, `internal/telemetry`,
  `internal/observability`, `internal/write`, and `internal/workers`
- **AND** the change prioritizes composition, lifecycle, middleware, and observability seams before
  broader domain-package refactors

#### Scenario: Background workers participate in managed lifecycle hooks

- **GIVEN** the runtime starts projection or cleanup workers
- **WHEN** the application enters running or stopping states
- **THEN** those workers start and stop through managed lifecycle hooks
- **AND** shutdown ordering allows in-flight work to exit cleanly within configured timeouts

### Requirement: Preserve clean architecture boundaries in runtime wiring

The KG Service MUST organize runtime wiring around explicit transport, application/use case, and
infrastructure responsibilities.

#### Scenario: Cross-cutting concerns stay in the runtime boundary

- **GIVEN** the service applies logging, tracing, recovery, or request metadata propagation
- **WHEN** handlers and business services are wired
- **THEN** those concerns are applied from Kratos middleware or providers from the official platform
- **AND** domain-facing services are not required to construct transport-specific cross-cutting behavior themselves

#### Scenario: Infrastructure adapters are wired behind application-facing services

- **GIVEN** the runtime depends on Postgres, Redis, graph, vector, or FTS adapters
- **WHEN** the application is assembled
- **THEN** infrastructure clients are constructed in the infrastructure/runtime layer
- **AND** transport handlers interact through application-facing services rather than directly owning infrastructure setup

#### Scenario: Bootstrap remains a composition root, not a god package

- **GIVEN** runtime wiring is being assembled
- **WHEN** the composition root is implemented
- **THEN** bootstrap owns orchestration only
- **AND** transport, use case, and infrastructure helpers keep single responsibilities and minimal coupling

#### Scenario: Official Kratos packages remain the source of truth

- **GIVEN** the runtime needs logging, middleware, server, lifecycle, or tracing behavior
- **WHEN** the implementation chooses supporting packages
- **THEN** the service uses official `github.com/go-kratos/kratos/v2` packages wherever they cover the need
- **AND** temporary wrapper packages that only mimic Kratos APIs are removed once migration is complete
