## ADDED Requirements

### Requirement: Project committed KG changes to graph and vector replicas asynchronously

The KG Service MUST use workers that consume committed outbox events to synchronize graph and vector replicas.

#### Scenario: Node upsert is projected to graph and vector stores

- **GIVEN** a node write commits successfully
- **WHEN** the corresponding outbox event is processed
- **THEN** the graph store is updated with the node projection
- **AND** the vector store is updated with the node embedding and payload

#### Scenario: Worker retries failed projection

- **GIVEN** projection to a replica store fails transiently
- **WHEN** the worker handles the event
- **THEN** the event remains retryable according to the configured retry policy

### Requirement: Synchronize ACL changes across caches and replicas

The KG Service MUST propagate access-grant changes to Redis caches, graph ACL fields, and vector ACL payloads.

#### Scenario: Grant creation expands visibility after propagation

- **GIVEN** a new access grant becomes active
- **WHEN** ACL synchronization finishes
- **THEN** the grantee can read or search the newly shared scope

#### Scenario: Grant revocation is enforced quickly

- **GIVEN** an active grant is revoked
- **WHEN** cache invalidation and replica updates complete
- **THEN** the grantee loses access within the service's revoke propagation target

#### Scenario: `DELETE /v1/access/grants/{id}` triggers immediate cache invalidation

- **GIVEN** an active grant contributes to the caller's visible ACL set
- **WHEN** an authorized admin revokes it through `DELETE /v1/access/grants/{id}`
- **THEN** related Redis ACL cache entries are invalidated immediately
- **AND** downstream graph/vector ACL refresh is scheduled or triggered

### Requirement: Cascade lifecycle state through configured sync rules

The KG Service MUST support status cascade updates driven by domain configuration during replica synchronization.

#### Scenario: Status change triggers configured cascade

- **GIVEN** a domain declares cascade rules in its status configuration
- **WHEN** a source node's lifecycle status changes
- **THEN** the worker applies the configured cascade updates to related projected nodes

#### Scenario: Domain without cascade rules does not attempt status propagation

- **GIVEN** a domain has no cascade rules configured
- **WHEN** one of its nodes changes status
- **THEN** the worker does not execute cascade propagation for that domain

### Requirement: Detect and report replica drift

The KG Service MUST provide reconciliation that compares source-of-truth data with graph and vector projections and reports drift.

#### Scenario: Reconciliation reports mismatched projections

- **GIVEN** PostgreSQL and a replica store diverge for some records
- **WHEN** reconciliation runs
- **THEN** it reports the mismatched records and drift metrics

#### Scenario: Healthy reconciliation stays within target drift

- **GIVEN** the synchronization pipeline is operating correctly
- **WHEN** reconciliation runs on schedule
- **THEN** reported drift remains within the configured service threshold

#### Scenario: `GET /v1/kg/integrity/tenant/{tenant_id}` returns tenant-scoped projection health

- **GIVEN** reconciliation or integrity data is available for a tenant
- **WHEN** an authorized caller requests `GET /v1/kg/integrity/tenant/{tenant_id}`
- **THEN** the service returns tenant-scoped integrity information derived from source and replica state

### Requirement: Return consistent integrity API and worker failure semantics

The KG Service MUST expose integrity results and worker failures through predictable operational responses.

#### Scenario: `GET /v1/kg/integrity/tenant/{tenant_id}` returns integrity summary payload

- **GIVEN** integrity information exists for the requested tenant
- **WHEN** the caller invokes `GET /v1/kg/integrity/tenant/{tenant_id}`
- **THEN** the service returns `200 OK`
- **AND** the response includes summary health, drift counters, and any relevant scoped details defined by the API contract

#### Scenario: `GET /v1/kg/integrity/missing-bridges?tenant_id=...` returns bridge gaps

- **GIVEN** bridge integrity analysis has identified missing required links
- **WHEN** the caller invokes `GET /v1/kg/integrity/missing-bridges?tenant_id=...`
- **THEN** the service returns `200 OK`
- **AND** the response includes the affected domains, node identifiers, or aggregate counts according to the contract

#### Scenario: Worker projection failure is observable without corrupting source-of-truth

- **GIVEN** a graph or vector projection fails after PostgreSQL commit
- **WHEN** the worker records the failure
- **THEN** the source-of-truth mutation remains committed
- **AND** the failed projection is surfaced through retryable worker state, metrics, or integrity reporting
