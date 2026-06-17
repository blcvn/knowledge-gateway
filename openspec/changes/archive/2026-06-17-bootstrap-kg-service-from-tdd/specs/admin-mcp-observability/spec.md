## ADDED Requirements

### Requirement: Expose administrative APIs for tenancy, grants, ontology, and integrity checks

The KG Service MUST provide administrative API surfaces needed to manage tenants, apps, grants, ontology, and integrity operations described in the TDD.

#### Scenario: Tenant admin can manage apps and grants

- **GIVEN** an authorized admin client calls the admin APIs
- **WHEN** it performs create, list, rotate-key, or revoke actions for apps and grants
- **THEN** the service executes the requested administrative mutation or retrieval with audit coverage

#### Scenario: Integrity endpoint returns projection health information

- **GIVEN** integrity data has been collected or computed
- **WHEN** an authorized caller invokes an integrity endpoint
- **THEN** the service returns health or drift information for the requested tenant or scope

#### Scenario: `POST /v1/access/grants` rejects unsafe cross-tenant write grants without expiry

- **GIVEN** a caller attempts to create a cross-tenant grant with `write` or `admin` permission and no `expires_at`
- **WHEN** the request is submitted to `POST /v1/access/grants`
- **THEN** the service rejects the request as invalid

#### Scenario: `GET /v1/access/audit?resource_owner_tenant_id=...` returns audit history

- **GIVEN** audit records exist for a tenant's protected resources
- **WHEN** an authorized admin requests `GET /v1/access/audit?resource_owner_tenant_id=...`
- **THEN** the service returns audit records for that tenant subject to access policy

### Requirement: Expose KG capabilities through MCP

The KG Service MUST expose an MCP server surface that allows agent clients to discover and invoke supported KG operations safely.

#### Scenario: Agent lists available templates through MCP

- **GIVEN** an authenticated MCP client is connected
- **WHEN** it invokes the template discovery capability
- **THEN** it receives only the templates visible to that client's effective ontology and permissions

#### Scenario: MCP tool invocation respects the same ACL rules as REST

- **GIVEN** a caller can access KG data through REST only within a limited scope
- **WHEN** the same caller invokes the equivalent MCP capability
- **THEN** the returned data is constrained by the same identity and ACL rules

#### Scenario: MCP template execution matches REST behavior

- **GIVEN** a caller can execute a registered template through the REST read API
- **WHEN** the caller invokes the equivalent MCP tool
- **THEN** the result shape and ACL enforcement are equivalent to the REST path for the same identity

### Requirement: Emit audit records for sensitive access decisions

The KG Service MUST audit allow and deny outcomes for protected read, search, write, and grant-management operations.

#### Scenario: Allowed read is audited

- **GIVEN** a protected graph read succeeds
- **WHEN** the service returns the result
- **THEN** it records an audit entry describing the requester, action, scope, and allow outcome

#### Scenario: Denied cross-tenant action is audited

- **GIVEN** a requester attempts an unauthorized cross-tenant action
- **WHEN** the service rejects the request
- **THEN** it records an audit entry with a deny outcome and reason

#### Scenario: `POST /v1/access/grants` success is audited

- **GIVEN** an authorized admin creates a valid grant
- **WHEN** the request to `POST /v1/access/grants` succeeds
- **THEN** the service records an audit entry for grant creation with requester and scope details

### Requirement: Publish operational metrics for service objectives

The KG Service MUST expose metrics needed to validate latency, sync lag, revoke propagation, and reconciliation objectives.

#### Scenario: Read and search latency are measurable by operation

- **GIVEN** the service handles read and search traffic
- **WHEN** metrics are emitted
- **THEN** latency can be measured by route or template category

#### Scenario: Sync health metrics expose projection lag

- **GIVEN** outbox workers are processing events
- **WHEN** operators inspect metrics
- **THEN** they can observe backlog size, processing lag, and failure counts

### Requirement: Return consistent admin and MCP responses

The KG Service MUST document and return predictable success payloads and authorization or validation errors for admin and MCP-facing operations.

#### Scenario: `POST /v1/access/grants` returns created grant metadata

- **GIVEN** an authorized caller submits a valid grant payload
- **WHEN** `POST /v1/access/grants` succeeds
- **THEN** the service returns `201 Created`
- **AND** the response includes the grant identifier, scope, permission, status, and expiry metadata

#### Scenario: `DELETE /v1/access/grants/{id}` returns revoke confirmation

- **GIVEN** an active grant exists and the caller is allowed to revoke it
- **WHEN** `DELETE /v1/access/grants/{id}` succeeds
- **THEN** the service returns `200 OK` or `204 No Content` according to the API contract

#### Scenario: Unauthorized audit or integrity access returns forbidden

- **GIVEN** a caller lacks admin scope for the requested tenant or owner
- **WHEN** it invokes audit or integrity endpoints
- **THEN** the service returns `403 Forbidden`

#### Scenario: MCP authentication failure returns connection or invocation error

- **GIVEN** an MCP client fails authentication or presents an invalid token
- **WHEN** it attempts to connect or invoke a tool
- **THEN** the service returns the documented MCP authentication error without exposing sensitive internals
