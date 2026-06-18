## ADDED Requirements

### Requirement: Publish a current API specification for the live HTTP surface

The KG Service MUST publish a versioned API specification that reflects the HTTP routes currently registered by the service runtime.

#### Scenario: Every live route is represented in the API specification

- **GIVEN** the service registers HTTP routes in `internal/bootstrap/app.go`
- **WHEN** the API specification is published or updated
- **THEN** every live route is represented in the spec
- **AND** each route includes its HTTP method and path
- **AND** each route is grouped under its relevant capability area such as health, access, ontology, read, search, write, integrity, or MCP

#### Scenario: The specification documents the current route inventory instead of historical examples

- **GIVEN** historical TDD material may include illustrative payloads or earlier assumptions
- **WHEN** there is a difference between historical examples and the live runtime contract
- **THEN** the API specification follows the live runtime contract
- **AND** historical examples are treated as non-normative context only

### Requirement: Document shared API conventions explicitly

The KG Service MUST document the shared conventions that apply across endpoints so consumers can interpret requests and responses consistently.

#### Scenario: Authentication requirements are documented per current middleware behavior

- **GIVEN** the access middleware protects the `/v1/*` HTTP surface
- **WHEN** the API specification describes authentication
- **THEN** it states that `/healthz` is public
- **AND** it states that the current `/v1/*` routes require `Authorization`
- **AND** it documents the authentication failure response shape and status code

#### Scenario: Common envelopes and pagination are documented once

- **GIVEN** the service uses shared response helpers for errors and list results
- **WHEN** the API specification describes common conventions
- **THEN** it defines the structured error envelope with machine-readable `code` and `message`
- **AND** it defines the collection envelope fields `data`, `next_cursor`, and `has_more`
- **AND** it documents cursor-and-limit pagination defaults and limits for list endpoints that use the shared list envelope

#### Scenario: Request sanitization rules are documented

- **GIVEN** middleware strips caller-supplied identity fields from JSON request bodies
- **WHEN** the API specification describes write and admin endpoints
- **THEN** it states that `tenant_id` and `app_id` are derived from caller identity rather than trusted from request bodies

### Requirement: Document endpoint-level request and response schemas

The KG Service MUST document the current request parameters, request bodies, success responses, and known error statuses for each live endpoint.

#### Scenario: Endpoint documentation covers path, query, body, and response fields

- **GIVEN** an endpoint accepts path parameters, query parameters, or a JSON body
- **WHEN** that endpoint appears in the API specification
- **THEN** the specification describes those inputs explicitly
- **AND** the specification describes the current success status code and response schema
- **AND** the specification describes the documented error statuses returned by that endpoint family

#### Scenario: Dynamic template execution route is documented as a generic capability

- **GIVEN** the service exposes `POST /v1/kg/read/template/{domain_id}/{template_name}`
- **WHEN** the API specification documents read endpoints
- **THEN** it describes the route as a generic template execution capability
- **AND** it explains that template availability depends on active domain query templates rather than hard-coded router entries

#### Scenario: Asynchronous write acknowledgments are documented

- **GIVEN** some write-path endpoints acknowledge accepted work before downstream projection completes
- **WHEN** the API specification documents write and ingest endpoints
- **THEN** it identifies which endpoints return `202 Accepted`
- **AND** it documents the acknowledgment payload fields such as resource or job identifier and processing status

### Requirement: Keep the API specification synchronized with implementation changes

The KG Service MUST update the published API specification whenever runtime API behavior changes.

#### Scenario: Route changes update the published spec in the same workstream

- **GIVEN** a change adds, removes, or renames an HTTP route
- **WHEN** that change is prepared for review or merge
- **THEN** the published API specification is updated in the same workstream
- **AND** the route inventory remains aligned with the runtime router

#### Scenario: Shared envelope changes update the documented components

- **GIVEN** shared error or list envelope behavior changes in `internal/httpapi/respond`
- **WHEN** the implementation change is prepared
- **THEN** the API specification updates the shared documented components in the same workstream
- **AND** endpoint references continue to point to the current shared behavior
