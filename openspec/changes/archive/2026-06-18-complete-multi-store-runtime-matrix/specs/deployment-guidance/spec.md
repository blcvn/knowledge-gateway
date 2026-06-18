## MODIFIED Requirements

### Requirement: Publish deployment guidance for supported runtime targets

The KG Service MUST publish operator-facing deployment guidance for Docker Compose, Kubernetes, and VM-based deployments, and that guidance MUST distinguish between memory-backed bootstrap paths and full-flow external-backend runtime profiles.

#### Scenario: Operator selects a full-flow runtime profile

- GIVEN an operator wants to run PostgreSQL together with one supported graph backend and one supported vector backend
- WHEN they open the deployment docs for Compose, Kubernetes, or VM
- THEN they SHALL find a named runtime profile that identifies the required graph adapter, vector adapter, dependencies, and validation steps
- AND the docs SHALL NOT describe a memory-backed deployment as equivalent to a full-flow replica deployment

### Requirement: Provide repeatable deployment entrypoints

The KG Service MUST provide executable scripts or entrypoints that make the supported deployment paths repeatable, including backend-profile selection.

#### Scenario: Compose starts a real graph/vector stack

- GIVEN `KG_RUNTIME_PROFILE=pgvector-memgraph`
- WHEN the operator runs the documented Compose entrypoint
- THEN the stack SHALL start PostgreSQL, `kg-service`, the profile's graph backend, and the profile's vector backend
- AND `kg-service` SHALL be configured to use non-memory graph/vector adapters for that profile

#### Scenario: Post-deploy validation proves the full storage flow

- GIVEN a deployment target has started successfully
- WHEN the operator runs the repository validation entrypoint
- THEN the validation SHALL prove write, projection, read, semantic search, and reconciliation behavior against the selected graph/vector backends
- AND the validation SHALL fail if the runtime falls back to memory adapters instead of the selected profile
