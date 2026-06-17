# sync-consistency

## ADDED Requirements

### Requirement: Idempotent outbox-driven replica sync
The system SHALL consume outbox events and apply graph/vector updates idempotently against the production projection adapters.

#### Scenario: Reprocess the same outbox event
- WHEN a worker reprocesses an already-seen event
- THEN the resulting graph and vector projections SHALL converge to the same state
- AND duplicate processing SHALL not create duplicate records or duplicate ACL tokens.

### Requirement: Revoke propagation and drift detection
The system SHALL propagate ACL grant changes, including revoke, to graph/vector projections and SHALL report drift against the authoritative store.

#### Scenario: Revoke a grant after projections are warm
- WHEN a grant is revoked after graph/vector projections already contain the grant-derived ACL token
- THEN the worker SHALL remove the token from graph and vector projections
- AND SHALL invalidate the relevant ACL cache entry
- AND reconciliation SHALL report no residual ACL drift once the projection state converges.
