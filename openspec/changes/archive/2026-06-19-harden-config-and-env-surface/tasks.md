# Tasks

## Milestone: `runtime-config-hardening`

- [x] Update environment parsing so invalid integer and duration values return configuration errors instead of panicking.
- [x] Extend config validation or load-time checks so error messages identify the offending environment variable clearly.
- [x] Add or update tests for malformed environment values and conditional backend requirements.

## Milestone: `health-sanitization`

- [x] Remove raw Postgres DSN values and any equivalent secrets from the public `/healthz` payload.
- [x] Keep health output useful for probes and operator debugging with non-sensitive metadata only.
- [x] Add or update tests covering the sanitized health response.

## Milestone: `deployment-contract-alignment`

- [x] Update Kubernetes deployment assets to pass `KG_GRAPH_DATABASE` when profile defaults or operators provide it.
- [x] Review Compose, Kubernetes, and VM deployment entrypoints for consistency with the runtime env contract.
- [x] Fix deployment docs that currently describe defaults or examples that do not match script behavior.

## Milestone: `docs-env-inventory`

- [x] Publish a single operator-facing environment variable inventory with defaults, conditions, and deployment notes.
- [x] Link that inventory from the main deployment docs and any relevant operator guides.
