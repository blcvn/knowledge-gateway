# sync-consistency

## MODIFIED Requirements

### Requirement: Idempotent outbox-driven replica sync

The system SHALL consume outbox events and apply graph/vector updates idempotently against the configured production adapters, while persisting a durable projection version record for each entity.

#### Scenario: Same source entity is compared across three stores

- GIVEN a node or relationship has been committed in PostgreSQL
- WHEN the worker projects it to graph and vector stores
- THEN PostgreSQL SHALL retain the authoritative source version for that entity
- AND graph and vector replicas SHALL each store the applied `_kg_sync_version`
- AND reconciliation SHALL be able to determine whether the three stores are aligned, stale, or partially applied

### Requirement: Reconciliation uses replica versions, not only payload equality

The system SHALL detect stale replicas by version metadata even when older payloads happen to look superficially valid.

#### Scenario: Graph replica is one event behind

- GIVEN PostgreSQL has source version `42` for a node
- AND the graph replica still stores `_kg_sync_version=41`
- WHEN reconciliation runs
- THEN it SHALL report a stale graph projection for that node
- EVEN IF the graph payload still matches older source fields closely enough to pass a shallow payload comparison

### Requirement: Repair flows preserve version monotonicity

Projection replay and targeted repair flows SHALL advance replica versions monotonically for each entity.

#### Scenario: Rebuild vector replica for a drifted node

- GIVEN a node's vector projection is missing or stale
- WHEN operators trigger replay or targeted repair
- THEN the recreated vector document SHALL be written with the current authoritative sync version
- AND reconciliation SHALL stop reporting drift once the vector replica version matches the source version
