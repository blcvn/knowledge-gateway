# Tasks

## Milestone: `internal/platform`

- [x] Replace the placeholder Postgres session manager with a real transaction-scoped session implementation.
- [x] Add real repository/adapters for source-of-truth write operations and outbox persistence boundaries.
- [x] Add a production vector adapter abstraction for embedding storage and retrieval.
- [x] Add a production graph adapter abstraction for template execution and projection reads.

## Milestone: `internal/write`

- [x] Wire write mutations to the production session manager and repository adapters.
- [x] Keep external-ref, bridge creation, soft delete, and ACL behavior identical after adapter migration.
- [x] Verify write authorization still honors grant-based and tenant-owned visibility with real backend wiring.

## Milestone: `internal/read`

- [x] Execute query templates through the graph adapter instead of iterating over the projection store directly.
- [x] Preserve ACL injection, hop filtering, lifecycle filtering, timeout handling, and row caps in graph execution.
- [x] Add integration coverage for graph-backed execution on a non-trivial fixture.

## Milestone: `internal/search`

- [x] Replace hash-based embedding generation with a pluggable embedding provider.
- [x] Persist and query searchable vectors through a vector adapter.
- [x] Implement RAG retrieval as a distinct pipeline from semantic search.
- [x] Preserve ACL, deletion, domain, lifecycle, and authority-score semantics after adapter migration.

## Milestone: `internal/workers`

- [x] Consume outbox events against production repositories and make graph/vector sync idempotent.
- [x] Preserve grant-change ACL propagation, revoke removal, and status cascade behavior across retries.
- [x] Move reconciliation to compare the authoritative store against the production projection adapters.

## Milestone: `internal/http` And `internal/mcp`

- [x] Keep REST and MCP responses aligned while swapping backend implementations.
- [x] Verify tool/result parity for search, read, ontology, access, and integrity surfaces against production adapters.

## Milestone: `tests/integration`

- [x] Add end-to-end tests proving Postgres session scope, graph-backed template execution, vector-backed search, and RAG retrieval.
- [x] Add end-to-end tests proving outbox replay, ACL revoke convergence, and reconciliation drift detection against production adapters.
- [x] Run benchmark validation for read, search, and write-to-sync latency against the TDD objectives.
- [x] Re-run security validation after backend adapter migration.
