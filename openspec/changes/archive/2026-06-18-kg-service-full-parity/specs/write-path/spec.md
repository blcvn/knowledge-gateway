# write-path

## ADDED Requirements

### Requirement: Real transaction-scoped write sessions
The system SHALL execute write mutations inside a real transaction-scoped Postgres session that sets tenant and app context with `SET LOCAL`.

#### Scenario: Create a node
- WHEN the service starts a node create mutation
- THEN it SHALL open a transaction-scoped session
- AND SHALL set the tenant/app session variables within that transaction
- AND SHALL persist the source record before emitting sync events.

### Requirement: Source-of-truth persistence precedes replication
The system SHALL persist write-path source data and outbox events in the authoritative store before downstream projections are updated.

#### Scenario: A write mutation succeeds
- WHEN a write mutation completes successfully
- THEN the source-of-truth record SHALL exist in the authoritative store
- AND the corresponding outbox event SHALL be durable
- AND downstream sync workers SHALL be able to replay the event without changing the resulting state.
