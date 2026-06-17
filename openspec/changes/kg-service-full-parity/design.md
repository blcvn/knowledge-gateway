# Design

## Current Behavior

The service already exposes the intended HTTP and MCP surface, and much of the domain logic is in place. However, the current runtime still relies on in-memory projection stores and simplified helpers for parts of the backend:

- read template execution iterates over the write projection store
- semantic search scores content in-process instead of calling a real embedding/vector pipeline
- RagSearch delegates directly to semantic search
- the session manager only records intended `SET LOCAL` statements
- reconciliation currently compares in-memory graph/vector mirrors against the source store

## Problem Statement

The TDD expects a production architecture with clear backend boundaries:

- PostgreSQL as source of truth
- graph and vector replicas as downstream projections
- transaction-scoped write sessions
- distinct semantic search and RAG retrieval paths
- reconciliation over the real projection boundary

Without a parity-focused follow-up, the repository can remain functionally green while still being architecturally incomplete.

## Goals

- Preserve the external API and domain semantics already implemented.
- Replace remaining in-memory/backend shortcuts with production adapters.
- Keep security and ACL behavior identical across REST, MCP, workers, and audits.
- Make search, read, and sync behavior reflect the actual deployment architecture.

## Non-Goals

- Redesign the public API surface.
- Revisit the bootstrap/legal seed ontology behavior unless required by backend parity.
- Add new product capabilities beyond the TDD target architecture.

## Key Decisions

### 1. Use adapter boundaries for backends

The change should keep service logic stable while swapping the backing stores and transport adapters:

- write-path repositories own Postgres persistence and outbox writes
- read-path owns graph query execution
- search-path owns vector embedding, indexing, and retrieval
- workers own sync fanout and reconciliation over those adapters

### 2. Keep ACL enforcement at the service layer and in the backend adapters

ACL predicates already exist in service code. The parity work must preserve them and ensure graph/vector adapters receive the same visibility constraints so the behavior is consistent across interfaces.

### 3. Make RAG distinct from semantic search

RAG should retrieve context from the vector layer and then produce an answer/retrieval payload from that context. It must not remain a direct alias of semantic search.

### 4. Make session handling real

The current Postgres session manager is useful for tests, but parity requires a real transaction-scoped session implementation so write mutations run with actual `SET LOCAL` semantics.

### 5. Keep worker processing idempotent

Outbox event handling must be safe for retries and replay. Graph/vector updates and ACL fanout should converge to the same state when reprocessed.

## Risks And Mitigations

- Backend adapter work can widen scope.
  - Mitigation: keep the API contracts stable and migrate one boundary at a time.
- Real embedding/vector integration may need external dependencies.
  - Mitigation: define a provider interface and default implementation that can be configured in deployment.
- Graph and write-path parity can regress latency.
  - Mitigation: gate the final merge on benchmark and reconciliation checks.

## Validation Strategy

- Run integration tests that prove backend wiring, not just in-memory projections.
- Verify write-path transactions set tenant/app session context against the real session manager.
- Verify RAG uses a retrieval path distinct from semantic search.
- Verify reconciliation reports drift against real projection state.
- Run performance and security validations before archiving the change.
