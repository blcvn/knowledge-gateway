# codegraph-bridge-and-sync-tools

## MODIFIED Requirements

### Requirement: Sync is idempotent by external_ref

Repeated sync runs SHALL use deterministic `external_ref` values so the same symbol is not created multiple times as duplicate logical nodes, while canonical persisted identities remain UUIDs.

#### Scenario: Repeated sync of the same symbol reuses canonical UUID

- GIVEN a symbol maps to a stable `<project_id>:<symbol_id>` external reference
- WHEN the bridge syncs the symbol multiple times
- THEN the sync workflow SHALL treat subsequent writes as updates of the same logical symbol
- AND the persisted canonical node identity SHALL remain the same UUID across runs

#### Scenario: Relationship sync resolves UUID endpoints from external references

- GIVEN a relationship references source and target symbols by stable external references
- WHEN the bridge writes or upserts the relationship
- THEN it SHALL resolve those symbols to canonical UUID node identities before submitting relationship endpoints
