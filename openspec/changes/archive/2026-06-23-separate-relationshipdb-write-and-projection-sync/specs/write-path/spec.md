## MODIFIED Requirements

### Requirement: Write through PostgreSQL as system of record

The KG Service MUST treat `relationshipdb`/PostgreSQL as the only synchronous write target for application
requests, and MUST persist node and relationship mutations there before any projection to graph or vector
replicas is attempted.

#### Scenario: Application write commits only to relationshipdb before async projection

- **GIVEN** a caller is authorized to create or update a node or relationship
- **WHEN** the write request succeeds
- **THEN** the request SHALL commit the source mutation to `relationshipdb`
- **AND** any downstream `graphdb` or `vectordb` synchronization SHALL happen only after that commit through the async projection pipeline

#### Scenario: Request path does not call graphdb or vectordb directly

- **GIVEN** a valid write request reaches the service
- **WHEN** the request is handled on the synchronous application path
- **THEN** the handler SHALL NOT depend on direct `graphdb` writes
- **AND** the handler SHALL NOT depend on direct `vectordb` writes

### Requirement: Couple writes and outbox events atomically

The KG Service MUST create outbox events or an equivalent durable async sync trigger in the same transaction
as the corresponding `relationshipdb` mutation.

#### Scenario: Successful write durably hands off async projection work

- **GIVEN** a valid write request
- **WHEN** the `relationshipdb` transaction commits
- **THEN** the persisted mutation SHALL have a corresponding pending outbox event or durable async trigger
- **AND** that handoff SHALL be sufficient for a background worker to continue projection later

### Requirement: Return consistent write API success and failure responses

The KG Service MUST return write responses based on source-commit outcome, not on downstream projection
completion.

#### Scenario: Successful write returns before graphdb and vectordb catch up

- **GIVEN** a valid authorized write succeeds in `relationshipdb`
- **WHEN** async projection to `graphdb` or `vectordb` has not completed yet
- **THEN** the service SHALL still return success or accepted according to implementation policy
- **AND** the response contract SHALL only guarantee durable source persistence plus async handoff

#### Scenario: Projection lag does not rewrite a successful write response

- **GIVEN** a write request already committed successfully to `relationshipdb`
- **AND** downstream projection is delayed by worker backlog
- **WHEN** the original request completes
- **THEN** the request SHALL NOT be turned into a source-write failure because of that projection lag

### Requirement: Preserve graph identity boundaries on source writes

The KG Service MUST persist or deterministically derive a graph identity / graph scope discriminator for
source writes whenever the domain can host multiple logical knowledge graphs that would otherwise collide.

#### Scenario: Write path derives stable graph identity for a scoped knowledge graph

- **GIVEN** a domain supports multiple logical graphs within the same service deployment
- **WHEN** a caller writes a node or relationship
- **THEN** the service SHALL persist or derive a stable graph identity / graph scope discriminator for that mutation
- **AND** repeated writes for the same logical graph SHALL reuse that same discriminator

#### Scenario: Codegraph writes distinguish project or repository scope

- **GIVEN** the source data comes from a code graph integration such as `codegraph`
- **WHEN** two different projects or repositories contain overlapping symbol names or shapes
- **THEN** the write path SHALL distinguish them by graph identity or equivalent scope metadata
- **AND** the service SHALL NOT merge them into one logical knowledge graph solely because other fields happen to match

#### Scenario: Relationship endpoints resolve within the intended graph scope

- **GIVEN** a relationship write references two endpoint nodes
- **WHEN** the service validates the relationship before persistence
- **THEN** it SHALL confirm the endpoints belong to the same intended graph identity unless the domain explicitly allows cross-graph relationships
