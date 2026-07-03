# Tasks

## Milestone: `kratos-runtime-shell`

- [x] Replace the custom bootstrap shell with `kratos.App` in `cmd/serve.go` and `internal/bootstrap` for startup, shutdown, HTTP serving, and background worker lifecycle.
- [x] Build the HTTP boundary with `transport/http.NewServer` and keep transport wiring in the composition root instead of ad hoc `http.ServeMux` ownership.
- [x] Preserve existing service capabilities while moving signal handling and graceful shutdown into the Kratos application lifecycle.

## Milestone: `kratos-middleware-stack`

- [x] Replace ad hoc runtime middleware with Kratos middleware for logging, metadata propagation, recovery, tracing, and metrics from `github.com/go-kratos/kratos/v2`.
- [x] Standardize request, worker, startup, and shutdown log fields so operators can correlate events consistently.
- [x] Refactor `internal/write/middleware.go` so write-path logging either joins the shared Kratos middleware stack or is explicitly justified as a specialized exception.
- [x] Remove any temporary custom runtime logger or middleware facades once the Kratos middleware path is in place.

## Milestone: `tracing-and-telemetry`

- [x] Introduce a Kratos-compatible observability module across `internal/telemetry`, `internal/observability`, and `internal/bootstrap` for tracing, metrics, and propagation.
- [x] Standardize trace context propagation across HTTP handlers and managed background workers, with `internal/workers` participating in lifecycle-managed correlation.
- [x] Prove worker span/link export across the async outbox boundary with targeted unit coverage and end-to-end request-meta parity verification.
- [x] Keep current service metrics available while reducing runtime dependence on scattered package-level observability patterns in `internal/telemetry` where practical.

## Milestone: `runtime-config-contract`

- [x] Add or update runtime config in `internal/config` and runtime wiring in `cmd/serve.go` or its replacement for log level, JSON log format, service metadata, OTEL endpoint, and sampling controls.
- [x] Keep the scope limited to `kg-service` code and do not refactor `examples/` as part of this change.
- [x] Add verification coverage for structured logs, trace propagation, and lifecycle-managed worker startup/shutdown in the targeted runtime packages.

## Milestone: `clean-code-boundaries`

- [x] Refactor runtime composition so bootstrap is the composition root and transport, application/use case, and infrastructure responsibilities stay separated.
- [x] Remove or isolate package-level runtime state where a constructor- or lifecycle-owned dependency is clearer.
- [x] Prefer small focused helpers over multi-purpose runtime functions when the change can reduce coupling without changing behavior.
- [x] Retire custom runtime shim packages that only exist to mimic Kratos APIs if the official Kratos packages can express the same behavior.
