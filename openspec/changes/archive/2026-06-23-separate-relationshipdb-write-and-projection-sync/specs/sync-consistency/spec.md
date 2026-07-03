## MODIFIED Requirements

### Requirement: Project committed KG changes to graph and vector replicas asynchronously

The KG Service MUST use background jobs or workers that consume committed source changes from
`relationshipdb` outbox records or an equivalent durable queue to synchronize `graphdb` and `vectordb`
replicas.

#### Scenario: Background worker owns graphdb and vectordb synchronization

- **GIVEN** a node or relationship write has already committed to `relationshipdb`
- **WHEN** downstream projection begins
- **THEN** a background job or worker SHALL load the committed source change
- **AND** that worker SHALL perform the synchronization to `graphdb` and `vectordb`

#### Scenario: Request handler is not the projection owner

- **GIVEN** an application request creates or updates KG source data
- **WHEN** the request handler returns to the caller
- **THEN** the handler SHALL NOT be required to finish `graphdb` synchronization first
- **AND** the handler SHALL NOT be required to finish `vectordb` synchronization first

#### Scenario: Worker retries failed projection without rolling back source commit

- **GIVEN** a source write has committed successfully to `relationshipdb`
- **AND** projection to `graphdb` or `vectordb` fails transiently
- **WHEN** the worker records and retries that failure
- **THEN** the source-of-truth mutation SHALL remain committed
- **AND** projection recovery SHALL remain a responsibility of the async worker pipeline

### Requirement: Return consistent integrity API and worker failure semantics

The KG Service MUST expose projection lag, backlog, and worker failures as operational state of the async
projection plane rather than as retroactive source-write failures.

#### Scenario: Projection backlog is observable without invalidating source writes

- **GIVEN** committed source changes are waiting in the outbox or worker queue
- **WHEN** operators inspect integrity or worker health signals
- **THEN** the service SHALL surface backlog or lag for the async projection plane
- **AND** previously successful source writes SHALL remain valid
