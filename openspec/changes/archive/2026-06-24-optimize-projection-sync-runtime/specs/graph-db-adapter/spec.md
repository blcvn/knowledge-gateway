# graph-db-adapter

## MODIFIED Requirements

### Requirement: Graph adapters support bulk projection writes

Production graph adapters SHALL support batch upsert and batch delete operations for nodes and
relationships so the projection runtime can reduce per-entity network round-trips.

#### Scenario: Worker projects a claimed page of node mutations

- GIVEN the worker has coalesced 40 node mutations for graph projection
- WHEN it dispatches them to the configured graph adapter
- THEN the adapter SHALL accept them through a batch write contract rather than requiring 40 isolated API calls

### Requirement: Graph adapters preserve per-entity result visibility in batch mode

Batch graph operations SHALL return enough per-entity success or failure detail for the runtime to update
projection versions and retries accurately.

#### Scenario: One relationship fails inside a graph batch

- GIVEN a relationship batch contains 20 entities
- AND one entity fails due to a backend constraint or transient error
- WHEN the graph adapter returns
- THEN the runtime SHALL be able to identify the failed relationship precisely
- AND unaffected relationships in the same batch SHALL still be eligible to advance their graph sync version

### Requirement: Graph adapters enforce sync-version monotonicity

Graph adapters SHALL preserve `_kg_sync_version` metadata and SHALL NOT apply a stale write that would
move a projected entity backwards in source version.

#### Scenario: Bulk graph upsert receives stale and fresh versions together

- GIVEN a batch contains two updates for distinct entities
- AND one entity carries source version lower than the replica's existing `_kg_sync_version`
- WHEN the graph adapter executes the batch
- THEN that entity's stale update SHALL be skipped or reported as a no-op
- AND the adapter SHALL still apply the fresh update for the other entity
