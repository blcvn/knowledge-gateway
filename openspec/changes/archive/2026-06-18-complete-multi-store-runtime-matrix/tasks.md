# Tasks

## Milestone: deployment-profiles

- [x] Add runtime profile definitions for Compose, Kubernetes, and VM deployment paths.
- [x] Replace `memory` adapter defaults in deploy assets with explicit profile-driven graph/vector backend selection.
- [x] Add profile-aware documentation for required environment variables, ports, and dependency services.
- [x] Add a post-deploy validation entrypoint that proves PostgreSQL write, graph projection, vector projection, read, search, and reconciliation.

## Milestone: graph-adapters

- [x] Wire `neo4j` into `internal/bootstrap/wiring.go` as a selectable graph adapter entrypoint.
- [x] Add `memgraph` support by reusing the Cypher-oriented graph adapter path with backend-specific configuration.
- [x] Replace the temporary `neo4j`/`memgraph` wrappers with real client-backed drivers.
- [x] Replace the temporary `nebula` wrapper with a real NebulaGraph client-backed adapter and translation layer.
- [x] Add a NebulaGraph `-tags nebula` smoke test that exercises query translation through a real `nebula-go` client or a client double.
- [x] Add graph-adapter integration coverage that proves `ListNodes`, `ListRelationships`, and sync-version reads against real backends.
- [x] Extend config parsing and validation to accept `memory`, `neo4j`, `memgraph`, and `nebula`.

## Milestone: vector-adapters

- [x] Replace the temporary `qdrant` wrapper with a real Qdrant client-backed adapter that preserves payload filtering and sync-version metadata.
- [x] Replace the temporary `milvus` wrapper with a real Milvus client-backed adapter that maps ANN options and sync-version metadata.
- [x] Extend `PgVectorAdapter` to expose snapshot and sync-version reads for reconciliation.
- [x] Extend config parsing and validation to accept `memory`, `pgvector`, `qdrant`, and `milvus`.
- [x] Add vector-adapter integration coverage that proves ANN, snapshot, and sync-version reads against real backends.

## Milestone: sync-versioning

- [x] Add a durable PostgreSQL projection-version ledger for node and relationship sync state.
- [x] Add `_kg_sync_version` metadata to graph node/relationship payloads and vector documents.
- [x] Update worker projection flow so source version, graph version, and vector version advance deterministically after each successful outbox application.
- [x] Update reconciliation to compare replica versions and report stale or partially-applied projections.
- [x] Remove reconciliation fallback to legacy in-memory projection state when a real adapter is configured.

## Milestone: validation-and-ops

- [x] Extend integration and conformance coverage to the supported backend profiles.
- [x] Update deployment and operations docs with backend/profile selection guidance and replica-version triage steps.
- [x] Add runbook steps for repairing graph-only, vector-only, and mixed-version drift between the three stores.
