## ADDED Requirements

### Requirement: Execute graph reads only through registered templates

The KG Service MUST expose graph reads only through registered query templates referenced by domain and template name.

#### Scenario: Active template runs through generic route

- **GIVEN** a domain has an active registered template
- **WHEN** a caller invokes `/v1/kg/read/template/{domain_id}/{template_name}`
- **THEN** the service loads that template definition
- **AND** executes the corresponding graph traversal

#### Scenario: Draft or unknown template is rejected

- **GIVEN** a template is missing or not active
- **WHEN** the caller invokes the generic read route
- **THEN** the service rejects the request

#### Scenario: `GET /v1/kg/read/templates?domain_id=...` lists only executable templates

- **GIVEN** a domain has templates in mixed states
- **WHEN** the caller requests `GET /v1/kg/read/templates?domain_id=...`
- **THEN** the response lists only the templates visible and allowed for that caller according to service policy

### Requirement: Inject ACL filtering at every traversal step

The KG Service MUST enforce requester visibility on the start node and on each traversed hop generated from a query template.

#### Scenario: Start node is filtered by ACL

- **GIVEN** a template start match points to a node the caller cannot see
- **WHEN** the template executes
- **THEN** the node is not matched
- **AND** the response contains no data derived from that inaccessible start node

#### Scenario: Intermediate hop is filtered by ACL

- **GIVEN** a traversal hop would cross into a node outside the caller's ACL
- **WHEN** the template executes
- **THEN** that hop is excluded from the result set

### Requirement: Bind template parameters safely

The KG Service MUST bind client-supplied read parameters through the template parameter schema rather than string-concatenating executable query text.

#### Scenario: Missing required parameter is rejected

- **GIVEN** a template declares a required parameter
- **WHEN** the caller omits it
- **THEN** the service rejects the request before graph execution

#### Scenario: Parameter type mismatch is rejected

- **GIVEN** a template declares a parameter type
- **WHEN** the caller supplies an incompatible value
- **THEN** the service rejects the request as invalid

#### Scenario: `POST /v1/kg/read/template/{domain_id}/{template_name}` binds parameters without raw query interpolation

- **GIVEN** a caller submits normal template parameters including special characters
- **WHEN** the request is executed through `POST /v1/kg/read/template/{domain_id}/{template_name}`
- **THEN** the service binds those values through the template parameter model
- **AND** does not interpret them as executable graph query text

### Requirement: Apply lifecycle filtering only when configured

The KG Service MUST apply template-level lifecycle filtering only when the target domain has status configuration and the template requests that behavior.

#### Scenario: Valid-only lifecycle filter is enforced for configured domain

- **GIVEN** a domain has status configuration and a template hop requests `valid_only`
- **WHEN** the template executes
- **THEN** nodes outside the domain's valid status set are excluded from that hop

#### Scenario: Lifecycle filter becomes no-op without domain config

- **GIVEN** a template includes lifecycle filtering but the domain has no status configuration
- **WHEN** the template executes
- **THEN** the service does not fail
- **AND** it treats lifecycle filtering as a no-op

### Requirement: Bound graph query execution

The KG Service MUST enforce server-side execution limits for graph reads.

#### Scenario: Query timeout is enforced

- **GIVEN** a template execution exceeds the configured graph timeout
- **WHEN** the graph database does not return in time
- **THEN** the service aborts the request and returns a timeout error

#### Scenario: Maximum row count is enforced

- **GIVEN** a template could return more records than the configured maximum
- **WHEN** the service executes the query
- **THEN** the returned result set is limited to the configured maximum bound

#### Scenario: `GET /v1/kg/read/nodes/{id}` respects caller visibility

- **GIVEN** a node exists in graph projections but is outside the caller's ACL
- **WHEN** the caller requests `GET /v1/kg/read/nodes/{id}`
- **THEN** the service does not return the node to that caller

### Requirement: Return consistent read API payloads and execution errors

The KG Service MUST return predictable result envelopes and bounded execution errors for graph-read endpoints.

#### Scenario: `POST /v1/kg/read/template/{domain_id}/{template_name}` returns result records

- **GIVEN** a registered active template executes successfully
- **WHEN** the caller invokes `POST /v1/kg/read/template/{domain_id}/{template_name}`
- **THEN** the service returns `200 OK`
- **AND** the response includes a result collection shaped by the template return fields

#### Scenario: Unknown or inactive template returns bad request or not found

- **GIVEN** the caller references a template that is unknown or not active
- **WHEN** it invokes the read-template endpoint
- **THEN** the service returns `400 Bad Request` or `404 Not Found` according to the endpoint contract

#### Scenario: Invalid read parameters return validation errors

- **GIVEN** the caller omits required params or supplies values of the wrong type
- **WHEN** it invokes a template read
- **THEN** the service returns `422 Unprocessable Entity`

#### Scenario: Graph timeout returns gateway or timeout error

- **GIVEN** graph execution exceeds the configured service timeout
- **WHEN** the request cannot complete in time
- **THEN** the service returns a timeout-class error such as `504 Gateway Timeout` or an equivalent documented timeout response
