## ADDED Requirements

### Requirement: Write through PostgreSQL as system of record

The KG Service MUST persist node and relationship mutations to PostgreSQL before projecting data to graph or vector replicas.

#### Scenario: Node write commits before replica sync

- **GIVEN** a caller is authorized to create a node
- **WHEN** the write request succeeds
- **THEN** the node is committed to PostgreSQL
- **AND** the service returns success without waiting for graph/vector sync completion

#### Scenario: Relationship write follows the same source-of-truth path

- **GIVEN** a caller is authorized to create a relationship
- **WHEN** the request succeeds
- **THEN** the relationship is committed to PostgreSQL first
- **AND** later projection is driven from the outbox

#### Scenario: `POST /v1/kg/write/nodes` accepts a valid node write

- **GIVEN** the caller has write permission for the target domain and the payload satisfies schema rules
- **WHEN** the caller submits `POST /v1/kg/write/nodes`
- **THEN** the service persists the node in PostgreSQL
- **AND** returns an accepted or success response containing the node identity

#### Scenario: `POST /v1/kg/write/relationships` rejects invalid endpoints

- **GIVEN** the caller submits a relationship whose endpoints or type do not match registered schema constraints
- **WHEN** the caller submits `POST /v1/kg/write/relationships`
- **THEN** the service rejects the request before persistence

### Requirement: Couple writes and outbox events atomically

The KG Service MUST create outbox events in the same transaction as the corresponding KG mutation.

#### Scenario: Successful write creates one or more outbox events

- **GIVEN** a valid write request
- **WHEN** the database transaction commits
- **THEN** the persisted mutation has corresponding pending outbox event records

#### Scenario: Failed transaction emits no projection event

- **GIVEN** a write fails validation or transaction commit
- **WHEN** the mutation is not persisted
- **THEN** no outbox event is left behind for that failed write

### Requirement: Record ownership, visibility, and version metadata on writes

The KG Service MUST attach domain, ownership, visibility, and domain-version metadata to persisted KG records.

#### Scenario: Node write stores ownership metadata

- **GIVEN** a node is created by an authenticated app
- **WHEN** the node is persisted
- **THEN** the record stores owner tenant and owner app identifiers
- **AND** the record stores the target domain identifier

#### Scenario: Domain version is snapshotted at write time

- **GIVEN** a domain has a current ontology version
- **WHEN** a node is written in that domain
- **THEN** the persisted node stores the active domain version used during validation

#### Scenario: `PUT /v1/kg/write/nodes/{id}` preserves ownership while updating properties

- **GIVEN** a node already exists with stored owner metadata
- **WHEN** an authorized caller updates it through `PUT /v1/kg/write/nodes/{id}`
- **THEN** the service updates mutable properties
- **AND** preserves immutable ownership and domain identity metadata unless an explicit supported change is allowed

### Requirement: Support idempotent cross-store identifiers

The KG Service MUST support a stable external reference for nodes when provided by domain conventions so downstream sync can upsert deterministically.

#### Scenario: Node with external reference uses stable projection identity

- **GIVEN** a write includes a valid domain-defined external reference
- **WHEN** the service persists and later projects the node
- **THEN** downstream graph/vector sync can use that reference as the stable upsert identity

#### Scenario: Missing external reference falls back to internal identity

- **GIVEN** a node write omits an external reference
- **WHEN** the service projects the node
- **THEN** it still projects successfully using the internal node identifier

#### Scenario: `DELETE /v1/kg/write/nodes/{id}` marks a node as deleted for downstream projection

- **GIVEN** a node exists in PostgreSQL and has been projected to graph and vector stores
- **WHEN** an authorized caller submits `DELETE /v1/kg/write/nodes/{id}`
- **THEN** the source-of-truth record is marked deleted according to service semantics
- **AND** a projection event is emitted so replicas stop returning that node

### Requirement: Return consistent write API success and failure responses

The KG Service MUST return predictable success payloads and typed validation/authorization errors for write endpoints.

#### Scenario: `POST /v1/kg/write/nodes` returns accepted node identity

- **GIVEN** a valid authorized node write succeeds
- **WHEN** `POST /v1/kg/write/nodes` completes
- **THEN** the service returns `202 Accepted` or `201 Created` according to implementation policy
- **AND** the response includes the node identifier and asynchronous processing status when applicable

#### Scenario: Schema or domain validation failure returns unprocessable entity

- **GIVEN** a write payload violates node, relationship, or bridge validation rules
- **WHEN** the caller invokes a write endpoint
- **THEN** the service returns `422 Unprocessable Entity`
- **AND** the response includes enough detail for the caller to identify the failing fields or rules

#### Scenario: Unauthorized write returns forbidden

- **GIVEN** the caller lacks write permission for the target domain or owner scope
- **WHEN** it submits `POST /v1/kg/write/nodes`, `PUT /v1/kg/write/nodes/{id}`, or `POST /v1/kg/write/relationships`
- **THEN** the service returns `403 Forbidden`

#### Scenario: Unknown node on update or delete returns not found

- **GIVEN** the caller references a non-existent node identifier
- **WHEN** it invokes `PUT /v1/kg/write/nodes/{id}` or `DELETE /v1/kg/write/nodes/{id}`
- **THEN** the service returns `404 Not Found`
