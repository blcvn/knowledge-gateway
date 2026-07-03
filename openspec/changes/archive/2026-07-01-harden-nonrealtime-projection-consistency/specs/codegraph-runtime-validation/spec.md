## MODIFIED Requirements

### Requirement: Provide a repeatable CodeGraph runtime validation flow

The KG Service MUST provide a repository-owned validation flow for running CodeGraph against a local `kg-service` Compose stack.

#### Scenario: Operator can run one script for the full CodeGraph lifecycle

- **GIVEN** the operator has Docker Compose available and the required embedding variables set
- **WHEN** they run the documented CodeGraph script entrypoint
- **THEN** the repository boots `kg-service` with Postgres, Memgraph, and Qdrant dependencies
- **AND** the same flow bootstraps or reuses the tenant, app, domain, and ontology required for `code-graph`
- **AND** the same flow proceeds to KG upsert and query validation without requiring ad hoc manual commands

#### Scenario: Validation flow fails fast on missing embedding configuration

- **GIVEN** CodeGraph validation uses `EMBEDDING_PROVIDER=http`
- **WHEN** `EMBEDDING_URL`, `EMBEDDING_MODEL`, or `EMBEDDING_API_KEY` is missing
- **THEN** the validation flow stops before running semantic verification
- **AND** it reports which required input is absent

#### Scenario: Rerun can skip completed initialization safely

- **GIVEN** the operator has already run the CodeGraph script once successfully
- **WHEN** they rerun the same repository-owned flow
- **THEN** the flow can skip or reuse completed initialization steps such as Compose startup, tenant/app creation, or ontology bootstrap
- **AND** it does not require the operator to wipe the local stack only to refresh data or rerun verification

#### Scenario: Validation logs CodeGraph skip when CLI is unavailable

- **GIVEN** the current environment does not have the local `codegraph` CLI available
- **WHEN** the validation flow reaches the CodeGraph create/update verification block
- **THEN** it SHALL log that CodeGraph create/update validation is being skipped
- **AND** it SHALL continue with the remaining projection consistency checks

### Requirement: Validation covers ontology bootstrap and CodeGraph queryability

The CodeGraph runtime validation flow MUST cover both ontology readiness and query execution against the running stack.

#### Scenario: Validation bootstraps or verifies the `code-graph` ontology before sync

- **GIVEN** the Compose stack is healthy
- **WHEN** the repository-owned validation flow runs
- **THEN** it bootstraps or verifies the `code-graph` ontology domain before CodeGraph sync/test steps proceed
- **AND** it verifies that bootstrap artifacts are queryable through repository-owned checks before data sync starts

#### Scenario: Validation upserts CodeGraph data into KG backends

- **GIVEN** the `code-graph` ontology is ready and the local CodeGraph index is available
- **WHEN** the repository-owned validation flow performs the data update step
- **THEN** it upserts CodeGraph data into the `kg-service` runtime instead of assuming insert-only semantics
- **AND** the refreshed data is available for both graph traversal and vector-backed search checks

#### Scenario: Validation confirms semantic or template-backed queries

- **GIVEN** CodeGraph symbols have been synced into domain `code-graph`
- **WHEN** the validation flow runs its final verification
- **THEN** it executes repository-owned checks that cover get or list behavior, semantic search, and at least one template-backed query against the running service
- **AND** it confirms the stack returns a successful result for synced CodeGraph data

#### Scenario: Validation verifies relationshipdb write and projection sync timing

- **GIVEN** the validation flow writes a fresh probe node through the normal write API
- **WHEN** it polls realtime reads, non-realtime reads, and per-entity sync status for that node
- **THEN** it SHALL confirm the source write is immediately visible through the relationshipdb-backed realtime path
- **AND** it SHALL wait for graph-head advancement, the non-realtime graph projection path, and
  other projection backends to converge within the documented timeout

#### Scenario: Validation treats in-flight graph versions as non-synced

- **GIVEN** the validation flow is probing a freshly written node whose logical graph version is
  still being projected
- **WHEN** per-entity sync status is queried before the graph backend head advances
- **THEN** the validation flow SHALL expect a non-synced graph status
- **AND** it SHALL NOT treat that intermediate state as a success condition

#### Scenario: Validation fails on false-SYNCED graph projection status

- **GIVEN** the validation flow is probing a freshly written node
- **AND** per-entity sync status reports `graph_lag_class="SYNCED"`
- **WHEN** the non-realtime graph read still cannot return that node
- **THEN** the validation flow SHALL fail explicitly
- **AND** it SHALL report the last observed sync-status payload and non-realtime graph-read result
