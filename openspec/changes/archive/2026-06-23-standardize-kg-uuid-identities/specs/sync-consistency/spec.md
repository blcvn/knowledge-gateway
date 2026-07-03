## MODIFIED Requirements

### Requirement: Project committed KG changes to graph and vector replicas asynchronously

The KG Service MUST use workers that consume committed outbox events to synchronize graph and vector replicas using canonical UUID identities for service-owned entities.

#### Scenario: Node upsert projects the same canonical UUID to graph and vector stores

- **GIVEN** a node write commits successfully
- **WHEN** the corresponding outbox event is processed
- **THEN** the graph store projection SHALL use the node's canonical UUID
- **AND** the vector store projection SHALL use the same canonical UUID or a deterministic backend-compatible mapping of that UUID

#### Scenario: Worker failure does not require text-id fallback

- **GIVEN** a backend rejects a malformed identity format
- **WHEN** the worker records the projection failure
- **THEN** the service SHALL fix the identity normalization path
- **AND** SHALL NOT widen source-of-truth identity columns from UUID to `TEXT` as the steady-state compatibility mechanism
