## MODIFIED Requirements

### Requirement: Publish deployment guidance for supported runtime targets

The KG Service MUST publish operator-facing deployment guidance for Docker Compose, Kubernetes, and VM-based deployments.

#### Scenario: Compose deployment path is documented

- **GIVEN** an operator wants the simplest supported deployment path
- **WHEN** they open the deployment docs
- **THEN** they can find a Docker Compose deployment flow
- **AND** the flow explains how to start the service and verify that it is healthy

#### Scenario: Compose CodeGraph validation stack is documented

- **GIVEN** an operator wants to run local CodeGraph validation against `kg-service`
- **WHEN** they open the Compose deployment guidance
- **THEN** they can find a dedicated stack using Postgres, Memgraph, and Qdrant
- **AND** the guidance states that Postgres remains `pgvector`-compatible for migrations even when `VECTOR_ADAPTER=qdrant`
- **AND** the guidance identifies `KG_RUNTIME_PROFILE=qdrant-memgraph` as the expected runtime selection

#### Scenario: Operators can discover the single-script CodeGraph flow

- **GIVEN** an operator wants one repeatable entrypoint for local CodeGraph setup and validation
- **WHEN** they open the Compose deployment guidance
- **THEN** they can find one repository-owned script that covers stack startup, bootstrap, sync, and verification
- **AND** the guidance explains which steps can be skipped or reused after the first successful run

### Requirement: Publish operator-facing configuration inventory

The KG Service MUST publish an operator-facing inventory of supported environment variables for deployment and runtime startup.

#### Scenario: Operators can discover the environment contract in one place

- **GIVEN** an operator needs to configure `kg-service` for Compose, Kubernetes, or VM deployment
- **WHEN** they open the deployment guidance
- **THEN** they can find one repository-owned inventory of supported environment variables
- **AND** the inventory explains defaults, conditional requirements, and deployment notes for each variable

#### Scenario: HTTP embedding requirements for CodeGraph validation are explicit

- **GIVEN** an operator configures the Compose stack for CodeGraph validation
- **WHEN** they review the environment inventory
- **THEN** the inventory identifies `EMBEDDING_PROVIDER=http` and the required companion variables `EMBEDDING_URL`, `EMBEDDING_MODEL`, and `EMBEDDING_API_KEY`
- **AND** the inventory references `tests/llm/embedding-vnp.txt` as the local test source for endpoint/model values
- **AND** the documented example uses placeholders or redacted values instead of committing live secrets
