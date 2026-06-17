# KG Service Full Parity

## Why

The bootstrap change translated the TDD into a working service, but the current implementation still contains parity gaps relative to the target design:

- read and search execute against in-memory projection stores instead of production graph/vector adapters
- embedding generation is deterministic hashing rather than a pluggable embedding provider
- `RagSearch` currently aliases semantic search instead of running a distinct retrieval-augmented pipeline
- the Postgres session manager records `SET LOCAL` statements instead of managing a real transaction-scoped session
- worker reconciliation and ACL fanout are implemented in-process but still need final production wiring and SLA validation

This change closes those gaps so the codebase matches the intended platform architecture, not just the test fixtures.

## What Changes

- Replace remaining in-memory backend behavior with production-ready adapters and wiring.
- Make write-path transaction handling use real Postgres session boundaries.
- Make read-path template execution run through the selected graph backend.
- Make semantic search use a real embedding provider and vector adapter.
- Make RagSearch a distinct retrieval pipeline that uses vector-retrieved context.
- Finalize sync workers and reconciliation to operate over production storage boundaries.

## Capabilities

- Transaction-scoped write sessions backed by Postgres
- Graph-backed template execution
- Vector-backed semantic search and RAG retrieval
- Idempotent outbox-driven sync to graph/vector replicas
- Reconciliation and drift reporting over source-of-truth versus projections

## Impact

- Aligns runtime behavior with the target TDD architecture.
- Preserves the existing HTTP and MCP contracts while changing backend implementation details.
- Reduces the risk that the service passes tests but diverges from intended production behavior.
