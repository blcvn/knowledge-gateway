# sync-consistency

## MODIFIED Requirements

### Requirement: Outbox-driven projection runtime batches and coalesces entity work

The system SHALL claim outbox events in pages, coalesce repeated mutations for the same entity within a
claimed page, and project the resulting entity work units instead of naively projecting every raw event.

#### Scenario: Multiple node updates collapse to one projection in the same claimed page

- GIVEN outbox page `P` contains three `NODE_UPSERTED` events for the same node `node-123`
- AND their source versions are `5`, `6`, and `7`
- WHEN the worker builds projection work for page `P`
- THEN it SHALL project only one node work unit for `node-123`
- AND that work unit SHALL use source version `7`
- AND earlier events in the same page SHALL NOT trigger separate graph/vector writes

#### Scenario: Delete wins over earlier upsert in the same claimed page

- GIVEN outbox page `P` contains `NODE_UPSERTED(node-123, version=7)` and later `NODE_DELETED(node-123, version=8)`
- WHEN the worker coalesces page `P`
- THEN the final work unit for `node-123` SHALL be delete
- AND graph/vector/FTS projection SHALL execute delete semantics idempotently

### Requirement: Projection concurrency SHALL NOT depend on a global event-processing lock

The system SHALL avoid holding a runtime-global mutex across source reads, graph writes, embedding calls,
vector writes, or FTS indexing.

#### Scenario: Two unrelated nodes project concurrently

- GIVEN the worker has claimed two node work units for `node-A` and `node-B`
- AND both require graph and vector projection
- WHEN the projection stage runs
- THEN processing `node-A` SHALL NOT require holding a global lock for the duration of processing `node-B`
- AND any shared lock usage SHALL be limited to short critical sections for shared in-memory state or ledger bookkeeping

### Requirement: Graph and vector projection progress SHALL be tracked independently

The system SHALL preserve partial backend progress so one replica can advance without falsely marking the
other replica as synced.

#### Scenario: Graph succeeds while vector fails

- GIVEN a node work unit with source version `11`
- AND graph projection succeeds
- AND embedding or vector upsert fails
- WHEN the worker commits projection results
- THEN `GraphVersion` SHALL advance to `11`
- AND `VectorVersion` SHALL remain below `11`
- AND reconciliation SHALL classify only the vector replica as lagging or stuck

### Requirement: Stale events SHALL be treated as successful no-op projections

The system SHALL reject stale projection writes that would move a replica version backwards, while treating
that outcome as idempotent success rather than retryable failure.

#### Scenario: Older event arrives after newer version already synced

- GIVEN graph replica for `node-123` already stores `_kg_sync_version=15`
- AND the worker later processes an older outbox event for `node-123` with source version `14`
- WHEN the worker evaluates the projection write
- THEN it SHALL skip the stale graph write
- AND it SHALL NOT lower the replica version from `15` to `14`
- AND it SHALL treat the stale write as a successful no-op for retry purposes
