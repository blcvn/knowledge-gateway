## ADDED Requirements

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
