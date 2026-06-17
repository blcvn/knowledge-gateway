## ADDED Requirements

### Requirement: Use a consistent success response envelope

The KG Service MUST use a documented and consistent success response shape for REST endpoints unless a route explicitly uses `204 No Content`.

#### Scenario: Object-returning endpoint uses a single-resource envelope

- **GIVEN** a REST endpoint returns one logical resource
- **WHEN** the request succeeds
- **THEN** the service returns `200 OK`, `201 Created`, or `202 Accepted` as appropriate
- **AND** the response body includes a top-level resource payload shaped according to the endpoint contract

#### Scenario: List-returning endpoint uses a collection envelope

- **GIVEN** a REST endpoint returns multiple resources
- **WHEN** the request succeeds
- **THEN** the response body includes a collection field such as `items` or `results`
- **AND** any pagination metadata defined by the API contract is included alongside the collection

#### Scenario: Delete endpoint may omit body only when contract allows it

- **GIVEN** an endpoint performs a delete or revoke action
- **WHEN** the endpoint is documented to return `204 No Content`
- **THEN** the response body is omitted
- **AND** otherwise the service returns a confirmation payload with status metadata

### Requirement: Use a consistent error response envelope

The KG Service MUST return a consistent structured error body for 4xx and 5xx responses.

#### Scenario: Error response includes stable machine-readable fields

- **GIVEN** a request fails
- **WHEN** the service returns an error response
- **THEN** the response body includes a stable error code and human-readable message
- **AND** may include structured `details` for field-level or rule-level diagnostics

#### Scenario: Sensitive internals are not exposed in error bodies

- **GIVEN** a backend failure, dependency outage, or internal exception occurs
- **WHEN** the service returns a 5xx error
- **THEN** the error body does not expose secrets, raw SQL, raw graph queries, stack traces, or internal infrastructure identifiers

### Requirement: Distinguish request, authorization, validation, and existence failures consistently

The KG Service MUST use a stable status-code strategy so clients can distinguish malformed requests, authorization failures, validation failures, and missing resources.

#### Scenario: Authentication failure returns unauthorized

- **GIVEN** the caller is unauthenticated or presents invalid credentials
- **WHEN** the service rejects the request
- **THEN** it returns `401 Unauthorized`

#### Scenario: Authorization failure returns forbidden

- **GIVEN** the caller is authenticated but lacks permission for the requested tenant, app, domain, or action
- **WHEN** the request is rejected
- **THEN** the service returns `403 Forbidden`

#### Scenario: Validation failure returns unprocessable entity

- **GIVEN** the request payload is syntactically acceptable but violates schema, rule, or business validation
- **WHEN** the service rejects the request
- **THEN** it returns `422 Unprocessable Entity`

#### Scenario: Missing resource returns not found

- **GIVEN** the caller references a resource identifier that does not exist in visible scope
- **WHEN** the service rejects the request
- **THEN** it returns `404 Not Found`

#### Scenario: Malformed request returns bad request

- **GIVEN** the caller submits a malformed request shape, unsupported query parameter form, or invalid route-level argument
- **WHEN** the service cannot parse or route the request meaningfully
- **THEN** it returns `400 Bad Request`

### Requirement: Standardize asynchronous write acknowledgment

The KG Service MUST document whether write endpoints return synchronous completion or asynchronous acceptance and MUST use a consistent acknowledgment payload.

#### Scenario: Asynchronous write returns accepted status

- **GIVEN** a write is committed to PostgreSQL and queued for downstream projection
- **WHEN** the service chooses asynchronous acknowledgment semantics
- **THEN** it returns `202 Accepted`
- **AND** the response includes the resource identifier and processing status

#### Scenario: Synchronous create returns created status

- **GIVEN** an endpoint is defined to complete creation synchronously
- **WHEN** the create request succeeds
- **THEN** the service returns `201 Created`
- **AND** the response includes the created resource representation or identifier

### Requirement: Standardize list filtering and pagination semantics

The KG Service MUST use consistent query parameter behavior for list and search-adjacent administrative endpoints.

#### Scenario: List endpoint returns pagination metadata when paginated

- **GIVEN** a list endpoint supports pagination
- **WHEN** the caller supplies page or cursor parameters
- **THEN** the response includes the current slice of `items`
- **AND** includes the next-page or cursor metadata defined by the API contract

#### Scenario: Unsupported filter parameter returns bad request

- **GIVEN** a caller supplies an unsupported filter or malformed query parameter
- **WHEN** the endpoint validates request parameters
- **THEN** the service returns `400 Bad Request`
- **AND** identifies the unsupported or malformed parameter in the error details

### Requirement: Preserve parity between REST and MCP authorization semantics

The KG Service MUST ensure MCP operations map to the same authorization and visibility semantics as equivalent REST endpoints.

#### Scenario: REST and MCP return equivalent authorization outcomes

- **GIVEN** a caller invokes equivalent KG capabilities through REST and MCP
- **WHEN** the caller has or lacks access for the target operation
- **THEN** both surfaces allow or deny access consistently for the same identity and scope

#### Scenario: MCP validation failure returns documented tool error semantics

- **GIVEN** an MCP caller submits invalid tool arguments
- **WHEN** the service rejects the invocation
- **THEN** it returns the documented MCP validation error shape aligned with the underlying API contract
